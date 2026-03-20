package runner

import (
	"fmt"

	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/config"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/poddaemon"
	"github.com/anthropics/agentsmesh/runner/internal/workspace"
)

// CreateDeps performs all I/O-heavy operations (certificate checks, workspace creation,
// gRPC connection, pod daemon setup) and returns a RunnerDeps ready for New().
// This separates I/O from pure assembly so tests can bypass it.
func CreateDeps(cfg *config.Config) (RunnerDeps, error) {
	log := logger.Runner()

	// Validate required configuration
	if cfg.OrgSlug == "" {
		return RunnerDeps{}, fmt.Errorf("org_slug is required - please re-register the runner")
	}
	if !cfg.UsesGRPC() {
		return RunnerDeps{}, fmt.Errorf("gRPC configuration is required - please re-register the runner using 'agentsmesh-runner register'")
	}

	// Create workspace manager
	ws, err := workspace.NewManager(cfg.WorkspaceRoot, cfg.GitConfigPath)
	if err != nil {
		return RunnerDeps{}, fmt.Errorf("failed to create workspace manager: %w", err)
	}

	// Create gRPC/mTLS connection
	log.Info("Using gRPC/mTLS connection", "endpoint", cfg.GRPCEndpoint)

	connOpts := []client.GRPCConnectionOption{
		client.WithGRPCServerURL(cfg.ServerURL),
		client.WithGRPCRunnerVersion(cfg.Version),
	}
	// Wire endpoint auto-discovery: persist new endpoints to config file.
	if cfg.ConfigFilePath != "" {
		cfgFile := cfg.ConfigFilePath
		connOpts = append(connOpts, client.WithGRPCEndpointChanged(func(newEndpoint string) error {
			return config.UpdateGRPCEndpointInFile(cfgFile, newEndpoint)
		}))
	}

	grpcConn := client.NewGRPCConnection(
		cfg.GRPCEndpoint, cfg.NodeID, cfg.OrgSlug,
		cfg.CertFile, cfg.KeyFile, cfg.CAFile,
		connOpts...,
	)

	// Check certificate validity before connecting
	certInfo, err := grpcConn.GetCertificateExpiryInfo()
	if err != nil {
		return RunnerDeps{}, fmt.Errorf("failed to check certificate: %w", err)
	}
	if certInfo.IsExpired {
		return RunnerDeps{}, fmt.Errorf("certificate has expired on %s. Please reactivate the runner using:\n  agentsmesh-runner reactivate --token <token>\nGet a reactivation token from the web UI",
			certInfo.ExpiresAt.Format("2006-01-02"))
	}
	if certInfo.NeedsRenewal {
		log.Warn("Certificate expires soon",
			"days_until_expiry", certInfo.DaysUntilExpiry,
			"expires_at", certInfo.ExpiresAt.Format("2006-01-02"))
	}

	// Create Pod Daemon manager for session persistence
	podDaemonMgr, err := poddaemon.NewPodDaemonManager(cfg.GetSandboxesDir(), cfg.GetSocketDir())
	if err != nil {
		log.Warn("Pod Daemon manager unavailable, sessions won't persist across restarts", "error", err)
	}

	return RunnerDeps{
		Config:           cfg,
		Connection:       grpcConn,
		Workspace:        ws,
		PodDaemonManager: podDaemonMgr,
	}, nil
}
