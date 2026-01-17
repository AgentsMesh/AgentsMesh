//go:build desktop

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/anthropics/agentsmesh/runner/internal/config"
	"github.com/anthropics/agentsmesh/runner/internal/tray"
)

// runDesktop handles the "desktop" subcommand for system tray mode.
func runDesktop(args []string) {
	fs := flag.NewFlagSet("desktop", flag.ExitOnError)
	configFile := fs.String("config", "", "Path to config file (default: ~/.agentsmesh/config.yaml)")

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
			log.Fatalf("Failed to get home directory: %v", err)
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

	log.Printf("Starting AgentsMesh Runner %s in desktop mode", version)

	// Create and run tray app with version info
	app := tray.NewWithVersion(cfg, version)
	app.Run()
}
