package router

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/GoFFXI/GoFFXI/internal/database"
	mapPackets "github.com/GoFFXI/GoFFXI/internal/packets/map"
	clientPackets "github.com/GoFFXI/GoFFXI/internal/packets/map/client"
	"github.com/GoFFXI/GoFFXI/internal/servers/base/udp"
	"github.com/GoFFXI/GoFFXI/internal/tools/zlib"
)

type MapRouterServer struct {
	*udp.UDPServer

	sessionsMu    sync.RWMutex
	sessions      map[string]*Session
	packetsMu     sync.Mutex
	packetsToSend map[string][]*mapPackets.RoutedPacket
	codec         *zlib.FFXICodec
}

const (
	// maxPayloadBytes mirrors the guard in LandSandBoat's send_parse:
	// packets larger than ~1400 bytes (payload + 42 byte IP header) are dropped by the client.
	// Using 1300 keeps us safely under that limit while matching the original behaviour.
	maxPayloadBytes      = 1300
	maxDecompressedBytes = 4096
	flushInterval        = 50 * time.Millisecond
)

func NewMapRouterServer(baseServer *udp.UDPServer) *MapRouterServer {
	var codec *zlib.FFXICodec
	if baseServer != nil && baseServer.Config() != nil {
		codec = zlib.NewCodec(baseServer.Config().FFXIResourcePath)
	} else {
		codec = zlib.NewCodec("")
	}

	return &MapRouterServer{
		UDPServer:     baseServer,
		sessions:      make(map[string]*Session),
		packetsToSend: make(map[string][]*mapPackets.RoutedPacket),
		codec:         codec,
	}
}

func (s *MapRouterServer) HandleIncomingPacket(ctx context.Context, length int, data []byte, clientAddr *net.UDPAddr) {
	clientAddrStr := clientAddr.String()
	s.Logger().Debug("handling incoming packet", "dataLength", length, "clientAddr", clientAddrStr)

	if length < mapPackets.HeaderSize+mapPackets.MD5ChecksumSize {
		s.Logger().Warn("packet too small", "clientAddr", clientAddrStr, "length", length)
		return
	}

	innerOffset := int(mapPackets.HeaderSize)
	if len(data) < innerOffset+mapPackets.MD5ChecksumSize {
		s.Logger().Warn("packet too small for inner header", "clientAddr", clientAddrStr, "length", length)
		return
	}

	if len(data) < innerOffset+4 {
		s.Logger().Warn("packet missing inner header", "clientAddr", clientAddrStr, "length", length)
		return
	}

	clientSequence := binary.LittleEndian.Uint16(data[0:2])
	clientAck := binary.LittleEndian.Uint16(data[2:4])

	head := mapPackets.PacketHeader{
		ID:   binary.LittleEndian.Uint16(data[innerOffset : innerOffset+2]),
		Sync: binary.LittleEndian.Uint16(data[innerOffset+2 : innerOffset+4]),
	}

	packetType := head.GetPacketID()
	packetUnits := int(head.GetPacketSize())
	if packetUnits <= 0 {
		s.Logger().Warn("invalid packet size", "clientAddr", clientAddrStr, "type", packetType, "units", packetUnits)
		return
	}

	packetSize := packetUnits * 4
	if packetSize <= 0 {
		s.Logger().Warn("invalid payload size", "clientAddr", clientAddrStr, "type", packetType, "units", packetUnits)
		return
	}

	// make sure the payload fits within the packet
	payloadStart := mapPackets.HeaderSize
	payloadEnd := payloadStart + packetSize
	if payloadEnd > len(data)-mapPackets.MD5ChecksumSize {
		s.Logger().Warn("payload exceeds packet length", "clientAddr", clientAddrStr, "type", packetType, "size", packetSize)
		return
	}

	// attempt to get the session for this client
	session := s.getSession(clientAddrStr)
	if session == nil {
		if packetType != clientPackets.PacketTypeLogin {
			s.Logger().Warn("received packet without session", "clientAddr", clientAddrStr, "packetType", packetType)
			return
		}

		s.Logger().Info("received initial login packet", "clientAddr", clientAddrStr, "length", length, "payloadSize", packetSize, "preview", hexPreview(data, 64))

		loginPacket, err := clientPackets.ParseLoginPacket(data)
		if err != nil {
			s.Logger().Warn("failed to parse login packet", "clientAddr", clientAddrStr, "error", err, "preview", hexPreview(data, 96))
			return
		}

		s.Logger().Info("login packet parsed", "clientAddr", clientAddrStr, "characterID", loginPacket.UniqueNo, "loginCheck", loginPacket.LoginPacketCheck, "sync", head.Sync)

		s.sessionCreated(ctx, clientAddr, loginPacket)
		session = s.getSession(clientAddrStr)
		if session == nil {
			return
		}

		session.lastClientPacketID = clientSequence
		s.updateServerAckFromClient(session, clientAck, clientAddrStr)
		copy(session.lastClientHeader[:], data[:mapPackets.HeaderSize])
		s.Logger().Debug("stored initial client header", "clientAddr", clientAddrStr, "header", hex.EncodeToString(session.lastClientHeader[:mapPackets.HeaderSize]))

		loginPayload := make([]byte, packetSize)
		copy(loginPayload, data[payloadStart:payloadEnd])
		if err := s.forwardPacketToInstance(session, packetType, head.Sync, loginPayload); err != nil {
			s.Logger().Error("failed to forward login packet", "clientAddr", clientAddrStr, "error", err)
		}
		return
	}

	session.lastClientPacketID = clientSequence
	s.updateServerAckFromClient(session, clientAck, clientAddrStr)
	copy(session.lastClientHeader[:], data[:mapPackets.HeaderSize])

	if packetType == clientPackets.PacketTypeLogin {
		return
	}

	if err := s.processEncryptedPacket(session, data[:]); err != nil {
		s.Logger().Warn("failed to process client packet", "clientAddr", clientAddrStr, "packetType", packetType, "error", err)
	}
}

