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
		return fmt.Errorf("failed to create service: %w", err)
	}

	err = s.Install()
	if err != nil {
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
		return fmt.Errorf("failed to create service: %w", err)
	}

	err = s.Uninstall()
	if err != nil {
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
		return fmt.Errorf("failed to create service: %w", err)
	}

	err = s.Start()
	if err != nil {
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
		return fmt.Errorf("failed to create service: %w", err)
	}

	err = s.Stop()
	if err != nil {
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
		return fmt.Errorf("failed to create service: %w", err)
	}

	err = s.Restart()
	if err != nil {
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
		return service.StatusUnknown, fmt.Errorf("failed to create service: %w", err)
	}

	status, err := s.Status()
	if err != nil {
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

// RestartForUpdate restarts the service after an update.
// This should be called after the binary has been replaced.
func RestartForUpdate() error {
	prg := &Program{}
	s, err := service.New(prg, ServiceConfig())
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	// Check if running as a service
	if !service.Interactive() {
		// Restart the service
		err = s.Restart()
		if err != nil {
			return fmt.Errorf("failed to restart service: %w", err)
		}
		log.Info("Service restarted for update")
		return nil
	}

	// If running interactively, just log and exit
	log.Info("Update applied. Please restart the runner manually.")
	return nil
}

// ScheduleRestartOnExit schedules a restart when the process exits.
// This is useful for graceful updates where we want to restart after the update is applied.
func ScheduleRestartOnExit() {
	// In service mode, the service manager will automatically restart the process
	// after it exits (if configured to do so).
	// For interactive mode, we just exit and let the user restart manually.
	log.Info("Update complete. Process will exit for restart.")
}

// RestartFunc returns a function that can be used to restart the service.
// This is designed to be passed to the graceful updater.
// Returns pid=0 because the service manager spawns the new process and we
// don't have its PID. GracefulUpdater.applyPendingUpdate() only runs the
// health check when pid > 0, so this is safe.
func RestartFunc() func() (int, error) {
	return func() (int, error) {
		return 0, RestartForUpdate()
	}
}
