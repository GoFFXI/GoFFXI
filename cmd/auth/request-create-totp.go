package auth

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"net"

	"github.com/GoFFXI/GoFFXI/internal/database"
)

const (
	CommandRequestCreateTOTP = 0x31

	SuccessCodeCreateTOTPSuccess = 0x10

	Base32OTPLength     = 32 // must be % 8 == 0
	Base32OTPCharacters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"
)

type ResponseCreateTOTPSuccess struct {
	ResultCode uint8  `json:"result"`
	TOTPURL    string `json:"TOTP_uri"`
}

func (re ResponseCreateTOTPSuccess) ToJSON() []byte {
	data, _ := json.Marshal(re)
	return data
}

func NewResponseCreateTOTPSuccess(totpURL string) ResponseCreateTOTPSuccess {
	return ResponseCreateTOTPSuccess{
		ResultCode: SuccessCodeCreateTOTPSuccess,
		TOTPURL:    totpURL,
	}
}

func (s *AuthServer) handleRequestCreateTOTP(ctx context.Context, conn net.Conn, header *RequestHeader) bool {
	logger := s.Logger().With("request", "create-totp")
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

	// check if the account already has TOTP enabled
	totpAlreadyEnabled, err := s.DB().AccountHasTOTPEnabled(ctx, account.ID)
	if err != nil {
		logger.Error("failed to check if account has TOTP enabled", "error", err)
		response := NewResponseError("Failed to validate credentials")
		_, _ = conn.Write(response.ToJSON())

		return false
	}

	if totpAlreadyEnabled {
		logger.Info("account already has TOTP enabled", "username", header.Username)
		response := NewResponseError("TOTP is already enabled for this account")
		_, _ = conn.Write(response.ToJSON())

		return false
	}

	// generate a secret & recovery code for TOTP
	totpSecret := getNewBase32Secret()
	totpRecoverCode := getNewBase32Secret()

	// store the TOTP secret & recovery code in the database
	accountTOTP := &database.AccountTOTP{
		AccountID:    account.ID,
		Secret:       totpSecret,
		RecoveryCode: totpRecoverCode,
	}

	if _, err = s.DB().CreateAccountTOTP(ctx, accountTOTP); err != nil {
		logger.Error("failed to create account TOTP", "error", err)
		response := NewResponseError("Failed to validate credentials")
		_, _ = conn.Write(response.ToJSON())

		return false
	}

	// generate the success response with the TOTP URL
	logger.Info("TOTP created successfully", "username", header.Username)
	totpURL := generateOTPURL(account.Username, s.Config().WorldName, totpSecret)
	response := NewResponseCreateTOTPSuccess(totpURL)
	_, _ = conn.Write(response.ToJSON())

	return false
}

// GetNewBase32Secret generates a new random Base32 secret for OTP
func getNewBase32Secret() string {
	secret := make([]byte, Base32OTPLength)

	for i := range Base32OTPLength {
		// Generate a random index between 0 and 31
		n, _ := rand.Int(rand.Reader, big.NewInt(32))
		secret[i] = Base32OTPCharacters[n.Int64()]
	}

	return string(secret)
}

func generateOTPURL(username, issuer, secret string) string {
	return fmt.Sprintf(
		"otpauth://totp/%s:%s?secret=%s&issuer=%s&algorithm=%s&digits=6&period=30",
		issuer,
		username,
		secret,
		issuer,
		"SHA1",
	)
}
