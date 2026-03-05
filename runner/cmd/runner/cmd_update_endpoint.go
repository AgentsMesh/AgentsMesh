package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/config"
)

func runUpdateEndpoint(args []string) {
	fs := flag.NewFlagSet("update-endpoint", flag.ExitOnError)
	configFile := fs.String("config", "", "Path to config file (default: ~/.agentsmesh/config.yaml)")
	serverURL := fs.String("server-url", "", "Override server URL for discovery (default: read from config)")

	fs.Usage = func() {
		fmt.Println(`Update the gRPC endpoint in the config file without re-registration.

Usage:
  agentsmesh-runner update-endpoint [options]

Options:`)
		fs.PrintDefaults()
		fmt.Println(`
Queries the server's discovery endpoint and updates grpc_endpoint in the config
file if the server's current endpoint differs from what is stored locally.

Use this when the server's gRPC port or hostname has changed and the runner
can no longer connect. This avoids a full re-registration.`)
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

	if _, err := os.Stat(cfgFile); os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, "Error: Runner not registered. Please run 'agentsmesh-runner register' first.")
		os.Exit(1)
	}

	// Load config to get server_url and current grpc_endpoint
	cfg, err := config.Load(cfgFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Allow overriding server URL
	if *serverURL != "" {
		cfg.ServerURL = *serverURL
	}

	if cfg.ServerURL == "" {
		fmt.Fprintln(os.Stderr, "Error: server_url is not configured. Use --server-url to specify it.")
		os.Exit(1)
	}

	fmt.Printf("Querying discovery endpoint: %s/api/v1/runners/grpc/discovery\n", cfg.ServerURL)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	newEndpoint, err := client.DiscoverGRPCEndpoint(ctx, cfg.ServerURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Discovery failed: %v\n", err)
		os.Exit(1)
	}

	if newEndpoint == cfg.GRPCEndpoint {
		fmt.Printf("✓ Endpoint is already up to date: %s\n", newEndpoint)
		return
	}

	fmt.Printf("  Old endpoint: %s\n", cfg.GRPCEndpoint)
	fmt.Printf("  New endpoint: %s\n", newEndpoint)

	if err := config.UpdateGRPCEndpointInFile(cfgFile, newEndpoint); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to update config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Updated grpc_endpoint in %s\n", cfgFile)
	fmt.Println("\nYou can now start the runner with:")
	fmt.Println("  agentsmesh-runner run")
}
