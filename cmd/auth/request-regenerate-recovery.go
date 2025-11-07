package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net"

	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"

	"github.com/GoFFXI/login-server/internal/database"
)

const (
	CommandRequestRegenerateRecovery = 0x33

	SuccessCodeRegenerateRecovery = 0x11
)

type ResponseRegenerateRecoverySuccess struct {
	ResponseResult

	RecoveryCode string `json:"recovery_code"`
}

func (r ResponseRegenerateRecoverySuccess) ToJSON() []byte {
	data, _ := json.Marshal(r)
	return data
}

func NewResponseRegenerateRecoverySuccess(recoveryCode string) *ResponseRegenerateRecoverySuccess {
	return &ResponseRegenerateRecoverySuccess{
		ResponseResult: ResponseResult{
			ResultCode: SuccessCodeRegenerateRecovery,
		},
		RecoveryCode: recoveryCode,
	}
}

func (s *AuthServer) handleRequestRegenerateRecovery(ctx context.Context, conn net.Conn, header *RequestHeader) bool {
	logger := s.Logger().With("request", "regenerate-recovery")
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

	// generate a new recovery code & save to db
	accountTOTP.RecoveryCode = getNewBase32Secret()
	if _, err = s.DB().UpdateAccountTOTP(ctx, &accountTOTP); err != nil {
		logger.Error("failed to update account TOTP recovery code", "error", err)
		response := NewResponseError("Failed to regenerate recovery code")
		_, _ = conn.Write(response.ToJSON())

		return false
	}

	// respond with success
	logger.Info("TOTP recovery code regenerated successfully", "username", header.Username)
	response := NewResponseRegenerateRecoverySuccess(accountTOTP.RecoveryCode)
	_, _ = conn.Write(response.ToJSON())

	return false
}
