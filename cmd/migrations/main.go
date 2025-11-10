package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"os"
	"runtime"
	"time"

	"github.com/joho/godotenv"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/mysqldialect"
	"github.com/uptrace/bun/extra/bunslog"

	"github.com/GoFFXI/GoFFXI/internal/config"
	"github.com/GoFFXI/GoFFXI/internal/database"
	"github.com/GoFFXI/GoFFXI/internal/database/migrations"
)

func main() {
	// load .env file automatically
	err := godotenv.Load()
	if err != nil {
		log.Println("no .env file found (continuing with system environment)")
	}

	// parse config from environment
	cfg := config.ParseConfigFromEnv()

	// detect the log level
	logLevel := slog.LevelInfo
	if err = logLevel.UnmarshalText([]byte(cfg.LogLevel)); err != nil {
		fmt.Fprintf(os.Stderr, "error: invalid log level: '%s'\n", cfg.LogLevel)
		os.Exit(1)
	}

	// setup our logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))

	// create a context
	ctx := context.Background()

	// create a database connection
	db, err := createDBConnection(ctx, &cfg, logger)
	if err != nil {
		logger.Error("failed to create database connection", "error", err)
		os.Exit(1)
	}

	// run migrations
	if err = migrations.Migrate(ctx, db.BunDB()); err != nil {
		logger.Error("failed to run database migrations", "error", err)
		os.Exit(1)
	}
}

func createDBConnection(ctx context.Context, cfg *config.Config, logger *slog.Logger) (*database.DBImpl, error) {
	var db *bun.DB

	sqldb, err := sql.Open("mysql", cfg.DBConnectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// https://bun.uptrace.dev/guide/running-bun-in-production.html#running-bun-in-production
	maxOpenConns := 4 * runtime.GOMAXPROCS(0)
	sqldb.SetMaxOpenConns(maxOpenConns)
	sqldb.SetMaxIdleConns(maxOpenConns)

	db = bun.NewDB(sqldb, mysqldialect.New())

	queryLogLevel := slog.LevelDebug
	if cfg.DBQueryLogLevel == "info" {
		queryLogLevel = slog.LevelInfo
	}

	db.AddQueryHook(bunslog.NewQueryHook(
		bunslog.WithQueryLogLevel(queryLogLevel),
		bunslog.WithSlowQueryLogLevel(slog.LevelWarn),
		bunslog.WithErrorQueryLogLevel(slog.LevelError),
		bunslog.WithSlowQueryThreshold(3*time.Second),
		bunslog.WithLogger(logger.With("component", "database")),
	))

	if err = db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return database.NewDB(db), nil
}
