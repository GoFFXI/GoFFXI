package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"go.uber.org/automaxprocs/maxprocs"

	"github.com/GoFFXI/login-server/cmd/auth"
	"github.com/GoFFXI/login-server/cmd/data"
	"github.com/GoFFXI/login-server/cmd/view"
	"github.com/GoFFXI/login-server/internal/config"
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

	role := handleFlags()
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

	switch role {
	case "auth":
		logger = logger.With("role", "auth")
		logger.Info("starting login-server...")
		if err = auth.Run(&cfg, logger); err != nil {
			logger.Error("failed to start login-server", "error", err)
			os.Exit(1)
		}
	case "data":
		logger = logger.With("role", "data")
		logger.Info("starting login-server...")
		if err = data.Run(&cfg, logger); err != nil {
			logger.Error("failed to start login-server", "error", err)
			os.Exit(1)
		}
	case "view":
		logger = logger.With("role", "view")
		logger.Info("starting login-server...")
		if err = view.Run(&cfg, logger); err != nil {
			logger.Error("failed to start login-server", "error", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "error: invalid role: '%s'. must be one of: auth, data, view\n", role)
		printUsage()
		os.Exit(1)
	}
}

func handleFlags() string {
	// define command line flags
	role := flag.String("role", "", "role to run: auth, data, view")
	version := flag.Bool("version", false, "show version information")
	help := flag.Bool("help", false, "show help message")

	// parse all flags
	flag.Parse()

	// handle version flag
	if *version {
		fmt.Printf("login-server: %s\n", Version)
		fmt.Printf("Build time: %s\n", BuildDate)
		fmt.Printf("Git commit: %s\n", GitCommit)
		os.Exit(0)
	}

	// handle help flag or empty role
	if *help || *role == "" {
		printUsage()
		if *help {
			os.Exit(0)
		}
		os.Exit(1)
	}

	return strings.ToLower(*role)
}

func printUsage() {
	fmt.Println("Usage: login-server --role=<role>")
	fmt.Println()
	fmt.Println("Available roles:")
	fmt.Println("  auth   - Run as authentication server")
	fmt.Println("  data   - Run as data management server")
	fmt.Println("  view   - Run as view/presentation server")
	fmt.Println()
	fmt.Println("Flags:")
	flag.PrintDefaults()
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  login-server --role=auth")
	fmt.Println("  login-server --role=data")
	fmt.Println("  login-server --role=view")
}