func (s *MapRouterServer) getSession(addr string) *Session {
	s.sessionsMu.RLock()
	defer s.sessionsMu.RUnlock()
	return s.sessions[addr]
}

func (s *MapRouterServer) setSession(addr string, session *Session) {
	s.sessionsMu.Lock()
	defer s.sessionsMu.Unlock()
	s.sessions[addr] = session
}

func (s *MapRouterServer) updateServerAckFromClient(session *Session, clientAck uint16, clientAddr string) {
	prev := session.lastServerPacketID
	if clientAck == prev {
		return
	}

	if clientAck > prev {
		session.lastServerPacketID = clientAck
		return
	}

	s.Logger().Debug("received client ack older than expected", "clientAddr", clientAddr, "ack", clientAck, "expected", prev)
}

func (s *MapRouterServer) sessionCreated(ctx context.Context, clientAddr *net.UDPAddr, loginPacket *clientPackets.LoginPacket) {
	s.Logger().Info("creating session", "clientAddr", clientAddr.String(), "characterID", loginPacket.UniqueNo)

	accountSession, err := s.DB().GetAccountSessionByCharacterID(ctx, loginPacket.UniqueNo)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			s.Logger().Warn("account session not found for character", "characterID", loginPacket.UniqueNo)
		} else {
			s.Logger().Error("failed to lookup account session", "characterID", loginPacket.UniqueNo, "error", err)
		}
		return
	}

	session, err := NewSession(clientAddr, accountSession.SessionKey, s)
	if err != nil {
		s.Logger().Error("failed to create session", "clientAddr", clientAddr.String(), "error", err)
		return
	}

	character, err := s.DB().GetCharacterByID(ctx, loginPacket.UniqueNo)
	if err == nil {
		session.character = &character
	} else if !errors.Is(err, database.ErrNotFound) {
		s.Logger().Warn("failed to load character for session", "characterID", loginPacket.UniqueNo, "error", err)
	}

	s.setSession(clientAddr.String(), session)
}

func (s *MapRouterServer) forwardPacketToInstance(session *Session, packetType uint16, sequence uint16, payload []byte) error {
	routedPacket := mapPackets.RoutedPacket{
		ClientAddr: session.clientAddr.String(),
		Packet: mapPackets.BasicPacket{
			Type:     packetType,
			Size:     uint16(len(payload)),
			Sequence: sequence,
			Data:     payload,
		},
	}

	if session.character != nil {
		routedPacket.CharacterID = session.character.ID
	}

	s.Logger().Debug("forwarding packet to instance", "clientAddr", session.clientAddr.String(), "packetType", packetType, "sequence", sequence, "payloadBytes", len(payload))

	subject := fmt.Sprintf("map.instance.%d", s.Config().MapInstanceID)
	return s.NATS().Publish(subject, routedPacket.ToJSON())
}

func hexPreview(data []byte, limit int) string {
	if limit <= 0 || limit > len(data) {
		limit = len(data)
	}
	preview := hex.EncodeToString(data[:limit])
	if limit < len(data) {
		return preview + "..."
	}
	return preview
}

func (s *MapRouterServer) DeliverPacketsToClients(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.flushPendingPackets()
		}
	}
}

func (s *MapRouterServer) flushPendingPackets() {
	sessions := s.snapshotSessions()
	for _, session := range sessions {
		s.flushSessionQueue(session)
	}
}

