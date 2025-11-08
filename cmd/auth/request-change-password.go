package auth

import (
	"context"
	"errors"
	"net"
	"strings"

	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"

	"github.com/GoFFXI/GoFFXI/internal/database"
)

const (
	CommandRequestChangePassword = 0x30

	SuccessCodeChangedPassword = 0x06

	ErrorCodeChangePasswordFailed = 0x07
)

func (s *AuthServer) handleRequestChangePassword(ctx context.Context, conn net.Conn, header *RequestHeader) bool {
	logger := s.Logger().With("request", "change-password")
	logger.Info("handling request")

	// validate that the new password meets minimum length requirements
	newPassword := strings.TrimSpace(header.NewPassword)
	if len(newPassword) < s.Config().MinPasswordLength {
		logger.Warn("password too short", "length", len(newPassword))
		response := NewResponseResult(ErrorCodeChangePasswordFailed)
		_, _ = conn.Write(response.ToJSON())

		return false
	}

	// attempt to lookup the account by username
	account, err := s.DB().GetAccountByUsername(ctx, header.Username)
	if err != nil {
		logger.Error("failed to get account", "error", err)
		response := NewResponseResult(ErrorCodeChangePasswordFailed)
		_, _ = conn.Write(response.ToJSON())

		return false
	}

	// check if the account is banned
	isBanned, err := s.DB().IsAccountBanned(ctx, account.ID)
	if err != nil {
		logger.Error("failed to check if account is banned", "error", err)
		response := NewResponseResult(ErrorCodeChangePasswordFailed)
		_, _ = conn.Write(response.ToJSON())

		return false
	}

	if isBanned {
		logger.Warn("account is banned", "username", header.Username)
		response := NewResponseResult(ErrorCodeChangePasswordFailed)
		_, _ = conn.Write(response.ToJSON())

		return true
	}

	// compare the passwords using bcrypt
	if err = bcrypt.CompareHashAndPassword([]byte(account.Password), []byte(header.Password)); err != nil {
		logger.Warn("invalid password", "username", header.Username)
		response := NewResponseResult(ErrorCodeChangePasswordFailed)
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

		return false
	}

	// hash the new password using bcrypt
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(header.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		logger.Error("failed to hash password", "error", err)
		response := NewResponseResult(ErrorCodeChangePasswordFailed)
		_, _ = conn.Write(response.ToJSON())

		return false
	}

	// update the account password
	account.Password = string(hashedPassword)
	_, err = s.DB().UpdateAccount(ctx, &account)
	if err != nil {
		logger.Error("failed to update account password", "error", err)
		response := NewResponseResult(ErrorCodeChangePasswordFailed)
		_, _ = conn.Write(response.ToJSON())

		return false
	}

	// send back a success response
	logger.Info("password changed successfully", "username", header.Username)
	response := NewResponseResult(SuccessCodeChangedPassword)
	_, _ = conn.Write(response.ToJSON())

	return false
}
