// Package service provides system service integration for the runner.
// Supports Windows Service, macOS LaunchDaemon, and Linux systemd.
package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"sync"

	"github.com/kardianos/service"

	"github.com/anthropics/agentsmesh/runner/internal/config"
	"github.com/anthropics/agentsmesh/runner/internal/envpath"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/pidfile"
	"github.com/anthropics/agentsmesh/runner/internal/runner"
)

// Module logger for service
var log = logger.Service()

const (
	ServiceName        = "agentsmesh-runner"
	ServiceDisplayName = "AgentsMesh Runner"
	ServiceDescription = "AgentsMesh Runner - executes AI agent tasks"
)

// Program implements the service.Interface for running as a system service.
type Program struct {
	cfg        *config.Config
	runner     *runner.Runner
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	statusChan chan Status
}

// Status represents the current runner status.
type Status struct {
	Running   bool
	Connected bool
	Error     error
}

// NewProgram creates a new service program instance.
func NewProgram(cfg *config.Config) *Program {
	return &Program{
		cfg:        cfg,
		statusChan: make(chan Status, 1),
	}
}

// Start is called when the service is started.
func (p *Program) Start(s service.Service) error {
	log.Info("Service starting")

	// Clean up stale process (non-fatal in service mode — manager will retry)
	if err := pidfile.CleanupStaleProcess(); err != nil {
		log.Warn("Failed to clean up stale process", "error", err)
	}

	// Write PID file
	if err := pidfile.Write(); err != nil {
		log.Warn("Failed to write PID file", "error", err)
	}

	// Create runner instance
	r, err := runner.New(p.cfg)
	if err != nil {
		p.sendStatus(Status{Running: false, Error: err})
		return fmt.Errorf("failed to create runner: %w", err)
	}
	p.runner = r

	// Create cancellable context
	p.ctx, p.cancel = context.WithCancel(context.Background())

	// Start runner in background
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		defer func() {
			if r := recover(); r != nil {
				log.Error("Runner panic recovered in service mode, exiting for restart",
					"panic", fmt.Sprintf("%v", r),
					"stack", string(debug.Stack()),
				)
				p.sendStatus(Status{Running: false, Error: fmt.Errorf("panic: %v", r)})
				os.Exit(1) // Let the service manager restart the process
			}
		}()
		p.sendStatus(Status{Running: true, Connected: true})

		if err := p.runner.Run(p.ctx); err != nil {
			log.Error("Runner error", "error", err)
			p.sendStatus(Status{Running: false, Error: err})
		}
	}()

	return nil
}

// Stop is called when the service is stopped.
func (p *Program) Stop(s service.Service) error {
	log.Info("Service stopping")

	if p.cancel != nil {
		p.cancel()
	}

	// Wait for runner to stop before removing PID file.
	// If we remove PID file first, another instance could start during
	// shutdown, fail to detect the still-running process, and hit port conflicts.
	p.wg.Wait()

	pidfile.Remove()

	p.sendStatus(Status{Running: false})
	log.Info("Service stopped")
	return nil
}

// StatusChan returns a channel for receiving status updates.
func (p *Program) StatusChan() <-chan Status {
	return p.statusChan
}

func (p *Program) sendStatus(status Status) {
	select {
	case p.statusChan <- status:
	default:
		// Non-blocking send, drop if channel is full
	}
}

// ServiceConfig returns the service configuration.
// Uses UserService option so the plist is installed to ~/Library/LaunchAgents/
// instead of /Library/LaunchDaemons/ (which requires root on macOS).
func ServiceConfig() *service.Config {
	return &service.Config{
		Name:        ServiceName,
		DisplayName: ServiceDisplayName,
		Description: ServiceDescription,
		Option: service.KeyValue{
			"UserService": true,
			// macOS launchd: auto-restart on crash
			"KeepAlive":        true,
			"ThrottleInterval": 10,
			// Linux systemd: auto-restart on failure
			"Restart":               "on-failure",
			"RestartSec":            "10",
			"StartLimitBurst":       "5",
			"StartLimitIntervalSec": "60",
			// Linux systemd: watchdog integration (WatchdogService sends heartbeats)
			"WatchdogSec": "60",
		},
	}
}

// GetService returns a service instance for the given program.
func GetService(prg *Program) (service.Service, error) {
	return service.New(prg, ServiceConfig())
}

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

// Status returns the current service status.
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
