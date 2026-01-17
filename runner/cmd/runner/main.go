package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/config"
	"github.com/anthropics/agentsmesh/runner/internal/console"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/runner"
)

var (
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "register":
		runRegister(os.Args[2:])
	case "run", "start":
		runRunner(os.Args[2:])
	case "service":
		runService(os.Args[2:])
	case "desktop":
		runDesktop(os.Args[2:])
	case "webconsole", "console":
		runWebConsole(os.Args[2:])
	case "reactivate":
		runReactivate(os.Args[2:])
	case "update":
		runUpdate(os.Args[2:])
	case "version", "-v", "--version":
		fmt.Printf("AgentsMesh Runner %s (built %s)\n", version, buildTime)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`AgentsMesh Runner

Usage:
  runner <command> [options]

Commands:
  register    Register this runner with the AgentsMesh server (gRPC/mTLS)
  run         Start the runner in CLI mode (requires prior registration)
  webconsole  Open the web console in browser
  service     Manage runner as a system service (install/start/stop)
  desktop     Start runner in desktop mode with system tray
  reactivate  Reactivate runner with expired certificate
  update      Check and install updates
  version     Show version information
  help        Show this help message

Use "runner <command> --help" for more information about a command.`)
}

func runRegister(args []string) {
	fs := flag.NewFlagSet("register", flag.ExitOnError)
	serverURL := fs.String("server", "", "AgentsMesh server URL (e.g., https://app.example.com)")
	token := fs.String("token", "", "Registration token (for token-based registration)")
	nodeID := fs.String("node-id", "", "Node ID for this runner (default: hostname)")

	fs.Usage = func() {
		fmt.Println(`Register this runner with the AgentsMesh server using gRPC/mTLS.

Usage:
  runner register [options]

Options:`)
		fs.PrintDefaults()
		fmt.Println(`
Registration Methods:

1. Interactive (Tailscale-style, recommended for first-time setup):
   runner register --server https://app.example.com

   Opens a browser for authorization. The runner will poll until you
   authorize it in the web UI.

2. Token-based (for automated/scripted deployment):
   runner register --server https://app.example.com --token <pre-generated-token>

   Uses a pre-generated token from the web UI. No browser required.

After successful registration, certificates and configuration will be saved to ~/.agentsmesh/`)
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	// Validate required flags
	if *serverURL == "" {
		fmt.Fprintln(os.Stderr, "Error: --server is required")
		os.Exit(1)
	}

	// Get node ID
	nID := *nodeID
	if nID == "" {
		hostname, err := os.Hostname()
		if err != nil {
			hostname = "runner"
		}
		nID = hostname
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute) // Longer timeout for interactive
	defer cancel()

	fmt.Printf("Registering runner '%s' with server %s...\n", nID, *serverURL)

	// gRPC/mTLS registration
	if *token != "" {
		// Token-based registration
		if err := registerWithGRPCToken(ctx, *serverURL, *token, nID); err != nil {
			fmt.Fprintf(os.Stderr, "Registration failed: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Interactive registration (Tailscale-style)
		if err := registerInteractive(ctx, *serverURL, nID); err != nil {
			fmt.Fprintf(os.Stderr, "Registration failed: %v\n", err)
			os.Exit(1)
		}
	}
	fmt.Println("✓ gRPC/mTLS Registration successful!")
}

func runReactivate(args []string) {
	fs := flag.NewFlagSet("reactivate", flag.ExitOnError)
	serverURL := fs.String("server", "", "AgentsMesh server URL (default: from config)")
	token := fs.String("token", "", "Reactivation token from the web UI")

	fs.Usage = func() {
		fmt.Println(`Reactivate a runner with an expired certificate.

Usage:
  runner reactivate --token <reactivation-token>

Options:`)
		fs.PrintDefaults()
		fmt.Println(`
When your runner's certificate expires (after long periods of inactivity),
you can generate a reactivation token from the web UI:

1. Go to Runner management page
2. Find your runner and click "Reactivate"
3. Copy the generated token
4. Run: runner reactivate --token <token>

The runner will receive new certificates and can reconnect.`)
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if *token == "" {
		fmt.Fprintln(os.Stderr, "Error: --token is required")
		os.Exit(1)
	}

	// Load server URL from config if not provided
	sURL := *serverURL
	if sURL == "" {
		home, _ := os.UserHomeDir()
		cfgFile := filepath.Join(home, ".agentsmesh", "config.yaml")
		cfg, err := config.Load(cfgFile)
		if err == nil && cfg.ServerURL != "" {
			sURL = cfg.ServerURL
		} else {
			fmt.Fprintln(os.Stderr, "Error: --server is required (no existing configuration found)")
			os.Exit(1)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Printf("Reactivating runner with server %s...\n", sURL)

	if err := reactivateRunner(ctx, sURL, *token); err != nil {
		fmt.Fprintf(os.Stderr, "Reactivation failed: %v\n", err)
		os.Exit(1)
	}
}

func runRunner(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	configFile := fs.String("config", "", "Path to config file (default: ~/.agentsmesh/config.yaml)")
	logLevel := fs.String("log-level", "", "Log level: debug, info, warn, error (overrides config)")

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

	// Load org slug
	if err := cfg.LoadOrgSlug(); err != nil {
		log.Warn("Failed to load org slug", "error", err)
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

	startRunner(cfg)
}

// DefaultConsolePort is the default port for the web console.
const DefaultConsolePort = 19080

func startRunner(cfg *config.Config) {
	log := logger.Runner()

	// Create runner instance
	r, err := runner.New(cfg)
	if err != nil {
		log.Error("Failed to create runner", "error", err)
		os.Exit(1)
	}

	// Start web console
	consoleServer := console.New(cfg, DefaultConsolePort, version)
	if err := consoleServer.Start(); err != nil {
		log.Warn("Failed to start web console", "error", err)
	} else {
		log.Info("Web console available", "url", consoleServer.GetURL())
	}

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

	// Stop web console
	consoleServer.Stop()
	log.Info("Runner shutdown complete")
}
