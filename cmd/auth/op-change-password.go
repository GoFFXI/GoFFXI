package auth

import (
	"context"
	"net"

	"github.com/GoFFXI/login-server/internal/tools"
)

const (
	ResponsePasswordChanged = 0x06

	ErrorChangingPassword = 0x07
)

func (s *AuthServer) opChangePassword(_ context.Context, _ net.Conn, username, password string, buffer []byte) {
	s.Logger().Debug("processing change password request")

	// extract the newPassword from the buffer
	newPassword := tools.BytesToString(buffer, 0x40, 32)
	newPassword = tools.StripStringUnicode(newPassword)
	_ = newPassword
	_ = username
	_ = password
}
