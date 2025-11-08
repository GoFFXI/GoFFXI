package server

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"runtime"
	"time"

	// This is necessary to register the MySQL driver
	_ "github.com/go-sql-driver/mysql"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/mysqldialect"
	"github.com/uptrace/bun/extra/bunslog"

	"github.com/GoFFXI/GoFFXI/internal/database"
)

func (s *Server) CreateDBConnection(ctx context.Context) error {
	var db *bun.DB

	sqldb, err := sql.Open("mysql", s.Config().DBConnectionString)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}

	// https://bun.uptrace.dev/guide/running-bun-in-production.html#running-bun-in-production
	maxOpenConns := 4 * runtime.GOMAXPROCS(0)
	sqldb.SetMaxOpenConns(maxOpenConns)
	sqldb.SetMaxIdleConns(maxOpenConns)

	db = bun.NewDB(sqldb, mysqldialect.New())

	queryLogLevel := slog.LevelDebug
	if s.Config().DBQueryLogLevel == "info" {
		queryLogLevel = slog.LevelInfo
	}

	db.AddQueryHook(bunslog.NewQueryHook(
		bunslog.WithQueryLogLevel(queryLogLevel),
		bunslog.WithSlowQueryLogLevel(slog.LevelWarn),
		bunslog.WithErrorQueryLogLevel(slog.LevelError),
		bunslog.WithSlowQueryThreshold(3*time.Second),
		bunslog.WithLogger(s.Logger().With("component", "database")),
	))

	if err = db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	s.db = database.NewDB(db)
	return nil
}
