package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"sync"

	"github.com/joho/godotenv"
	"go.uber.org/automaxprocs/maxprocs"

	"github.com/GoFFXI/GoFFXI/internal/config"
	"github.com/GoFFXI/GoFFXI/internal/servers/base/tcp"
	"github.com/GoFFXI/GoFFXI/internal/servers/lobby/data"
)

// version information - to be set during build time
var (
	Version   = "dev"
	BuildDate = "unknown"
	GitCommit = "none"
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

	// set the maxprocs
	if _, err = maxprocs.Set(maxprocs.Logger(func(message string, args ...any) {
		logger.Info(fmt.Sprintf(message, args...))
	})); err != nil {
		logger.Error("could not set GOMAXPROCS", "error", err)
	}

	// setup wait group for goroutines
	var wg sync.WaitGroup

	// create a context for graceful shutdown
	ctx, cancelCtx := context.WithCancel(context.Background())

	// setup a new lobby data server
	baseServer, err := tcp.NewTCPServer(ctx, &cfg, logger)
	if err != nil {
		logger.Error("failed to create base server", "error", err)
		os.Exit(1)
	}

	dataServer := &data.DataServer{
		TCPServer: baseServer,
	}

	// connect to NATS server
	if err = dataServer.CreateNATSConnection(); err != nil {
		logger.Error("failed to connect to NATS", "error", err)
		os.Exit(1)
	}

	// connect to database
	if err = dataServer.CreateDBConnection(ctx); err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}

	//nolint:errcheck // socket will be closed on shutdown
	defer dataServer.Socket().Close()

	// some house-keeping
	logger.Info("lobby-data server started", "version", Version, "buildDate", BuildDate, "gitCommit", GitCommit)
	defer cancelCtx()

	// start connection processor goroutine
	wg.Add(1)
	go dataServer.ProcessConnections(ctx, &wg, dataServer.HandleConnection)

	// start accepting connections
	wg.Add(1)
	go dataServer.AcceptConnections(ctx, &wg)

	// wait for shutdown signal
	if err = dataServer.WaitForShutdown(cancelCtx, &wg); err != nil {
		logger.Error("error during shutdown", "error", err)
	}
}
