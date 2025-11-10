package auth

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"time"

	"github.com/GoFFXI/GoFFXI/internal/servers/base/tcp"
)

const (
	ErrorInvalidClientVersion = 0x0B
)

type AuthServer struct {
	*tcp.TCPServer
}

func (s *AuthServer) HandleConnection(ctx context.Context, conn net.Conn) {
	logger := s.Logger().With("client", conn.RemoteAddr().String())
	logger.Info("processing connection")

	//nolint:errcheck // connection will be closed at the end of this function
	defer conn.Close()

	// set read/write timeout for the connection
	_ = conn.SetDeadline(time.Now().Add(time.Duration(s.Config().ServerReadTimeoutSeconds) * time.Second))

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

		if shouldExit := s.parseIncomingRequest(ctx, logger, conn, buffer[:length]); shouldExit {
			break
		}
	}
}

func (s *AuthServer) parseIncomingRequest(ctx context.Context, logger *slog.Logger, conn net.Conn, request []byte) bool {
	// check if the request is from an old version of xiloader
	if request[0] == 0xFF {
		logger.Info("detected old version of xiloader client")
		_, _ = conn.Write([]byte{ErrorInvalidClientVersion})
		return true
	}

	// attempt to parse the JSON payloads
	header := RequestHeader{}
	if err := json.Unmarshal(request, &header); err != nil {
		logger.Error("failed to parse JSON header", "error", err)
		return true
	}

	// handle client version enforcement
	logger.Info("detected client version", "version", header.ClientVersion())
	if s.Config().XILoaderEnforceVersion == 2 {
		// enforce minimum version
		if !header.VersionAtLeast(s.Config().XILoaderVersion) {
			logger.Error("client version mismatch - disconnecting", "got", header.ClientVersion())
			responseError := NewResponseError("Your XI Loader version is outdated. Please update to at least " + s.Config().XILoaderVersion)
			_, _ = conn.Write(responseError.ToJSON())
			return true
		}
	} else if s.Config().XILoaderEnforceVersion == 1 {
		// enforce exact version match
		if !header.VersionMatches(s.Config().XILoaderVersion) {
			logger.Error("client version mismatch - disconnecting", "got", header.ClientVersion())
			responseError := NewResponseError("Your XI Loader version is incompatible. Please use version " + s.Config().XILoaderVersion)
			_, _ = conn.Write(responseError.ToJSON())
			return true
		}
	}

	// handle commands
	switch header.Command {
	case CommandRequestAttemptLogin:
		return s.handleRequestAttemptLogin(ctx, conn, &header)
	case CommandRequestCreateAccount:
		return s.handleRequestCreateAccount(ctx, conn, &header)
	case CommandRequestChangePassword:
		return s.handleRequestChangePassword(ctx, conn, &header)
	case CommandRequestCreateTOTP:
		return s.handleRequestCreateTOTP(ctx, conn, &header)
	case CommandRequestRemoveTOTP:
		return s.handleRequestRemoveTOTP(ctx, conn, &header)
	case CommandRequestRegenerateRecovery:
		return s.handleRequestRegenerateRecovery(ctx, conn, &header)
	case CommandRequestVerifyTOTP:
		return s.handleRequestVerifyTOTP(ctx, conn, &header)
	}

	return false
}
