package auth

import (
	"context"
	"net"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/GoFFXI/login-server/internal/database"
)

const (
	ResponseAccountCreated = 0x03

	ErrorUsernameTaken           = 0x04
	ErrorAccountCreationDisabled = 0x08
	ErrorCreatingAccount         = 0x09
)

func (s *AuthServer) handleRequestCreateAccount(ctx context.Context, conn net.Conn, username, password string) {
	logger := s.Logger().With("request", "create-account")
	logger.Info("handling request")

	// check if account creation is enabled
	if !s.Config().AccountCreationEnabled {
		logger.Warn("account creation is disabled")
		_, _ = conn.Write([]byte{ErrorAccountCreationDisabled})

		return
	}

	// validate that the username meets minimum length requirements
	username = strings.TrimSpace(username)
	if len(username) < s.Config().MinUsernameLength {
		logger.Warn("username too short", "length", len(username))
		_, _ = conn.Write([]byte{ErrorCreatingAccount})

		return
	}

	// validate that the password meets minimum length requirements
	password = strings.TrimSpace(password)
	if len(password) < s.Config().MinPasswordLength {
		logger.Warn("password too short", "length", len(password))
		_, _ = conn.Write([]byte{ErrorCreatingAccount})

		return
	}

	// check if the username is already taken
	exists, err := s.DB().AccountExists(ctx, username)
	if err != nil {
		logger.Error("failed to check if account exists", "error", err)
		_, _ = conn.Write([]byte{ErrorCreatingAccount})

		return
	}

	if exists {
		logger.Warn("username already taken", "username", username)
		_, _ = conn.Write([]byte{ErrorUsernameTaken})

		return
	}

	// hash the password using bcrypt
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		logger.Error("failed to hash password", "error", err)
		_, _ = conn.Write([]byte{ResponseErrorOccurred})
		return
	}

	// create the account
	account := database.Account{
		Username: username,
		Password: string(hashedPassword),
	}
	_, err = s.DB().CreateAccount(ctx, &account)
	if err != nil {
		logger.Error("failed to create account", "error", err)
		_, _ = conn.Write([]byte{ErrorCreatingAccount})

		return
	}

	// send back a success response
	logger.Info("account created successfully", "username", username)
	_, _ = conn.Write([]byte{ResponseAccountCreated})
}
