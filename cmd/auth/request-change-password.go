package auth

import (
	"context"
	"net"
	"strings"

	"github.com/GoFFXI/login-server/internal/tools"
	"golang.org/x/crypto/bcrypt"
)

const (
	ResponsePasswordChanged = 0x06

	ErrorChangingPassword = 0x07
)

func (s *AuthServer) handleRequestChangePassword(ctx context.Context, conn net.Conn, username, password string, buffer []byte) {
	s.Logger().Info("processing change password request")

	// extract the newPassword from the buffer
	newPassword := tools.BytesToString(buffer, 0x40, 32)
	newPassword = tools.StripStringUnicode(newPassword)

	// validate that the new password meets minimum length requirements
	newPassword = strings.TrimSpace(newPassword)
	if len(newPassword) < s.Config().MinPasswordLength {
		s.Logger().Warn("password too short", "length", len(newPassword))
		_, _ = conn.Write([]byte{ErrorChangingPassword})

		return
	}

	// attempt to lookup the account by username
	account, err := s.DB().GetAccountByUsername(ctx, username)
	if err != nil {
		s.Logger().Error("failed to get account", "error", err)
		_, _ = conn.Write([]byte{ErrorChangingPassword})
		return
	}

	// compare the passwords using bcrypt
	if err = bcrypt.CompareHashAndPassword([]byte(account.Password), []byte(password)); err != nil {
		s.Logger().Warn("invalid password", "username", username)
		_, _ = conn.Write([]byte{ErrorChangingPassword})
		return
	}

	// hash the new password using bcrypt
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		s.Logger().Error("failed to hash password", "error", err)
		_, _ = conn.Write([]byte{ErrorChangingPassword})
		return
	}

	// update the account password
	account.Password = string(hashedPassword)
	_, err = s.DB().UpdateAccount(ctx, &account)
	if err != nil {
		s.Logger().Error("failed to update account password", "error", err)
		_, _ = conn.Write([]byte{ErrorChangingPassword})

		return
	}

	// send back a success response
	s.Logger().Info("password changed successfully", "username", username)
	_, _ = conn.Write([]byte{ResponsePasswordChanged})
}
