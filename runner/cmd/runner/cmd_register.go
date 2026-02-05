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
	fmt.Println("gRPC/mTLS Registration successful!")
}
