package auth

import "log/slog"

func (s *AuthServer) opCreateAccount(logger *slog.Logger, username, password string) {
	logger.Debug("processing create account request")
}
