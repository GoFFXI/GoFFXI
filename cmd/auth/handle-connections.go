package auth

import (
	"errors"
	"io"
	"net"
	"time"

	"github.com/GoFFXI/login-server/internal/tools"
)

const (
	OpResponseFail          = 0x00
	OpResponseSuccess       = 0x01
	OpErrorOccurred         = 0x02
	OpRequestAttemptLogin   = 0x10
	OpRequestCreateAccount  = 0x20
	OpRequestChangePassword = 0x30

	OpErrorInvalidClientVersion = 0x0B
)

func (s *AuthServer) handleConnection(conn net.Conn) {
	//nolint:errcheck // closing connection
	defer conn.Close()

	logger := s.Logger().With("client", conn.RemoteAddr().String())
	logger.Info("processing connection")

	// set read/write timeout for the connection
	_ = conn.SetDeadline(time.Now().Add(time.Duration(s.Config().ServerReadTimeoutSeconds) * time.Second))

	// connection handling loop
	for {
		// make sure we exit if the server is shutting down
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		// read data from client
		buffer := make([]byte, 1024)
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

		// parse the expected received data
		request := buffer[:length]
		username := tools.BytesToString(request, 0x09, 16)
		password := tools.BytesToString(request, 0x19, 32)
		clientVersion := tools.BytesToString(request, 0x61, 5)
		opCode := tools.GetIntFromByteBuffer(request, 0x39)

		// handle client version enforcement
		logger.Debug("detected client version", "version", clientVersion)
		if !s.clientVersionIsValid(clientVersion) {
			logger.Error("client version mismatch - disconnecting", "got", clientVersion)
			conn.Write([]byte{OpErrorInvalidClientVersion})
			conn.Close()
			return
		}

		switch opCode {
		case OpRequestAttemptLogin:
			s.opAttemptLogin(logger, username, password)
		case OpRequestCreateAccount:
			s.opCreateAccount(logger, username, password)
		case OpRequestChangePassword:
			s.opChangePassword(logger, username, password, buffer)
		}
	}
}

func (s *AuthServer) clientVersionIsValid(clientVersion string) bool {
	if s.Config().XIClientEnforceVersion == 1 {
		// enforce exact version match
		if !tools.VersionIsEqualTo(clientVersion, s.Config().XILoaderVersion) {
			return false
		}
	} else if s.Config().XIClientEnforceVersion == 2 {
		// enforce minimum version
		clientMajor, clientMinor, clientPatch := tools.GetVersionsFromString(clientVersion)
		minMajor, minMinor, minPatch := tools.GetVersionsFromString(s.Config().XILoaderVersion)

		if (clientMajor < minMajor) ||
			(clientMajor == minMajor && clientMinor < minMinor) ||
			(clientMajor == minMajor && clientMinor == minMinor && clientPatch < minPatch) {
			return false
		}
	}

	return true
}
