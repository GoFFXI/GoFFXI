package auth

import (
	"context"
	"errors"
	"net"

	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"

	"github.com/GoFFXI/GoFFXI/internal/database"
)

const (
	CommandRequestRemoveTOTP = 0x32

	SuccessCodeRemovedTOTP = 0x12
)

type ResponseRemoveTOTPSuccess struct {
	Result string `json:"result"`
}

func (s *AuthServer) handleRequestRemoveTOTP(ctx context.Context, conn net.Conn, header *RequestHeader) bool {
	logger := s.Logger().With("request", "remove-totp")
	logger.Info("handling request")

	// attempt to lookup the account by username
	account, err := s.DB().GetAccountByUsername(ctx, header.Username)
	if err != nil {
		logger.Error("failed to get account", "error", err)
		response := NewResponseError("Failed to validate credentials")
		_, _ = conn.Write(response.ToJSON())

		return false
	}

	// check if the account is banned
	isBanned, err := s.DB().IsAccountBanned(ctx, account.ID)
	if err != nil {
		logger.Error("failed to check if account is banned", "error", err)
		response := NewResponseError("Failed to validate credentials")
		_, _ = conn.Write(response.ToJSON())

		return false
	}

	if isBanned {
		logger.Warn("account is banned", "username", header.Username)
		response := NewResponseError("Failed to validate credentials")
		_, _ = conn.Write(response.ToJSON())

		return true
	}

	// compare the passwords using bcrypt
	if err = bcrypt.CompareHashAndPassword([]byte(account.Password), []byte(header.Password)); err != nil {
		logger.Warn("invalid password", "username", header.Username)
		response := NewResponseError("Failed to validate credentials")
		_, _ = conn.Write(response.ToJSON())

		return false
	}

	// check if TOPP is enabled for the account
	accountTOTP, err := s.DB().GetAccountTOTPByAccountID(ctx, account.ID)
	if err != nil {
		if !errors.Is(err, database.ErrNotFound) {
			logger.Error("failed to get account TOTP info", "error", err)
			response := NewResponseError("TOTP is not enabled for account")
			_, _ = conn.Write(response.ToJSON())

			return false
		}
	}

	// ensure TOTP is enabled
	if !accountTOTP.Validated {
		logger.Warn("TOTP not validated for account", "username", header.Username)
		response := NewResponseError("TOTP is not enabled for account")
		_, _ = conn.Write(response.ToJSON())

		return false
	}

	// make sure TOTP is valid OR the recovery code matches
	isValidOTP := totp.Validate(header.OTP, accountTOTP.Secret)
	recoverCodeMatches := (header.OTP == accountTOTP.RecoveryCode)

	if !isValidOTP && !recoverCodeMatches {
		logger.Warn("invalid TOTP", "username", header.Username)
		response := NewResponseError("Failed to validate credentials")
		_, _ = conn.Write(response.ToJSON())

		return false
	}

	// remove the TOTP entry from the database
	if err = s.DB().DeleteAccountTOTP(ctx, account.ID); err != nil {
		logger.Error("failed to remove TOTP for account", "error", err)
		response := NewResponseError("Failed to remove TOTP")
		_, _ = conn.Write(response.ToJSON())

		return false
	}

	// respond with success
	logger.Info("TOTP removed successfully", "username", header.Username)
	response := NewResponseResult(SuccessCodeRemovedTOTP)
	_, _ = conn.Write(response.ToJSON())

	return false
}
