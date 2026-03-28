package service

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kardianos/service"

	"github.com/anthropics/agentsmesh/runner/internal/envpath"
)

// Install installs the runner as a system service.
func Install(configPath string) error {
	cfg := ServiceConfig()

	// Set executable path
	execPath, err := os.Executable()
	if err != nil {
		log.Error("Failed to get executable path", "error", err)
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	cfg.Executable = execPath

	// Set arguments to run with config
	if configPath != "" {
		cfg.Arguments = []string{"run", "--config", configPath}
	} else {
		cfg.Arguments = []string{"run"}
	}

	// Capture current PATH so that the service inherits user-installed binaries
	// (e.g. ~/.local/bin, /opt/homebrew/bin). Without this, launchd/systemd
	// starts with a minimal PATH that cannot find agent commands like "claude".
	cfg.EnvVars = buildServiceEnvVars()

	// Create a minimal program for installation
	prg := &Program{}
	s, err := service.New(prg, cfg)
	if err != nil {
		log.Error("Failed to create service for install", "error", err)
		return fmt.Errorf("failed to create service: %w", err)
	}

	err = s.Install()
	if err != nil {
		log.Error("Failed to install service", "error", err)
		return fmt.Errorf("failed to install service: %w", err)
	}

	log.Info("Service installed successfully")
	return nil
}

// buildServiceEnvVars constructs the environment variables for the service.
// It captures the current PATH and ensures common user binary directories are included.
func buildServiceEnvVars() map[string]string {
	envVars := make(map[string]string)

	// Start with the current shell PATH (richest source of user-installed dirs)
	currentPath := os.Getenv("PATH")

	// Prepend common user binary directories (only if directory actually exists)
	extraDirs := envpath.UserBinaryDirs()
	var existingDirs []string
	for _, dir := range extraDirs {
		if _, err := os.Stat(dir); err == nil {
			existingDirs = append(existingDirs, dir)
		}
	}
	envVars["PATH"] = envpath.PrependToPath(currentPath, existingDirs...)

	return envVars
}

// Uninstall removes the runner system service.
func Uninstall() error {
	prg := &Program{}
	s, err := service.New(prg, ServiceConfig())
	if err != nil {
		log.Error("Failed to create service for uninstall", "error", err)
		return fmt.Errorf("failed to create service: %w", err)
	}

	err = s.Uninstall()
	if err != nil {
		log.Error("Failed to uninstall service", "error", err)
		return fmt.Errorf("failed to uninstall service: %w", err)
	}

	log.Info("Service uninstalled successfully")
	return nil
}

// Start starts the system service.
func Start() error {
	prg := &Program{}
	s, err := service.New(prg, ServiceConfig())
	if err != nil {
		log.Error("Failed to create service for start", "error", err)
		return fmt.Errorf("failed to create service: %w", err)
	}

	err = s.Start()
	if err != nil {
		log.Error("Failed to start service", "error", err)
		return fmt.Errorf("failed to start service: %w", err)
	}

	log.Info("Service started")
	return nil
}

// Stop stops the system service.
func Stop() error {
	prg := &Program{}
	s, err := service.New(prg, ServiceConfig())
	if err != nil {
		log.Error("Failed to create service for stop", "error", err)
		return fmt.Errorf("failed to create service: %w", err)
	}

	err = s.Stop()
	if err != nil {
		log.Error("Failed to stop service", "error", err)
		return fmt.Errorf("failed to stop service: %w", err)
	}

	log.Info("Service stopped")
	return nil
}

// Restart restarts the system service.
func Restart() error {
	prg := &Program{}
	s, err := service.New(prg, ServiceConfig())
	if err != nil {
		log.Error("Failed to create service for restart", "error", err)
		return fmt.Errorf("failed to create service: %w", err)
	}

	err = s.Restart()
	if err != nil {
		log.Error("Failed to restart service", "error", err)
		return fmt.Errorf("failed to restart service: %w", err)
	}

	log.Info("Service restarted")
	return nil
}

// GetStatus returns the current service status.
func GetStatus() (service.Status, error) {
	prg := &Program{}
	s, err := service.New(prg, ServiceConfig())
	if err != nil {
		log.Error("Failed to create service for status check", "error", err)
		return service.StatusUnknown, fmt.Errorf("failed to create service: %w", err)
	}

	status, err := s.Status()
	if err != nil {
		log.Error("Failed to get service status", "error", err)
		return service.StatusUnknown, fmt.Errorf("failed to get status: %w", err)
	}

	return status, nil
}

// IsInteractive returns true if the service is running interactively.
func IsInteractive() bool {
	return service.Interactive()
}

// GetDefaultConfigPath returns the default config file path.
func GetDefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".agentsmesh", "config.yaml")
}
