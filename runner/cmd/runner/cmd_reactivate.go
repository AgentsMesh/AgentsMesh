package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/config"
)

func runReactivate(args []string) {
	fs := flag.NewFlagSet("reactivate", flag.ExitOnError)
	serverURL := fs.String("server", "", "AgentsMesh server URL (default: from config)")
	token := fs.String("token", "", "Reactivation token from the web UI")

	fs.Usage = func() {
		fmt.Println(`Reactivate a runner with an expired certificate.

Usage:
  agentsmesh-runner reactivate --token <reactivation-token>

Options:`)
		fs.PrintDefaults()
		fmt.Println(`
When your runner's certificate expires (after long periods of inactivity),
you can generate a reactivation token from the web UI:

1. Go to Runner management page
2. Find your runner and click "Reactivate"
3. Copy the generated token
4. Run: agentsmesh-runner reactivate --token <token>

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
