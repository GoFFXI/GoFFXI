package view

import (
	"context"
	"errors"
	"io"
	"net"

	"github.com/GoFFXI/GoFFXI/internal/packets/lobby"
	"github.com/GoFFXI/GoFFXI/internal/servers/base/tcp"
)

type ViewServer struct {
	*tcp.TCPServer
}

func (s *ViewServer) HandleConnection(ctx context.Context, conn net.Conn) {
	logger := s.Logger().With("client", conn.RemoteAddr().String())
	logger.Info("new client connection established")

	// create a new session for this connection
	sessionCtx := sessionContext{
		ctx:    ctx,
		conn:   conn,
		server: s,
		logger: logger,
	}
	defer sessionCtx.Close()

	// connection handling loop
	for {
		// make sure we exit if the server is shutting down
		select {
		case <-ctx.Done():
			return
		default:
		}

		// buffer for reading data
		buffer := make([]byte, 4096)

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

func (s ViewServer) parseIncomingRequest(sessionCtx *sessionContext, request []byte) bool {
	header, err := lobby.NewPacketHeader(request)
	if err != nil {
		sessionCtx.logger.Error("failed to parse request header", "error", err)
		return true
	}

	// attempt to lookup the account session
	sessionKey := string(header.Identifier[:])
	sessionCtx.logger.Info("looking up session", "sessionKey", sessionKey, "opCode", header.Command)
	accountSession, err := s.DB().GetAccountSessionBySessionKey(sessionCtx.ctx, sessionKey)
	if err != nil {
		// this shouldn't happen normally, log and close the connection
		sessionCtx.logger.Error("failed to lookup account session", "session_key", header.Identifier, "error", err)
		return true
	}

	// make sure this session context has subscriptions set up
	// this should only be done once per session
	if err = sessionCtx.SetupSubscriptions(accountSession.SessionKey); err != nil {
		sessionCtx.logger.Error("failed to setup subscriptions", "error", err)
		return true
	}

	// now, handle the request based on the command
	switch header.Command {
	case CommandRequestLobbyLogin:
		return s.handleRequestLobbyLogin(sessionCtx, request)
	case CommandRequestGetCharacter:
		return s.handleRequestGetCharacter(sessionCtx, &accountSession, request)
	case CommandRequestQueryWorldList:
		return s.handleRequestWorldList(sessionCtx, request)
	case CommandRequestCreateCharacterPre:
		return s.handleRequestCreateCharacterPre(sessionCtx, request)
	case CommandRequestCreateCharacter:
		return s.handleRequestCreateCharacter(sessionCtx, &accountSession, request)
	case CommandRequestSelectCharacter:
		return s.handleRequestSelectCharacter(sessionCtx, string(header.Identifier[:]), &accountSession, request)
	case CommandRequestDeleteCharacter:
		return s.handleRequestDeleteCharacter(sessionCtx, &accountSession, request)
	}

	return false
}

func (s *ViewServer) sendErrorResponse(sessionCtx *sessionContext, errorCode uint32) {
	response, err := lobby.NewResponseError(errorCode)
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