func (s *MapRouterServer) snapshotSessions() []*Session {
	s.sessionsMu.RLock()
	defer s.sessionsMu.RUnlock()
	snapshot := make([]*Session, 0, len(s.sessions))
	for _, session := range s.sessions {
		snapshot = append(snapshot, session)
	}
	return snapshot
}

func (s *MapRouterServer) flushSessionQueue(session *Session) {
	addr := session.clientAddr.String()

	for {
		pending := s.dequeuePackets(addr)
		if len(pending) == 0 {
			return
		}

		s.Logger().Debug("flushing pending packets", "clientAddr", addr, "queued", len(pending))

		if err := s.sendPacketBurst(session, pending); err != nil {
			s.Logger().Error("failed to send packet burst", "clientAddr", addr, "error", err)
			s.requeuePackets(addr, pending)
			return
		}
	}
}

func (s *MapRouterServer) dequeuePackets(addr string) []*mapPackets.RoutedPacket {
	s.packetsMu.Lock()
	defer s.packetsMu.Unlock()
	queue := s.packetsToSend[addr]
	if len(queue) == 0 {
		return nil
	}
	s.packetsToSend[addr] = nil
	return queue
}

func (s *MapRouterServer) requeuePackets(addr string, packets []*mapPackets.RoutedPacket) {
	if len(packets) == 0 {
		return
	}

	s.packetsMu.Lock()
	defer s.packetsMu.Unlock()
	if existing := s.packetsToSend[addr]; len(existing) > 0 {
		packets = append(packets, existing...)
	}
	s.packetsToSend[addr] = packets
}

func (s *MapRouterServer) sendPacketBurst(session *Session, pending []*mapPackets.RoutedPacket) error {
	remaining := pending
	for len(remaining) > 0 {
		nextServerPacketID := session.lastServerPacketID + 1
		payload, consumed, err := s.buildPayload(remaining, nextServerPacketID)
		if err != nil {
			return err
		}

		if consumed == 0 {
			return fmt.Errorf("no packets consumed while building payload")
		}

		networkPacket, err := s.buildNetworkPacket(session, payload, nextServerPacketID)
		if err != nil {
			return err
		}

		if _, err := s.Socket().WriteToUDP(networkPacket, session.clientAddr); err != nil {
			return fmt.Errorf("send udp packet: %w", err)
		}

		s.Logger().Debug("sent packet to client", "clientAddr", session.clientAddr.String(), "payloadBytes", len(payload), "udpBytes", len(networkPacket))
		session.lastServerPacketID = nextServerPacketID
		remaining = remaining[consumed:]
	}

	return nil
}

func (s *MapRouterServer) buildPayload(packets []*mapPackets.RoutedPacket, sequence uint16) ([]byte, int, error) {
	buffer := bytes.NewBuffer(make([]byte, 0, maxPayloadBytes))
	consumed := 0

	for consumed < len(packets) {
		packet := packets[consumed]
		required := s.subPacketLength(packet)
		if buffer.Len()+required > maxPayloadBytes {
			if consumed == 0 {
				return nil, 0, fmt.Errorf("packet %d exceeds payload budget", packet.Packet.Type)
			}
			break
		}

		if err := s.writeSubPacket(buffer, packet, sequence); err != nil {
			return nil, consumed, err
		}

		consumed++
	}

	return buffer.Bytes(), consumed, nil
}

func (s *MapRouterServer) writeSubPacket(buf *bytes.Buffer, routed *mapPackets.RoutedPacket, sequence uint16) error {
	payload := routed.Packet.Data
	payloadLen := len(payload)
	alignedPayloadLen := (payloadLen + 3) & ^3

	sizeUnits := uint16((alignedPayloadLen / 4) & 0x7F)
	idField := routed.Packet.Type & 0x1FF
	header := make([]byte, 4)
	binary.LittleEndian.PutUint16(header[0:], idField|(sizeUnits<<9))
	binary.LittleEndian.PutUint16(header[2:], sequence)
	buf.Write(header)
	buf.Write(payload)

	pad := alignedPayloadLen - payloadLen
	if pad > 0 {
		buf.Write(make([]byte, pad))
	}

	return nil
}

func (s *MapRouterServer) subPacketLength(routed *mapPackets.RoutedPacket) int {
	payloadLen := len(routed.Packet.Data)
	alignedPayloadLen := (payloadLen + 3) & ^3
	return 4 + alignedPayloadLen
}

