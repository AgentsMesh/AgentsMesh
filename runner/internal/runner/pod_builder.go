package runner

import (
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/clipboard"
	"github.com/anthropics/agentsmesh/runner/internal/config"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/vt"
	"github.com/anthropics/agentsmesh/runner/internal/workspace"
)

// PodBuilderDeps defines the dependencies for PodBuilder.
// This decouples PodBuilder from the Runner struct, following DIP.
type PodBuilderDeps struct {
	// Config provides workspace configuration.
	Config *config.Config

	// Workspace provides git worktree management.
	// Can be nil if git operations are not needed.
	Workspace workspace.WorkspaceManagerInterface

	// ProgressSender sends pod initialization progress.
	// Can be nil if progress reporting is not needed.
	ProgressSender client.ProgressSender

	// Clipboard provides clipboard backend for image paste support.
	// Can be nil; if nil, clipboard support is disabled.
	Clipboard clipboard.Backend
}

// PodBuilder builds pods using the Builder pattern.
// It provides a fluent API for configuring and creating pods.
// Uses Proto types directly for zero-copy message passing.
type PodBuilder struct {
	deps PodBuilderDeps

	// Pod command (Proto type)
	cmd *runnerv1.CreatePodCommand

	// Terminal configuration
	rows int
	cols int

	// VirtualTerminal configuration
	vtHistoryLimit int

	// PTY logging configuration
	enablePTYLogging bool
	ptyLogDir        string

	// OSC handler (called when OSC sequences are received)
	oscHandler vt.OSCHandler

	// Setup strategies (Strategy Pattern for OCP compliance)
	// Strategies are tried in order; first matching strategy is used.
	setupStrategies []SetupStrategy
}

// NewPodBuilder creates a new pod builder with explicit dependencies.
// This is the preferred constructor for new code.
func NewPodBuilder(deps PodBuilderDeps) *PodBuilder {
	b := &PodBuilder{
		deps:           deps,
		rows:           24,
		cols:           80,
		vtHistoryLimit: 100, // Default scrollback history
	}
	// Register default setup strategies
	b.setupStrategies = DefaultSetupStrategies(b)
	return b
}

// NewPodBuilderFromRunner creates a new pod builder from a Runner instance.
// This maintains backward compatibility with existing code.
func NewPodBuilderFromRunner(runner *Runner) *PodBuilder {
	deps := PodBuilderDeps{
		Config:         runner.cfg,
		ProgressSender: runner.conn,
		Clipboard:      runner.clipboard,
	}
	// Explicitly set Workspace only if not nil to avoid interface nil comparison issues
	if runner.workspace != nil {
		deps.Workspace = runner.workspace
	}
	return NewPodBuilder(deps)
}

// WithCommand sets the create pod command (Proto type).
// This is the primary way to configure the pod.
func (b *PodBuilder) WithCommand(cmd *runnerv1.CreatePodCommand) *PodBuilder {
	b.cmd = cmd
	return b
}

// WithSetupStrategies allows customizing the setup strategies.
// This is useful for testing or extending with custom strategies.
// Strategies are tried in order; first matching strategy is used.
func (b *PodBuilder) WithSetupStrategies(strategies []SetupStrategy) *PodBuilder {
	b.setupStrategies = strategies
	return b
}

// WithTerminalSize sets terminal dimensions.
// Parameters follow the standard convention: cols (width) first, then rows (height).
// This matches xterm.js, ANSI standards, and most terminal libraries.
func (b *PodBuilder) WithTerminalSize(cols, rows int) *PodBuilder {
	if cols > 0 {
		b.cols = cols
	}
	if rows > 0 {
		b.rows = rows
	}
	return b
}

// WithPTYLogging enables PTY logging to the specified directory.
func (b *PodBuilder) WithPTYLogging(logDir string) *PodBuilder {
	b.enablePTYLogging = true
	b.ptyLogDir = logDir
	return b
}

// WithOSCHandler sets the handler for OSC (Operating System Command) sequences.
func (b *PodBuilder) WithOSCHandler(handler vt.OSCHandler) *PodBuilder {
	b.oscHandler = handler
	return b
}

// WithVirtualTerminalHistoryLimit sets the scrollback history limit.
func (b *PodBuilder) WithVirtualTerminalHistoryLimit(limit int) *PodBuilder {
	if limit > 0 {
		b.vtHistoryLimit = limit
	}
	return b
}

// Note: Build method is in pod_builder_build.go
// Note: setup, setupGitWorktree, createFiles etc. are in pod_builder_setup.go and pod_builder_git.go
