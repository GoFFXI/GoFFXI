package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net"

	"github.com/pquerna/otp/totp"

	"github.com/GoFFXI/login-server/internal/database"
)

const (
	CommandRequestVerifyTOTP = 0x34

	SuccessCodeVerifyTOTPSuccess = 0x11
)

type ResponseVerifyTOTPSuccess struct {
	ResponseResult

	RecoveryCode string `json:"recovery_code"`
}

func (r ResponseVerifyTOTPSuccess) ToJSON() []byte {
	data, _ := json.Marshal(r)
	return data
}

func NewResponseVerifyTOTPSuccess(recoveryCode string) ResponseVerifyTOTPSuccess {
	return ResponseVerifyTOTPSuccess{
		ResponseResult: ResponseResult{
			ResultCode: SuccessCodeVerifyTOTPSuccess,
		},
		RecoveryCode: recoveryCode,
	}
}

func (s *AuthServer) handleRequestVerifyTOTP(ctx context.Context, conn net.Conn, header *RequestHeader) bool {
	logger := s.Logger().With("request", "verify-totp")
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
		response := NewResponseError("Account is banned")
		_, _ = conn.Write(response.ToJSON())

		return true
	}

	// check that the account has TOTP enabled
	accountTOTP, err := s.DB().GetAccountTOTPByAccountID(ctx, account.ID)
	if err != nil {
		var response ResponseError

		if errors.Is(err, database.ErrNotFound) {
			logger.Info("account does not have TOTP enabled", "username", header.Username)
			response = NewResponseError("TOTP is not enabled for this account")
		} else {
			logger.Error("failed to get account TOTP info", "error", err)
			response = NewResponseError("Failed to validate credentials")
		}

		_, _ = conn.Write(response.ToJSON())
		return false
	}

	// make sure account TOTP is not already validated
	if accountTOTP.Validated {
		logger.Info("account TOTP already validated", "username", header.Username)
		response := NewResponseError("TOTP is already validated for this account")
		_, _ = conn.Write(response.ToJSON())

		return false
	}

	// validate the totp code
	if !totp.Validate(header.OTP, accountTOTP.Secret) {
		logger.Info("invalid TOTP code", "username", header.Username)
		response := NewResponseError("Failed to validate credentials")
		_, _ = conn.Write(response.ToJSON())

		return false
	}

	// mark the TOTP as validated
	accountTOTP.Validated = true
	if _, err = s.DB().UpdateAccountTOTP(ctx, &accountTOTP); err != nil {
		logger.Error("failed to update account TOTP", "error", err)
		response := NewResponseError("Failed to validate credentials")
		_, _ = conn.Write(response.ToJSON())

		return false
	}

	logger.Info("TOTP verified successfully", "username", header.Username)
	response := NewResponseVerifyTOTPSuccess(accountTOTP.RecoveryCode)
	_, _ = conn.Write(response.ToJSON())

	return false
}
