package auth

import "log/slog"

func (s *AuthServer) opAttemptLogin(logger *slog.Logger, username, password string) {
	logger.Debug("attempting login")
	_ = username
	_ = password
}
