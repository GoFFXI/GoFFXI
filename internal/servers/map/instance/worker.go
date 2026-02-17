package instance

import (
	"context"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/GoFFXI/GoFFXI/internal/config"
	"github.com/GoFFXI/GoFFXI/internal/database"
	mapPackets "github.com/GoFFXI/GoFFXI/internal/packets/map"
	serverPackets "github.com/GoFFXI/GoFFXI/internal/packets/map/server"
)

type InstanceWorker struct {
	natsConn      *nats.Conn
	cfg           *config.Config
	db            *database.DBImpl
	logger        *slog.Logger
	ctx           context.Context
	subscriptions []*nats.Subscription
}

func NewInstanceWorker(ctx context.Context, cfg *config.Config, logger *slog.Logger) (*InstanceWorker, error) {
	var err error

	srv := InstanceWorker{
		cfg:    cfg,
		logger: logger,
		ctx:    ctx,
	}

	// initialize NATS connection
	if err = srv.CreateNATSConnection(); err != nil {
		return nil, fmt.Errorf("could not create NATS connection: %w", err)
	}

	// initialize database connection
	if err = srv.CreateDBConnection(ctx); err != nil {
		return nil, fmt.Errorf("could not create database connection: %w", err)
	}

	return &srv, nil
}

func (s *InstanceWorker) Config() *config.Config {
	return s.cfg
}

func (s *InstanceWorker) Logger() *slog.Logger {
	return s.logger
}

func (s *InstanceWorker) DB() *database.DBImpl {
	return s.db
}

func (s *InstanceWorker) NATS() *nats.Conn {
	return s.natsConn
}

func (s *InstanceWorker) StartProcessingPackets() {
	var subject string

	// create a subscription to the instance subject
	if s.Config().MapInstanceID == 0 {
		subject = "map.instance.*"
	} else {
		subject = fmt.Sprintf("map.instance.%d", s.cfg.MapInstanceID)
	}

	s.Logger().Info("subscribing to NATS subject", "subject", subject)
	newSubscription, err := s.NATS().Subscribe(subject, s.ProcessPacket)
	if err != nil {
		s.Logger().Error("could not subscribe to NATS subject", "subject", subject, "error", err)
		return
	}

	s.subscriptions = append(s.subscriptions, newSubscription)
}

func (s *InstanceWorker) WaitForShutdown(cancelCtx context.CancelFunc, wg *sync.WaitGroup) error {
	// setup signal handling
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	// block until signal received
	sig := <-signalChannel
	s.Logger().Info("shutdown signal received", "signal", sig.String())

	// cancel context to signal all gouroutines to stop
	cancelCtx()

	// wait for all goroutines to finish
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		s.Logger().Info("all goroutines have finished")
		return nil
	case <-time.After(time.Duration(s.Config().ShutdownTimeoutSeconds) * time.Second):
		s.Logger().Warn("shutdown timeout reached, forcing exit")
		return fmt.Errorf("shutdown timeout reached")
	}
}

func (s *InstanceWorker) sendPacket(clientAddr string, packet serverPackets.ServerPacket) error {
	packetData, err := packet.Serialize()
	if err != nil {
		return fmt.Errorf("failed to serialize packet: %w", err)
	}
	if expected := int(packet.Size()); expected > 0 && len(packetData) < expected {
		pad := make([]byte, expected-len(packetData))
		packetData = append(packetData, pad...)
	}
	packetSize := len(packetData)

	s.logPacketDetails(packet, packetData)

	s.Logger().Info("sending packet", "clientAddr", clientAddr, "packetType", packet.Type(), "packetSize", packetSize)

	routedPacket := mapPackets.RoutedPacket{
		ClientAddr: clientAddr,
		Packet: mapPackets.BasicPacket{
			Type: packet.Type(),
			Size: uint16(packetSize),
			Data: packetData,
		},
	}

	subject := fmt.Sprintf("map.router.%s.send", clientAddr)
	return s.NATS().Publish(subject, routedPacket.ToJSON())
}

func (s *InstanceWorker) logPacketDetails(packet serverPackets.ServerPacket, data []byte) {
	switch p := packet.(type) {
	case *serverPackets.CharUpdatePacket:
		logCharUpdateDebug(s.Logger(), p, data)
	}
}

func logCharUpdateDebug(logger *slog.Logger, packet *serverPackets.CharUpdatePacket, raw []byte) {
	if logger == nil {
		return
	}

	sizeUnits := len(raw) / 4
	flags5Composite := uint32(packet.Flags5) |
		(uint32(packet.ModelHitboxSize) << 8) |
		((uint32(packet.Flags6) & 0xFFFF) << 16)
	name := strings.TrimRight(string(packet.Name[:]), "\x00")

	logger.Debug("GP_SERV_CHAR_PC",
		"sizeUnits", sizeUnits,
		"hpp", packet.HPPercent,
		"status", packet.ServerStatus,
		"flags0", fmt.Sprintf("%08X", packet.Flags0),
		"flags1", fmt.Sprintf("%08X", packet.Flags1),
		"flags2", fmt.Sprintf("%08X", packet.Flags2),
		"flags3", fmt.Sprintf("%08X", packet.Flags3),
		"flags4", fmt.Sprintf("%02X", packet.Flags4),
		"flags5", fmt.Sprintf("%04X", flags5Composite),
		"flags6", fmt.Sprintf("%08X", packet.Flags6),
		"grap0", fmt.Sprintf("%04X", packet.GrapIDTbl[0]),
		"name", name,
		"preview", hexPreview(raw, 64),
	)
}

func hexPreview(data []byte, limit int) string {
	if len(data) == 0 {
		return ""
	}
	if limit <= 0 || limit > len(data) {
		limit = len(data)
	}
	preview := hex.EncodeToString(data[:limit])
	if limit < len(data) {
		return preview + "..."
	}
	return preview
}
