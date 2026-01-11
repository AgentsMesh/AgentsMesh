//go:build nocgo

// Package tray provides system tray functionality for the desktop mode.
// This file is a stub for builds without CGO support.
package tray

import (
	"fmt"
	"os"

	"github.com/anthropics/agentmesh/runner/internal/config"
)

// TrayApp represents the system tray application.
// This is a stub implementation for non-CGO builds.
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

// Run prints an error message since desktop mode is not available without CGO.
func (t *TrayApp) Run() {
	fmt.Println("Desktop mode is not available in this build.")
	fmt.Println("")
	fmt.Println("This binary was compiled without CGO support.")
	fmt.Println("Desktop mode requires CGO for system tray functionality.")
	fmt.Println("")
	fmt.Println("To enable desktop mode, rebuild with CGO:")
	fmt.Println("  CGO_ENABLED=1 go build -o runner ./cmd/runner")
	fmt.Println("")
	fmt.Println("Or use the Makefile:")
	fmt.Println("  make build")
	fmt.Println("")
	fmt.Println("For CLI-only usage, use 'runner run' instead.")
	os.Exit(1)
}
