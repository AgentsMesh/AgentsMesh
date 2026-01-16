//go:build !desktop

// Package tray provides system tray functionality for the desktop mode.
// This file is a stub for CLI builds without desktop support.
package tray

import (
	"fmt"
	"os"

	"github.com/anthropics/agentmesh/runner/internal/config"
)

// TrayApp represents the system tray application.
// This is a stub implementation for CLI builds.
type TrayApp struct {
	cfg *config.Config
}

// New creates a new tray application.
func New(cfg *config.Config) *TrayApp {
	return &TrayApp{cfg: cfg}
}

// NewWithVersion creates a new tray application with version info.
func NewWithVersion(cfg *config.Config, version string) *TrayApp {
	return &TrayApp{cfg: cfg}
}

// Run prints an error message since desktop mode is not available in CLI builds.
func (t *TrayApp) Run() {
	fmt.Println("Desktop mode is not available in this build.")
	fmt.Println("")
	fmt.Println("This is the CLI version of AgentsMesh Runner.")
	fmt.Println("Desktop mode with system tray requires the Desktop build.")
	fmt.Println("")
	fmt.Println("For CLI usage, use 'runner run' to start the runner.")
	fmt.Println("Use 'runner webconsole' to open the web console in browser.")
	os.Exit(1)
}
