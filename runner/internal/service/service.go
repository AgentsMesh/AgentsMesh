// Package service provides system service integration for the runner.
// Supports Windows Service, macOS LaunchDaemon, and Linux systemd.
package service

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"
	"sync"

	"github.com/kardianos/service"

	"github.com/anthropics/agentsmesh/runner/internal/config"
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
	deps, err := runner.CreateDeps(p.cfg)
	if err != nil {
		p.sendStatus(Status{Running: false, Error: err})
		return fmt.Errorf("failed to create runner deps: %w", err)
	}
	r, err := runner.New(deps)
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
			// Linux systemd: auto-restart (always, not just on-failure)
			"Restart":               "always",
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

// Note: Install, Uninstall, Start, Stop, Restart, GetStatus and other management
// functions are in service_management.go
