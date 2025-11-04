package auth

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"net"

	"github.com/GoFFXI/login-server/internal/database"
	"golang.org/x/crypto/bcrypt"
)

const (
	RequestAttemptLogin = 0x10

	// For now, we're not going to enforce this particular error
	// It's just here for reference in the future if we want to use it
	ErrorAlreadyLoggedIn = 0x0A
)

type ResponseAttemptLogin struct {
	Status     uint8    // Status code (0x01 for success)
	AccountID  uint32   // Account ID (Big-endian!)
	SessionKey [16]byte // Session key/token (variable length)
}

func (r *ResponseAttemptLogin) Serialize() []byte {
	buf := new(bytes.Buffer)

	// Write status byte
	buf.WriteByte(r.Status)

	// Write account ID in BIG-ENDIAN (matching your original code)
	_ = binary.Write(buf, binary.BigEndian, r.AccountID)

	// Write session key
	buf.Write(r.SessionKey[:])

	return buf.Bytes()
}

func (s *AuthServer) handleRequestAttemptLogin(ctx context.Context, conn net.Conn, username, password string) bool {
	logger := s.Logger().With("request", "attempt-login")
	logger.Info("handling request")

	// attempt to lookup the account by username
	account, err := s.DB().GetAccountByUsername(ctx, username)
	if err != nil {
		logger.Error("failed to get account", "error", err)
		_, _ = conn.Write([]byte{ResponseFail})
		return false
	}

	// compare the passwords using bcrypt
	if err = bcrypt.CompareHashAndPassword([]byte(account.Password), []byte(password)); err != nil {
		logger.Warn("invalid password", "username", username)
		_, _ = conn.Write([]byte{ResponseFail})
		return false
	}

	// generate a session token
	logger.Info("login successful", "username", username)
	sessionKey := s.generateSessionKey()
	logger.Debug("session token generated", "username", username, "sessionToken", sessionKey)

	// parse the client IP address
	host, _, err := net.SplitHostPort(conn.RemoteAddr().String())
	if err != nil {
		logger.Error("failed to parse client address", "error", err)
		_, _ = conn.Write([]byte{ResponseErrorOccurred})
		return false
	}

	// only support IPv4 for now
	clientAddr := net.ParseIP(host).To4()
	if clientAddr == nil {
		logger.Error("failed to parse client IPv4 address", "address", host)
		_, _ = conn.Write([]byte{ResponseErrorOccurred})
		return false
	}

	// create the account session
	accountSession := &database.AccountSession{
		AccountID:     account.ID,
		CharacterID:   0, // No character selected yet
		SessionKey:    string(sessionKey[:]),
		ClientAddress: binary.BigEndian.Uint32(clientAddr),
	}

	_, err = s.DB().CreateAccountSession(ctx, accountSession)
	if err != nil {
		logger.Error("failed to create account session", "error", err)
		_, _ = conn.Write([]byte{ResponseErrorOccurred})
		return false
	}

	response := &ResponseAttemptLogin{
		Status:     ResponseSuccess,
		AccountID:  account.ID,
		SessionKey: sessionKey,
	}

	// finally, write the response buffer to the connection
	_, _ = conn.Write(response.Serialize())

	return true
}

func (s *AuthServer) generateSessionKey() [16]byte {
	// Generate a random 8-byte session key
	randomBytes := make([]byte, 8)
	_, _ = rand.Read(randomBytes)

	// Convert to hex string (8 bytes = 16 hex characters)
	encoded := hex.EncodeToString(randomBytes)

	var sessionKey [16]byte
	copy(sessionKey[:], encoded)

	return sessionKey
}
