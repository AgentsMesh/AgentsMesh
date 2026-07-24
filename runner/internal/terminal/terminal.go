package terminal

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/envfilter"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/safego"
)

const (
	// gracefulStopTimeout is the maximum time to wait for the process to exit
	// after sending SIGTERM before escalating to SIGKILL.
	gracefulStopTimeout = 5 * time.Second
)

// New creates a new terminal instance.
func New(opts Options) (*Terminal, error) {
	if opts.Command == "" {
		return nil, fmt.Errorf("command is required")
	}

	// Build environment with proper deduplication.
	// Using a map prevents duplicate keys (e.g., TERM appearing twice)
	// which can confuse some programs.
	// Filter Runner-internal vars to prevent leakage to child processes.
	envMap := make(map[string]string)
	for _, e := range envfilter.FilterEnv(os.Environ()) {
		if idx := strings.Index(e, "="); idx >= 0 {
			envMap[e[:idx]] = e[idx+1:]
		}
	}
	// Remove CLAUDECODE to prevent nested session detection when running
	// Claude Code inside a pod - the runner intentionally spawns claude sessions.
	delete(envMap, "CLAUDECODE")
	// Ensure terminal supports colors (critical for CLI tools like claude, ls, etc.)
	envMap["TERM"] = "xterm-256color"
	envMap["COLORTERM"] = "truecolor"
	// Apply user-specified env vars (highest priority)
	for k, v := range opts.Env {
		envMap[k] = v
	}
	env := make([]string, 0, len(envMap))
	for k, v := range envMap {
		env = append(env, k+"="+v)
	}

	// Default terminal size if not specified
	rows := opts.Rows
	cols := opts.Cols
	if rows <= 0 {
		rows = 24
	}
	if cols <= 0 {
		cols = 80
	}

	logger.Terminal().Debug("Terminal instance created",
		"command", opts.Command,
		"work_dir", opts.WorkDir,
		"cols", cols,
		"rows", rows)

	return &Terminal{
		command:        opts.Command,
		args:           opts.Args,
		workDir:        opts.WorkDir,
		env:            env,
		label:          opts.Label,
		ptyFactory:     opts.PTYFactory,
		onOutput:       opts.OnOutput,
		onExit:         opts.OnExit,
		rows:           rows,
		cols:           cols,
		doneCh:         make(chan struct{}),
		readDoneCh:     make(chan struct{}),
		readActiveCh:   make(chan struct{}),
		readProgressCh: make(chan struct{}, 1),
		resumeCh:       make(chan struct{}, 1), // Buffered to avoid blocking
	}, nil
}

// Start starts the terminal process
func (t *Terminal) Start() error {
	t.lifecycleMu.Lock()
	defer t.lifecycleMu.Unlock()
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed || t.stopping {
		return fmt.Errorf("terminal is closed")
	}
	if t.proc != nil || t.readStarted {
		return fmt.Errorf("terminal is already started")
	}

	log := logger.Terminal()
	log.Debug("Starting command", "command", t.command, "args", t.args, "dir", t.workDir, "cols", t.cols, "rows", t.rows)

	// Start with PTY and initial size (custom factory or platform-specific default)
	var proc ptyProcess
	var err error
	if t.ptyFactory != nil {
		proc, err = t.ptyFactory(t.command, t.args, t.workDir, t.env, t.cols, t.rows)
	} else {
		proc, err = startPTY(t.command, t.args, t.workDir, t.env, t.cols, t.rows)
	}
	if err != nil {
		return fmt.Errorf("failed to start pty: %w", err)
	}
	t.proc = proc
	t.readStarted = true

	log.Debug("PTY started", "pid", t.proc.Pid(), "cols", t.cols, "rows", t.rows)

	// Start output reader
	safego.Go("pty-read", t.readOutput)

	// Wait for process exit
	safego.Go("pty-wait", t.waitExit)

	log.Info("Terminal started", "pid", t.proc.Pid(), "cols", t.cols, "rows", t.rows)

	return nil
}
