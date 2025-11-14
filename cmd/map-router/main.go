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
	"github.com/GoFFXI/GoFFXI/internal/servers/base/udp"
	"github.com/GoFFXI/GoFFXI/internal/servers/map/router"
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

	// setup a new map router server
	baseServer, err := udp.NewUDPServer(&cfg, logger)
	if err != nil {
		logger.Error("could not create base server", "error", err)
		os.Exit(1)
	}

	mapRouterServer := router.NewMapRouterServer(baseServer)

	// connect to NATS server
	if err = mapRouterServer.CreateNATSConnection(); err != nil {
		logger.Error("failed to connect to NATS", "error", err)
		os.Exit(1)
	}

	// connect to database
	if err = mapRouterServer.CreateDBConnection(ctx); err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}

	//nolint:errcheck // socket will be closed on shutdown
	defer mapRouterServer.Socket().Close()

	// some house-keeping
	logger.Info("map-router server started", "version", Version, "buildDate", BuildDate, "gitCommit", GitCommit)
	defer cancelCtx()

	// start connection processor goroutine
	wg.Add(1)
	go mapRouterServer.ProcessConnections(ctx, &wg, mapRouterServer.HandleIncomingPacket)

	// start sending packets to clients
	wg.Add(1)
	go mapRouterServer.DeliverPacketsToClients(ctx, &wg)

	// wait for shutdown signal
	if err = mapRouterServer.WaitForShutdown(cancelCtx, &wg); err != nil {
		logger.Error("error during shutdown", "error", err)
	}
}
