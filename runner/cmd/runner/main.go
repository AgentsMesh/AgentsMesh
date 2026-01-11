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

	"github.com/anthropics/agentmesh/runner/internal/config"
	"github.com/anthropics/agentmesh/runner/internal/runner"
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
	case "version", "-v", "--version":
		fmt.Printf("AgentsMesh Runner %s (built %s)\n", version, buildTime)
	case "help", "-h", "--help":
		printUsage()
	default:
		// Backward compatibility: if no subcommand, assume it's the old flag-based style
		runRunnerLegacy(os.Args[1:])
	}
}

func printUsage() {
	fmt.Println(`AgentsMesh Runner

Usage:
  runner <command> [options]

Commands:
  register    Register this runner with the AgentsMesh server
  run         Start the runner in CLI mode (requires prior registration)
  service     Manage runner as a system service (install/start/stop)
  desktop     Start runner in desktop mode with system tray
  version     Show version information
  help        Show this help message

Use "runner <command> --help" for more information about a command.`)
}

func runRegister(args []string) {
	fs := flag.NewFlagSet("register", flag.ExitOnError)
	serverURL := fs.String("server", "", "AgentsMesh server URL (e.g., http://localhost:8080)")
	token := fs.String("token", "", "Registration token from the server")
	nodeID := fs.String("node-id", "", "Node ID for this runner (default: hostname)")
	description := fs.String("description", "AgentsMesh Runner", "Description for this runner")
	maxPods := fs.Int("max-pods", 5, "Maximum concurrent pods")

	fs.Usage = func() {
		fmt.Println(`Register this runner with the AgentsMesh server.

Usage:
  runner register [options]

Options:`)
		fs.PrintDefaults()
		fmt.Println(`
Example:
  runner register --server http://localhost:8080 --token abc123def456

After successful registration, the auth token and config will be saved to ~/.agentsmesh/`)
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	// Validate required flags
	if *serverURL == "" {
		log.Fatal("Error: --server is required")
	}
	if *token == "" {
		log.Fatal("Error: --token is required")
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

	// Create a temporary client for registration
	client := &registrationClient{
		serverURL: *serverURL,
		nodeID:    nID,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Printf("Registering runner '%s' with server %s...\n", nID, *serverURL)

	result, err := client.register(ctx, *token, *description, *maxPods)
	if err != nil {
		log.Fatalf("Registration failed: %v", err)
	}

	// Save configuration to ~/.agentsmesh/
	if err := saveConfig(nID, *serverURL, result.AuthToken, result.OrgSlug, *description, *maxPods); err != nil {
		log.Fatalf("Failed to save configuration: %v", err)
	}

	fmt.Println("✓ Registration successful!")
	fmt.Printf("✓ Organization: %s\n", result.OrgSlug)
	fmt.Printf("✓ Configuration saved to ~/.agentsmesh/\n")
	fmt.Println("\nYou can now start the runner with:")
	fmt.Println("  runner run")
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
Configuration is loaded from ~/.agentsmesh/config.yaml by default.`)
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

	// Load auth token
	if err := cfg.LoadAuthToken(); err != nil {
		log.Printf("Warning: Failed to load auth token: %v", err)
	}

	// Validate config
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid config: %v", err)
	}

	startRunner(cfg)
}

// runRunnerLegacy provides backward compatibility with old flag-based CLI
func runRunnerLegacy(args []string) {
	fs := flag.NewFlagSet("runner", flag.ExitOnError)
	configFile := fs.String("config", "", "Path to config file")
	showVersion := fs.Bool("version", false, "Show version")
	registerToken := fs.String("token", "", "Registration token for initial registration")

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if *showVersion {
		fmt.Printf("AgentsMesh Runner %s (built %s)\n", version, buildTime)
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.Load(*configFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Override with command line flags
	if *registerToken != "" {
		cfg.RegistrationToken = *registerToken
	}

	// Validate config
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid config: %v", err)
	}

	startRunner(cfg)
}

func startRunner(cfg *config.Config) {
	// Create runner instance
	r, err := runner.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create runner: %v", err)
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
	if err := r.Run(ctx); err != nil {
		log.Fatalf("Runner error: %v", err)
	}

	log.Println("Runner shutdown complete")
}
