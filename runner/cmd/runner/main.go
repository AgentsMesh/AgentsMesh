package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/config"
	"github.com/anthropics/agentsmesh/runner/internal/console"
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
		log.Fatal("Error: --server is required")
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
			log.Fatalf("Registration failed: %v", err)
		}
	} else {
		// Interactive registration (Tailscale-style)
		if err := registerInteractive(ctx, *serverURL, nID); err != nil {
			log.Fatalf("Registration failed: %v", err)
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
		log.Fatal("Error: --token is required")
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
			log.Fatal("Error: --server is required (no existing configuration found)")
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Printf("Reactivating runner with server %s...\n", sURL)

	if err := reactivateRunner(ctx, sURL, *token); err != nil {
		log.Fatalf("Reactivation failed: %v", err)
	}
}

func runRunner(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	configFile := fs.String("config", "", "Path to config file (default: ~/.agentsmesh/config.yaml)")

	fs.Usage = func() {
		fmt.Println(`Start the AgentsMesh runner.

Usage:
  runner run [options]

Options:`)
		fs.PrintDefaults()
		fmt.Println(`
The runner must be registered first using 'runner register'.
Configuration is loaded from ~/.agentsmesh/config.yaml by default.

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
			log.Fatalf("Failed to get home directory: %v", err)
		}
		cfgFile = filepath.Join(home, ".agentsmesh", "config.yaml")
	}

	// Check if config exists
	if _, err := os.Stat(cfgFile); os.IsNotExist(err) {
		log.Fatal("Error: Runner not registered. Please run 'runner register' first.")
	}

	// Load configuration
	cfg, err := config.Load(cfgFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Load gRPC config (certificates)
	if err := cfg.LoadGRPCConfig(); err != nil {
		log.Fatalf("Failed to load gRPC config: %v - please re-register the runner", err)
	}

	// Load org slug
	if err := cfg.LoadOrgSlug(); err != nil {
		log.Printf("Warning: Failed to load org slug: %v", err)
	}

	// Validate config
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid config: %v", err)
	}

	if !cfg.UsesGRPC() {
		log.Fatal("Error: gRPC configuration is required. Please re-register the runner using 'runner register'")
	}

	log.Printf("Using gRPC/mTLS connection mode (endpoint: %s)", cfg.GRPCEndpoint)

	startRunner(cfg)
}

// DefaultConsolePort is the default port for the web console.
const DefaultConsolePort = 19080

func startRunner(cfg *config.Config) {
	// Create runner instance
	r, err := runner.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create runner: %v", err)
	}

	// Start web console
	consoleServer := console.New(cfg, DefaultConsolePort, version)
	if err := consoleServer.Start(); err != nil {
		log.Printf("Warning: Failed to start web console: %v", err)
	} else {
		log.Printf("Web console available at %s", consoleServer.GetURL())
	}

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("Received signal %v, shutting down...", sig)
		cancel()
	}()

	// Start runner
	log.Printf("Starting AgentsMesh Runner %s", version)

	// Update console status when runner state changes
	consoleServer.UpdateStatus(true, false, 0, 0, "")
	consoleServer.AddLog("info", "Runner starting...")

	if err := r.Run(ctx); err != nil {
		consoleServer.UpdateStatus(false, false, 0, 0, err.Error())
		consoleServer.AddLog("error", fmt.Sprintf("Runner error: %v", err))
		log.Fatalf("Runner error: %v", err)
	}

	// Stop web console
	consoleServer.Stop()
	log.Println("Runner shutdown complete")
}
