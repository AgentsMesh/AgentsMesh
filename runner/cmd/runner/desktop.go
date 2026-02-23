//go:build desktop

package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/anthropics/agentsmesh/runner/internal/config"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/tray"
)

// runDesktop handles the "desktop" subcommand for system tray mode.
func runDesktop(args []string) {
	fs := flag.NewFlagSet("desktop", flag.ExitOnError)
	configFile := fs.String("config", "", "Path to config file (default: ~/.agentsmesh/config.yaml)")
	logLevel := fs.String("log-level", "", "Log level: debug, info, warn, error (overrides config)")

	fs.Usage = func() {
		fmt.Println(`Start AgentsMesh Runner in desktop mode with system tray.

Usage:
  runner desktop [options]

Options:`)
		fs.PrintDefaults()
		fmt.Println(`
The runner will appear in the system tray with a menu to:
  - View connection status
  - Start/Stop the runner
  - Open web console
  - View logs
  - Enable/disable auto-start at login

Log file is written to $TMPDIR/agentsmesh/runner.log by default (with rotation).
The runner must be registered first using 'runner register'.`)
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
		fmt.Println("Error: Runner not registered.")
		fmt.Println("Please run 'runner register' first.")
		os.Exit(1)
	}

	// Load configuration
	cfg, err := config.Load(cfgFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Load gRPC config (certificates)
	if err := cfg.LoadGRPCConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load gRPC config: %v - please re-register the runner\n", err)
		os.Exit(1)
	}

	// Override log level from command line if provided
	if *logLevel != "" {
		cfg.LogLevel = *logLevel
	}

	// Initialize logger
	if err := logger.Init(cfg.GetLogConfig()); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Close()

	// Validate config
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid config: %v\n", err)
		os.Exit(1)
	}

	slog.Info("Starting AgentsMesh Runner in desktop mode", "version", version)

	// Create and run tray app with version info
	app := tray.NewWithVersion(cfg, version)
	app.Run()
}
