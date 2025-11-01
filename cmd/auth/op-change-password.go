package auth

import (
	"log/slog"

	"github.com/GoFFXI/login-server/internal/tools"
)

func (s *AuthServer) opChangePassword(logger *slog.Logger, username, password string, buffer []byte) {
	logger.Debug("processing change password request")

	// extract the newPassword from the buffer
	newPassword := tools.BytesToString(buffer, 0x40, 32)
	newPassword = tools.StripStringUnicode(newPassword)
	_ = newPassword
}
