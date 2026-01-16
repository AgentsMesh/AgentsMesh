// Package service provides system service integration for the runner.
// Supports Windows Service, macOS LaunchDaemon, and Linux systemd.
package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/kardianos/service"

	"github.com/anthropics/agentsmesh/runner/internal/config"
	"github.com/anthropics/agentsmesh/runner/internal/runner"
)

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
	log.Printf("Service starting...")

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
		p.sendStatus(Status{Running: true, Connected: true})

		if err := p.runner.Run(p.ctx); err != nil {
			log.Printf("Runner error: %v", err)
			p.sendStatus(Status{Running: false, Error: err})
		}
	}()

	return nil
}

// Stop is called when the service is stopped.
func (p *Program) Stop(s service.Service) error {
	log.Printf("Service stopping...")

	if p.cancel != nil {
		p.cancel()
	}

	// Wait for runner to stop
	p.wg.Wait()

	p.sendStatus(Status{Running: false})
	log.Printf("Service stopped")
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
func ServiceConfig() *service.Config {
	return &service.Config{
		Name:        ServiceName,
		DisplayName: ServiceDisplayName,
		Description: ServiceDescription,
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

	log.Printf("Service installed successfully")
	return nil
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

	log.Printf("Service uninstalled successfully")
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

	log.Printf("Service started")
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

	log.Printf("Service stopped")
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

	log.Printf("Service restarted")
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
