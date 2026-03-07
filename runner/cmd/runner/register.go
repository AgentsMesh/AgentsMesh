package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/config"
	"github.com/anthropics/agentsmesh/runner/internal/envpath"
	"gopkg.in/yaml.v3"
)

// ==================== gRPC/mTLS Registration ====================

// generateMachineKey generates a unique machine key for interactive registration.
func generateMachineKey() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based key
		return fmt.Sprintf("runner-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

// registerInteractive performs Tailscale-style interactive registration.
// If headless is true, the browser will not be opened automatically (for SSH/remote sessions).
func registerInteractive(ctx context.Context, serverURL, nodeID string, headless bool) error {
	machineKey := generateMachineKey()

	result, err := client.InteractiveRegister(ctx, client.InteractiveRegistrationRequest{
		ServerURL:  serverURL,
		MachineKey: machineKey,
		NodeID:     nodeID,
		Headless:   headless,
	})
	if err != nil {
		return fmt.Errorf("interactive registration failed: %w", err)
	}

	// Save certificates and configuration
	if err := saveGRPCConfig(nodeID, serverURL, result.OrgSlug, result.Certificate, result.PrivateKey, result.CACertificate, result.GRPCEndpoint); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Printf("✓ Organization: %s\n", result.OrgSlug)
	fmt.Printf("✓ gRPC Endpoint: %s\n", result.GRPCEndpoint)
	fmt.Printf("✓ Certificates saved to ~/.agentsmesh/certs/\n")
	fmt.Println("\nYou can now start the runner with:")
	fmt.Println("  agentsmesh-runner run")

	return nil
}

// registerWithGRPCToken performs token-based gRPC registration.
func registerWithGRPCToken(ctx context.Context, serverURL, token, nodeID string) error {
	result, err := client.RegisterWithToken(ctx, client.TokenRegistrationRequest{
		ServerURL: serverURL,
		Token:     token,
		NodeID:    nodeID,
	})
	if err != nil {
		return fmt.Errorf("token registration failed: %w", err)
	}

	// Save certificates and configuration
	if err := saveGRPCConfig(nodeID, serverURL, result.OrgSlug, result.Certificate, result.PrivateKey, result.CACertificate, result.GRPCEndpoint); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Printf("✓ Organization: %s\n", result.OrgSlug)
	fmt.Printf("✓ gRPC Endpoint: %s\n", result.GRPCEndpoint)
	fmt.Printf("✓ Certificates saved to ~/.agentsmesh/certs/\n")
	fmt.Println("\nYou can now start the runner with:")
	fmt.Println("  agentsmesh-runner run")

	return nil
}

// reactivateRunner reactivates a runner with an expired certificate.
func reactivateRunner(ctx context.Context, serverURL, token string) error {
	result, err := client.Reactivate(ctx, client.ReactivationRequest{
		ServerURL: serverURL,
		Token:     token,
	})
	if err != nil {
		return fmt.Errorf("reactivation failed: %w", err)
	}

	// Load existing config to get certificate paths
	cfg := &config.Config{}
	if err := cfg.LoadGRPCConfig(); err != nil {
		return fmt.Errorf("failed to load existing config: %w", err)
	}

	// Save new certificates
	if err := cfg.SaveCertificates([]byte(result.Certificate), []byte(result.PrivateKey), []byte(result.CACertificate)); err != nil {
		return fmt.Errorf("failed to save certificates: %w", err)
	}

	fmt.Println("✓ Runner reactivated successfully!")
	fmt.Println("✓ New certificates saved to ~/.agentsmesh/certs/")
	fmt.Println("\nYou can now start the runner with:")
	fmt.Println("  agentsmesh-runner run")

	return nil
}

// backupExistingConfig backs up existing configuration and certificates before overwriting.
// Files are backed up as config.<org-slug>.yaml.bak and certs.<org-slug>.bak/
func backupExistingConfig(configDir string) {
	configFile := filepath.Join(configDir, "config.yaml")
	data, err := os.ReadFile(configFile)
	if err != nil {
		return // No existing config to backup
	}

	// Parse org_slug from existing config
	var cfg struct {
		OrgSlug string `yaml:"org_slug"`
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil || cfg.OrgSlug == "" {
		return
	}

	// Backup config file
	backupConfigFile := filepath.Join(configDir, fmt.Sprintf("config.%s.yaml.bak", cfg.OrgSlug))
	if err := os.WriteFile(backupConfigFile, data, 0600); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to backup config: %v\n", err)
	} else {
		fmt.Printf("  Backed up config to %s\n", backupConfigFile)
	}

	// Backup certs directory
	certsDir := filepath.Join(configDir, "certs")
	backupCertsDir := filepath.Join(configDir, fmt.Sprintf("certs.%s.bak", cfg.OrgSlug))
	if _, err := os.Stat(certsDir); err == nil {
		// Remove old backup if exists
		os.RemoveAll(backupCertsDir)
		if err := os.Rename(certsDir, backupCertsDir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to backup certs: %v\n", err)
		} else {
			fmt.Printf("  Backed up certs to %s\n", backupCertsDir)
		}
	}
}

// saveGRPCConfig saves gRPC registration result to ~/.agentsmesh/
func saveGRPCConfig(nodeID, serverURL, orgSlug, certPEM, keyPEM, caCertPEM, grpcEndpoint string) error {
	// Ensure config directory exists first
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(home, ".agentsmesh")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Backup existing configuration before overwriting (must happen before SaveCertificates
	// so the backup doesn't rename the freshly saved certs directory)
	backupExistingConfig(configDir)

	cfg := &config.Config{
		NodeID:       nodeID,
		ServerURL:    serverURL,
		OrgSlug:      orgSlug,
		GRPCEndpoint: grpcEndpoint,
	}

	// Save certificates
	if err := cfg.SaveCertificates([]byte(certPEM), []byte(keyPEM), []byte(caCertPEM)); err != nil {
		return fmt.Errorf("failed to save certificates: %w", err)
	}

	grpcConfig := savedGRPCConfig{
		ServerURL:         serverURL,
		NodeID:            nodeID,
		OrgSlug:           orgSlug,
		GRPCEndpoint:      grpcEndpoint,
		CertFile:          cfg.CertFile,
		KeyFile:           cfg.KeyFile,
		CAFile:            cfg.CAFile,
		MaxConcurrentPods: 5,
		WorkspaceRoot:     defaultWorkspaceRoot(),
		DefaultAgent:      "claude-code",
		DefaultShell:      getDefaultShell(),
		HealthCheckPort:   9090,
		LogLevel:          "info",
	}

	configData, err := yaml.Marshal(grpcConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	configFile := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configFile, configData, 0600); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

// getDefaultShell returns the default shell for the current platform.
func getDefaultShell() string {
	shell := os.Getenv("SHELL")
	if shell != "" {
		return shell
	}
	s, _ := envpath.ShellCommand()
	return s
}

// defaultWorkspaceRoot returns the platform-appropriate default workspace root
// for writing to config.yaml during registration.
// On Windows: consistent with config.DefaultWorkspaceRoot().
// On Unix (local user): uses os.TempDir() since /workspace may not be writable.
func defaultWorkspaceRoot() string {
	if runtime.GOOS == "windows" {
		// Delegate to config package for consistent Windows paths
		return config.DefaultWorkspaceRoot()
	}
	return filepath.Join(os.TempDir(), "agentsmesh-workspace")
}

// savedGRPCConfig represents the gRPC configuration saved to ~/.agentsmesh/config.yaml
type savedGRPCConfig struct {
	ServerURL         string `yaml:"server_url"`
	NodeID            string `yaml:"node_id"`
	OrgSlug           string `yaml:"org_slug"`
	GRPCEndpoint      string `yaml:"grpc_endpoint"`
	CertFile          string `yaml:"cert_file"`
	KeyFile           string `yaml:"key_file"`
	CAFile            string `yaml:"ca_file"`
	MaxConcurrentPods int    `yaml:"max_concurrent_pods"`
	WorkspaceRoot     string `yaml:"workspace_root"`
	DefaultAgent      string `yaml:"default_agent"`
	DefaultShell      string `yaml:"default_shell"`
	HealthCheckPort   int    `yaml:"health_check_port"`
	LogLevel          string `yaml:"log_level"`
}
