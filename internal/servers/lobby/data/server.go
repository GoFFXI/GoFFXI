package data

import (
	"context"
	"errors"
	"io"
	"net"

	"github.com/GoFFXI/GoFFXI/internal/lobby/packets"
	"github.com/GoFFXI/GoFFXI/internal/servers/base/tcp"
)

const (
	CommandRequestKeepXILoaderSpinning = 0xFE
)

type DataServer struct {
	*tcp.TCPServer
}

func (s *DataServer) HandleConnection(ctx context.Context, conn net.Conn) {
	logger := s.Logger().With("client", conn.RemoteAddr().String())
	logger.Info("processing connection")

	// create a new session for this connection
	sessionCtx := sessionContext{
		ctx:    ctx,
		conn:   conn,
		server: s,
		logger: logger,
	}
	defer sessionCtx.Close()

	// buffer for reading data
	buffer := make([]byte, 4096)

	// connection handling loop
	for {
		// make sure we exit if the server is shutting down
		select {
		case <-ctx.Done():
			return
		default:
		}

		// read data from client
		length, err := conn.Read(buffer)
		if err != nil {
			if errors.Is(err, io.EOF) {
				logger.Info("client disconnected")
				break
			} else if errors.Is(err, net.ErrClosed) {
				break
			}

			logger.Error("error reading from connection", "error", err)
			break
		}

		if shouldExit := s.parseIncomingRequest(&sessionCtx, buffer[:length]); shouldExit {
			break
		}
	}
}

func (s *DataServer) parseIncomingRequest(sessionCtx *sessionContext, request []byte) bool {
	header, err := NewRequestHeader(request)
	if err != nil {
		sessionCtx.logger.Error("failed to parse request header", "error", err)
		return true
	}

	// attempt to lookup the account session
	sessionKey := string(header.Identifier[:])
	sessionCtx.logger.Info("looking up session", "sessionKey", sessionKey, "opCode", header.Command)
	accountSession, err := s.DB().GetAccountSessionBySessionKey(sessionCtx.ctx, sessionKey)
	if err != nil {
		// don't treat missing session as an error, just log and continue
		// the 2nd part of selecting a character won't pass in a valid session key for whatever reason
		sessionCtx.logger.Warn("failed to lookup account session", "sessionKey", sessionKey, "error", err)
	}

	// make sure this session context has subscriptions set up
	// this should only be done once per session
	if err = sessionCtx.SetupSubscriptions(sessionKey); err != nil {
		sessionCtx.logger.Error("failed to setup subscriptions", "error", err)
		return true
	}

	switch header.Command {
	case CommandRequestKeepXILoaderSpinning:
		// this is just a keep-alive, respond with empty payload
		_, _ = sessionCtx.conn.Write([]byte{})
	case CommandRequestGetCharacters:
		return s.handleRequestGetCharacters(sessionCtx, &accountSession, request)
	case CommandRequestSelectCharacter:
		return s.handleRequestSelectCharacter(sessionCtx, request)
	}

	return false
}

func (s *DataServer) sendErrorResponse(sessionCtx *sessionContext) {
	response, err := packets.NewResponseError(packets.ErrorCodeUnableToConnectToLobbyServer)
	if err != nil {
		return
	}

	responsePacket, err := response.Serialize()
	if err != nil {
		return
	}

	// it's okay if this write fails, we're already in an error state
	_, _ = sessionCtx.conn.Write(responsePacket)
}
