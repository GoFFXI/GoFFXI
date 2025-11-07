package auth

import (
	"context"
	"net"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/GoFFXI/login-server/internal/database"
)

const (
	CommandRequestCreateAccount = 0x20

	SuccessCodeAccountCreated = 0x03

	ErrorCodeAccountCreationDisabled = 0x08
	ErrorCodeAccountCreateFailed     = 0x09
	ErrorCodeAccountUsernameTaken    = 0x04
)

func (s *AuthServer) handleRequestCreateAccount(ctx context.Context, conn net.Conn, header *RequestHeader) bool {
	logger := s.Logger().With("request", "create-account")
	logger.Info("handling request")

	// check if account creation is enabled
	if !s.Config().AccountCreationEnabled {
		logger.Warn("account creation is disabled")
		response := NewResponseResult(ErrorCodeAccountCreationDisabled)
		_, _ = conn.Write(response.ToJSON())

		return false
	}

	// validate that the username meets minimum length requirements
	username := strings.TrimSpace(header.Username)
	if len(username) < s.Config().MinUsernameLength {
		logger.Warn("username too short", "length", len(username))
		response := NewResponseResult(ErrorCodeAccountCreateFailed)
		_, _ = conn.Write(response.ToJSON())

		return false
	}

	// validate that the password meets minimum length requirements
	password := strings.TrimSpace(header.Password)
	if len(password) < s.Config().MinPasswordLength {
		logger.Warn("password too short", "length", len(password))
		response := NewResponseResult(ErrorCodeAccountCreateFailed)
		_, _ = conn.Write(response.ToJSON())

		return false
	}

	// check if the username is already taken
	exists, err := s.DB().AccountExists(ctx, username)
	if err != nil {
		logger.Error("failed to check if account exists", "error", err)
		response := NewResponseResult(ErrorCodeAccountCreateFailed)
		_, _ = conn.Write(response.ToJSON())

		return false
	}

	if exists {
		logger.Warn("username already taken", "username", username)
		response := NewResponseResult(ErrorCodeAccountUsernameTaken)
		_, _ = conn.Write(response.ToJSON())

		return false
	}

	// hash the password using bcrypt
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		logger.Error("failed to hash password", "error", err)
		response := NewResponseResult(ErrorCodeAccountCreateFailed)
		_, _ = conn.Write(response.ToJSON())
		return false
	}

	// create the account
	account := database.Account{
		Username: username,
		Password: string(hashedPassword),
	}
	_, err = s.DB().CreateAccount(ctx, &account)
	if err != nil {
		logger.Error("failed to create account", "error", err)
		response := NewResponseResult(ErrorCodeAccountCreateFailed)
		_, _ = conn.Write(response.ToJSON())

		return false
	}

	logger.Info("account created successfully", "username", username)
	response := NewResponseResult(SuccessCodeAccountCreated)
	_, _ = conn.Write(response.ToJSON())
	return false
}
