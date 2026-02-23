package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"syscall"

	"github.com/anthropics/agentsmesh/runner/internal/config"
	"github.com/anthropics/agentsmesh/runner/internal/console"
	"github.com/anthropics/agentsmesh/runner/internal/lifecycle"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/runner"
)

// DefaultConsolePort is the default port for the web console.
const DefaultConsolePort = 19080

func runRunner(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	configFile := fs.String("config", "", "Path to config file (default: ~/.agentsmesh/config.yaml)")
	logLevel := fs.String("log-level", "", "Log level: debug, info, warn, error (overrides config)")
	logPTY := fs.Bool("logpty", false, "Log raw PTY and aggregator output to files for debugging")
	logPTYDir := fs.String("logpty-dir", "", "Directory for PTY logs (default: $TMPDIR/agentsmesh/pty-logs)")

	fs.Usage = func() {
		fmt.Println(`Start the AgentsMesh runner.

Usage:
  runner run [options]

Options:`)
		fs.PrintDefaults()
		fmt.Println(`
The runner must be registered first using 'runner register'.
Configuration is loaded from ~/.agentsmesh/config.yaml by default.
Log file is written to $TMPDIR/agentsmesh/runner.log by default (with rotation).

The runner uses gRPC/mTLS for secure communication with the server.`)
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	// Determine config file path
	cfgFile := *configFile
	if cfgFile == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get home directory: %v\n", err)
			os.Exit(1)
		}
		cfgFile = filepath.Join(home, ".agentsmesh", "config.yaml")
	}

	// Check if config exists
	if _, err := os.Stat(cfgFile); os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, "Error: Runner not registered. Please run 'runner register' first.")
		os.Exit(1)
	}

	// Load configuration
	cfg, err := config.Load(cfgFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Override log level from command line if provided
	if *logLevel != "" {
		cfg.LogLevel = *logLevel
	}

	// Override PTY logging from command line
	if *logPTY {
		cfg.LogPTY = true
	}
	if *logPTYDir != "" {
		cfg.LogPTYDir = *logPTYDir
	}

	// Print PTY log directory if enabled
	if cfg.LogPTY {
		fmt.Printf("PTY logging enabled, output directory: %s\n", cfg.GetLogPTYDir())
	}

	// Initialize logger
	if err := logger.Init(cfg.GetLogConfig()); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Close()

	log := slog.Default()

	// Load gRPC config (certificates)
	if err := cfg.LoadGRPCConfig(); err != nil {
		log.Error("Failed to load gRPC config - please re-register the runner", "error", err)
		os.Exit(1)
	}

	// Validate config
	if err := cfg.Validate(); err != nil {
		log.Error("Invalid config", "error", err)
		os.Exit(1)
	}

	if !cfg.UsesGRPC() {
		log.Error("gRPC configuration is required. Please re-register the runner using 'runner register'")
		os.Exit(1)
	}

	log.Info("Using gRPC/mTLS connection mode", "endpoint", cfg.GRPCEndpoint)

	// Pass build-time version to config for gRPC handshake
	cfg.Version = version

	startRunner(cfg)
}

func startRunner(cfg *config.Config) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "FATAL: Runner panic: %v\n%s\n", r, debug.Stack())
			os.Exit(1)
		}
	}()

	log := logger.Runner()

	// Create runner instance
	r, err := runner.New(cfg)
	if err != nil {
		log.Error("Failed to create runner", "error", err)
		os.Exit(1)
	}

	// Create web console (lifecycle managed by Supervisor)
	consoleServer := console.New(cfg, DefaultConsolePort, version)
	r.AddService(&lifecycle.ConsoleService{Server: consoleServer})

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Info("Received signal, shutting down...", "signal", sig)
		cancel()
	}()

	// Start runner
	log.Info("Starting AgentsMesh Runner", "version", version)

	// Update console status when runner state changes
	consoleServer.UpdateStatus(true, false, 0, 0, "")
	consoleServer.AddLog("info", "Runner starting...")

	if err := r.Run(ctx); err != nil {
		consoleServer.UpdateStatus(false, false, 0, 0, err.Error())
		consoleServer.AddLog("error", fmt.Sprintf("Runner error: %v", err))
		log.Error("Runner error", "error", err)
		os.Exit(1)
	}

	log.Info("Runner shutdown complete")
}