func (s *MapRouterServer) buildNetworkPacket(session *Session, payload []byte, serverPacketID uint16) ([]byte, error) {
	if len(payload) == 0 {
		return nil, fmt.Errorf("empty payload")
	}

	s.Logger().Debug("compressing payload", "clientAddr", session.clientAddr.String(), "payloadBytes", len(payload), "preview", hexPreview(payload, 64))

	compressedBuf := make([]byte, len(payload)*2+64)
	bitCount, err := s.codec.Compress(payload, compressedBuf)
	if err != nil {
		return nil, fmt.Errorf("compress payload: %w", err)
	}

	compressedBytes := int((bitCount + 7) / 8)
	chunk := make([]byte, compressedBytes+4)
	copy(chunk, compressedBuf[:compressedBytes])
	binary.LittleEndian.PutUint32(chunk[compressedBytes:], bitCount)

	hash := md5.Sum(chunk)
	chunk = append(chunk, hash[:]...)

	packet := make([]byte, mapPackets.HeaderSize+len(chunk))
	copy(packet[:mapPackets.HeaderSize], session.lastClientHeader[:])
	binary.LittleEndian.PutUint16(packet[0:2], serverPacketID)
	binary.LittleEndian.PutUint16(packet[2:4], session.lastClientPacketID)
	binary.LittleEndian.PutUint32(packet[8:12], uint32(time.Now().Unix()))
	copy(packet[mapPackets.HeaderSize:], chunk)

	s.Logger().Debug("building network packet", "clientAddr", session.clientAddr.String(), "serverPacketID", serverPacketID, "clientPacketID", session.lastClientPacketID, "bitCount", bitCount, "md5", hex.EncodeToString(hash[:]), "chunkPreview", hexPreview(chunk, 64))
	s.Logger().Debug("udp header preview", "clientAddr", session.clientAddr.String(), "header", hex.EncodeToString(packet[:mapPackets.HeaderSize]), "bodyPreview", hexPreview(packet[mapPackets.HeaderSize:], 64))

	if session.currentBlowfish != nil {
		session.currentBlowfish.EncryptPacket(packet, int(mapPackets.HeaderSize))
	}

	return packet, nil
}

func (s *MapRouterServer) processEncryptedPacket(session *Session, data []byte) error {
	if len(data) < int(mapPackets.HeaderSize)+mapPackets.MD5ChecksumSize+4 {
		return fmt.Errorf("packet too small after header")
	}

	decoded := make([]byte, len(data))
	copy(decoded, data)

	// Keep the latest client header so we can mirror its fields back to the client.
	copy(session.lastClientHeader[:], data[:mapPackets.HeaderSize])
	session.lastClientPacketID = binary.LittleEndian.Uint16(data[0:2])

	if session.currentBlowfish != nil {
		session.currentBlowfish.DecryptPacket(decoded, int(mapPackets.HeaderSize))
	}

	payloadStart := int(mapPackets.HeaderSize)
	payloadEnd := len(decoded) - mapPackets.MD5ChecksumSize
	if payloadEnd <= payloadStart+4 {
		return fmt.Errorf("encrypted payload too small")
	}

	payload := decoded[payloadStart:payloadEnd]
	checksum := decoded[payloadEnd:]
	expected := md5.Sum(payload)
	if !bytes.Equal(expected[:], checksum) {
		return fmt.Errorf("md5 mismatch")
	}

	bitCount := binary.LittleEndian.Uint32(payload[len(payload)-4:])
	compressed := payload[:len(payload)-4]

	decompressed := make([]byte, maxDecompressedBytes)
	written, err := s.codec.Decompress(compressed, bitCount, decompressed)
	if err != nil {
		return fmt.Errorf("decompress: %w", err)
	}

	count := s.dispatchSubPackets(session, decompressed[:written])
	if count > 0 {
		s.Logger().Debug("processed encrypted packet", "clientAddr", session.clientAddr.String(), "subPackets", count, "bytes", written)
	}
	return nil
}

func (s *MapRouterServer) dispatchSubPackets(session *Session, data []byte) int {
	offset := 0
	processed := 0
	for offset+4 <= len(data) {
		header := binary.LittleEndian.Uint16(data[offset:])
		packetType := header & 0x1FF
		sizeUnits := header >> 9
		packetSize := int(sizeUnits) * 2
		if packetSize == 0 || offset+packetSize > len(data) {
			return processed
		}

		sequence := binary.LittleEndian.Uint16(data[offset+2:])
		payloadSize := packetSize - 4
		if payloadSize < 0 {
			return processed
		}

		payload := make([]byte, payloadSize)
		copy(payload, data[offset+4:offset+packetSize])

		s.Logger().Debug("dispatching sub-packet", "clientAddr", session.clientAddr.String(), "packetType", packetType, "sequence", sequence, "payloadBytes", payloadSize)
		if err := s.forwardPacketToInstance(session, packetType, sequence, payload); err != nil {
			s.Logger().Error("failed to forward decompressed packet", "clientAddr", session.clientAddr.String(), "packetType", packetType, "error", err)
		}

		offset += packetSize
		processed++
	}
	return processed
}
