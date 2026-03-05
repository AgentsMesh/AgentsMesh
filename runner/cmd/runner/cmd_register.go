package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

func runRegister(args []string) {
	fs := flag.NewFlagSet("register", flag.ExitOnError)
	serverURL := fs.String("server", "https://agentsmesh.ai", "AgentsMesh server URL (default: https://agentsmesh.ai)")
	token := fs.String("token", "", "Registration token (for token-based registration)")
	nodeID := fs.String("node-id", "", "Node ID for this runner (default: hostname)")
	headless := fs.Bool("headless", false, "Run without opening browser automatically (for SSH/remote sessions)")
	force := fs.Bool("force", false, "Skip confirmation when overwriting existing registration")

	fs.Usage = func() {
		fmt.Println(`Register this runner with the AgentsMesh server using gRPC/mTLS.

Usage:
  agentsmesh-runner register [options]

Examples:
  agentsmesh-runner register                    # Interactive login (opens browser)
  agentsmesh-runner register --headless         # Interactive without browser (for SSH)
  agentsmesh-runner register --token <token>    # Token-based registration
  agentsmesh-runner register --server <url>     # Self-hosted server
  agentsmesh-runner register --force            # Overwrite existing registration without confirmation

Options:
  --server <url>     Server URL (default: https://agentsmesh.ai)
  --token <token>    Registration token for automated deployment
  --node-id <id>     Runner node ID (default: hostname)
  --headless         Don't open browser (for SSH/remote sessions)
  --force            Skip confirmation when overwriting existing registration`)
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

	// Check for existing registration and warn user
	if existingOrg := checkExistingRegistration(); existingOrg != "" {
		fmt.Printf("\n⚠️  WARNING: This runner is already registered to organization '%s'.\n", existingOrg)
		fmt.Println("   Re-registering will overwrite the existing configuration and certificates.")
		fmt.Println("   The old configuration will be backed up automatically.")
		fmt.Println()
		if !*force {
			if *token != "" {
				// Token-based registration requires --force to overwrite
				fmt.Fprintln(os.Stderr, "Use --force to overwrite existing registration in token mode.")
				os.Exit(1)
			}
			// Interactive mode: prompt for confirmation
			fmt.Print("Continue? [y/N]: ")
			var answer string
			fmt.Scanln(&answer)
			if answer != "y" && answer != "Y" && answer != "yes" && answer != "Yes" {
				fmt.Println("Registration cancelled.")
				os.Exit(0)
			}
		}
	}

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

// checkExistingRegistration checks if there's an existing registration config.
// Returns the org slug if found, empty string otherwise.
func checkExistingRegistration() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	configFile := filepath.Join(home, ".agentsmesh", "config.yaml")
	data, err := os.ReadFile(configFile)
	if err != nil {
		return ""
	}

	// Parse just the org_slug field
	var cfg struct {
		OrgSlug string `yaml:"org_slug"`
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return ""
	}

	return cfg.OrgSlug
}
