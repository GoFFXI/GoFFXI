package data

import (
	"context"
	"net"

	"github.com/GoFFXI/login-server/internal/database"
)

func (s *DataServer) opNotifyLobbyOfCurrentSelection(_ context.Context, _ net.Conn, _ *database.AccountSession, _ []byte) {
	s.Logger().Info("notifying lobby of current selection")
}
