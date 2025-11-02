package auth

import (
	"bytes"
	"context"
	"crypto/md5" //nolint:gosec // MD5 is used here for session key generation only
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"net"

	"github.com/GoFFXI/login-server/internal/database"
	"golang.org/x/crypto/bcrypt"
)

const (
	// For now, we're not going to enforce this particular error
	// It's just here for reference in the future if we want to use it
	ErrorAlreadyLoggedIn = 0x0A
)

func (s *AuthServer) opAttemptLogin(ctx context.Context, conn net.Conn, username, password string) {
	s.Logger().Info("attempting login", "username", username)

	// attempt to lookup the account by username
	account, err := s.DB().GetAccountByUsername(ctx, username)
	if err != nil {
		s.Logger().Error("failed to get account", "error", err)
		_, _ = conn.Write([]byte{ResponseFail})
		return
	}

	// compare the passwords using bcrypt
	if err = bcrypt.CompareHashAndPassword([]byte(account.Password), []byte(password)); err != nil {
		s.Logger().Warn("invalid password", "username", username)
		_, _ = conn.Write([]byte{ResponseFail})
		return
	}

	// generate a session token
	s.Logger().Info("login successful", "username", username)
	sessionKey := s.generateSessionKey()

	// parse the client IP address
	host, _, err := net.SplitHostPort(conn.RemoteAddr().String())
	if err != nil {
		s.Logger().Error("failed to parse client address", "error", err)
		_, _ = conn.Write([]byte{ResponseErrorOccurred})
		return
	}

	// only support IPv4 for now
	clientAddr := net.ParseIP(host).To4()
	if clientAddr == nil {
		s.Logger().Error("failed to parse client IPv4 address", "address", host)
		_, _ = conn.Write([]byte{ResponseErrorOccurred})
		return
	}

	// create the account session
	accountSession := &database.AccountSession{
		AccountID:     account.ID,
		CharacterID:   0, // No character selected yet
		SessionKey:    sessionKey,
		ClientAddress: binary.BigEndian.Uint32(clientAddr),
	}

	_, err = s.DB().CreateAccountSession(ctx, accountSession)
	if err != nil {
		s.Logger().Error("failed to create account session", "error", err)
		_, _ = conn.Write([]byte{ResponseErrorOccurred})
		return
	}

	// setup a new byte buffer
	responseBuffer := bytes.Buffer{}
	responseBuffer.WriteByte(ResponseSuccess)

	// add the account ID to the response buffer
	accountIDBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(accountIDBytes, account.ID)
	responseBuffer.Write(accountIDBytes)

	// add the session token to the response buffer
	responseBuffer.Write([]byte(sessionKey))

	// finally, write the response buffer to the connection
	_, _ = conn.Write(responseBuffer.Bytes())
	s.Logger().Debug("session token generated", "username", username, "sessionToken", sessionKey)
}

func (s *AuthServer) generateSessionKey() string {
	// Generate a random session hash
	randomBytes := make([]byte, 32)
	_, _ = rand.Read(randomBytes)

	// Create MD5 hash
	//nolint:gosec // MD5 is used here for session key generation only
	md5Hash := md5.Sum(randomBytes)

	return hex.EncodeToString(md5Hash[:])
}
