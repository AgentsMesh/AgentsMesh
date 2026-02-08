package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"
)

func runRegister(args []string) {
	fs := flag.NewFlagSet("register", flag.ExitOnError)
	serverURL := fs.String("server", "https://agentsmesh.ai", "AgentsMesh server URL (default: https://agentsmesh.ai)")
	token := fs.String("token", "", "Registration token (for token-based registration)")
	nodeID := fs.String("node-id", "", "Node ID for this runner (default: hostname)")
	headless := fs.Bool("headless", false, "Run without opening browser automatically (for SSH/remote sessions)")

	fs.Usage = func() {
		fmt.Println(`Register this runner with the AgentsMesh server using gRPC/mTLS.

Usage:
  runner register [options]

Examples:
  runner register                    # Interactive login (opens browser)
  runner register --headless         # Interactive without browser (for SSH)
  runner register --token <token>    # Token-based registration
  runner register --server <url>     # Self-hosted server

Options:
  --server <url>     Server URL (default: https://agentsmesh.ai)
  --token <token>    Registration token for automated deployment
  --node-id <id>     Runner node ID (default: hostname)
  --headless         Don't open browser (for SSH/remote sessions)

After successful registration, certificates and configuration will be saved to ~/.agentsmesh/`)
	}

	if err := fs.Parse(args); err != nil {
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
		if err := registerInteractive(ctx, *serverURL, nID, *headless); err != nil {
			fmt.Fprintf(os.Stderr, "Registration failed: %v\n", err)
			os.Exit(1)
		}
	}
	fmt.Println("gRPC/mTLS Registration successful!")
}
