package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"

	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"

	"github.com/GoFFXI/GoFFXI/internal/database"
)

const (
	CommandRequestAttemptLogin = 0x10

	SuccessCodeLoginSuccessful = 0x01

	ErrorCodeAttemptLoginFailed = 0x00
	ErrorCodeAttemptLoginError  = 0x02
)

type ResponseAttemptLoginSuccess struct {
	ResultCode uint8    `json:"result"`
	AccountID  uint32   `json:"account_id"`
	SessionKey [16]byte `json:"session_hash"`
}

func (re ResponseAttemptLoginSuccess) ToJSON() []byte {
	data, _ := json.Marshal(re)
	return data
}

func NewResponseAttemptLoginSuccess(accountID uint32, sessionKey [16]byte) ResponseAttemptLoginSuccess {
	return ResponseAttemptLoginSuccess{
		ResultCode: SuccessCodeLoginSuccessful,
		AccountID:  accountID,
		SessionKey: sessionKey,
	}
}

func (s *AuthServer) handleRequestAttemptLogin(ctx context.Context, conn net.Conn, header *RequestHeader) bool {
	logger := s.Logger().With("request", "attempt-login")
	logger.Info("handling request")

	// attempt to lookup the account by username
	account, err := s.DB().GetAccountByUsername(ctx, header.Username)
	if err != nil {
		logger.Error("failed to get account", "error", err)
		response := NewResponseResult(ErrorCodeAttemptLoginError)
		_, _ = conn.Write(response.ToJSON())

		return false
	}

	// check if the account is banned
	isBanned, err := s.DB().IsAccountBanned(ctx, account.ID)
	if err != nil {
		logger.Error("failed to check if account is banned", "error", err)
		response := NewResponseResult(ErrorCodeAttemptLoginError)
		_, _ = conn.Write(response.ToJSON())

		return false
	}

	if isBanned {
		logger.Warn("account is banned", "username", header.Username)
		response := NewResponseResult(ErrorCodeAttemptLoginFailed)
		_, _ = conn.Write(response.ToJSON())

		return true
	}

	// compare the passwords using bcrypt
	if err = bcrypt.CompareHashAndPassword([]byte(account.Password), []byte(header.Password)); err != nil {
		logger.Warn("invalid password", "username", header.Username)
		response := NewResponseResult(ErrorCodeAttemptLoginError)
		_, _ = conn.Write(response.ToJSON())

		return false
	}

	// check if TOPP is enabled for the account
	accountTOTP, err := s.DB().GetAccountTOTPByAccountID(ctx, account.ID)
	if err != nil {
		if !errors.Is(err, database.ErrNotFound) {
			logger.Error("failed to get account TOTP info", "error", err)
			response := NewResponseError("Failed to validate credentials")
			_, _ = conn.Write(response.ToJSON())

			return false
		}
	}

	// make sure TOTP is validated if it is enabled
	if accountTOTP.Validated && !totp.Validate(header.OTP, accountTOTP.Secret) {
		logger.Warn("invalid TOTP", "username", header.Username)
		response := NewResponseResult(ErrorCodeAttemptLoginError)
		_, _ = conn.Write(response.ToJSON())

		return true
	}

	// generate a session token
	logger.Info("login successful", "username", header.Username)
	sessionKey := generateSessionKey()
	logger.Debug("session token generated", "username", header.Username, "sessionToken", sessionKey)

	// parse the client IP address
	host, _, err := net.SplitHostPort(conn.RemoteAddr().String())
	if err != nil {
		logger.Error("failed to parse client address", "error", err)
		response := NewResponseResult(ErrorCodeAttemptLoginError)
		_, _ = conn.Write(response.ToJSON())

		return false
	}

	// extract the IPv4 address
	clientAddr := net.ParseIP(host).To4()
	if clientAddr == nil {
		logger.Error("failed to parse client IPv4 address", "address", host)
		response := NewResponseResult(ErrorCodeAttemptLoginError)
		_, _ = conn.Write(response.ToJSON())

		return false
	}

	// create the account session
	accountSession := &database.AccountSession{
		AccountID:   account.ID,
		CharacterID: 0, // No character selected yet
		SessionKey:  string(sessionKey[:]),
		ClientIP:    clientAddr.String(),
	}

	_, err = s.DB().CreateAccountSession(ctx, accountSession)
	if err != nil {
		logger.Error("failed to create account session", "error", err)
		response := NewResponseResult(ErrorCodeAttemptLoginError)
		_, _ = conn.Write(response.ToJSON())

		return false
	}

	// send the success response
	response := NewResponseAttemptLoginSuccess(account.ID, sessionKey)
	_, _ = conn.Write(response.ToJSON())

	return true
}

func generateSessionKey() [16]byte {
	// Generate a random 8-byte session key
	randomBytes := make([]byte, 8)
	_, _ = rand.Read(randomBytes)

	// Convert to hex string (8 bytes = 16 hex characters)
	encoded := hex.EncodeToString(randomBytes)

	var sessionKey [16]byte
	copy(sessionKey[:], encoded)

	return sessionKey
}
