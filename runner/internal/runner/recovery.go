package runner

import (
	"fmt"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/poddaemon"
	"github.com/anthropics/agentsmesh/runner/internal/terminal"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/aggregator"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/vt"
)

// recoverDaemonSessions scans for surviving daemon processes from a previous
// Runner lifecycle and rebuilds their Pod objects in the pod store.
// Recovered pods will be reported in heartbeat, triggering backend's
// orphaned → running recovery path.
func (r *Runner) recoverDaemonSessions() {
	log := logger.Runner()

	if r.podDaemonManager == nil {
		return
	}

	states, err := r.podDaemonManager.RecoverSessions()
	if err != nil {
		log.Error("failed to scan for recoverable sessions", "error", err)
		return
	}
	if len(states) == 0 {
		return
	}

	log.Info("found recoverable daemon sessions", "count", len(states))

	for _, state := range states {
		pod, err := r.recoverSingleSession(state)
		if err != nil {
			log.Warn("failed to recover session, cleaning up",
				"pod_key", state.PodKey, "error", err)
			r.podDaemonManager.CleanupSession(state.SandboxPath)
			continue
		}

		r.podStore.Put(pod.PodKey, pod)
		log.Info("session recovered",
			"pod_key", pod.PodKey,
			"pid", pod.Terminal.PID(),
			"sandbox", pod.SandboxPath)
	}
}

// recoverSingleSession re-attaches to a surviving daemon and rebuilds its Pod.
func (r *Runner) recoverSingleSession(state *poddaemon.PodDaemonState) (*Pod, error) {
	// Attach to daemon via IPC
	dpty, err := r.podDaemonManager.AttachSession(state)
	if err != nil {
		return nil, fmt.Errorf("attach to daemon: %w", err)
	}

	// Wrap daemonPTY in a PTYFactory for Terminal
	ptyFactory := func(command string, args []string, workDir string, env []string, cols, rows int) (terminal.PtyProcess, error) {
		return dpty, nil
	}

	// Create Terminal with pre-connected daemon PTY
	term, err := terminal.New(terminal.Options{
		Command:    state.Command,
		Args:       state.Args,
		WorkDir:    state.WorkDir,
		Rows:       state.Rows,
		Cols:       state.Cols,
		Label:      state.PodKey,
		PTYFactory: ptyFactory,
	})
	if err != nil {
		dpty.Close()
		return nil, fmt.Errorf("create terminal: %w", err)
	}

	// Create VirtualTerminal and Aggregator (fresh state after recovery)
	virtualTerm := vt.NewVirtualTerminal(state.Cols, state.Rows, state.VTHistoryLimit)
	agg := aggregator.NewSmartAggregator(nil, nil,
		aggregator.WithFullRedrawThrottling(),
	)

	// Build Pod
	pod := &Pod{
		ID:            state.PodKey,
		PodKey:        state.PodKey,
		AgentType:     state.AgentType,
		RepositoryURL: state.RepositoryURL,
		Branch:        state.Branch,
		SandboxPath:   state.SandboxPath,
		LaunchCommand: state.Command,
		LaunchArgs:    state.Args,
		WorkDir:       state.WorkDir,
		TicketSlug:    state.TicketSlug,
		Terminal:      term,
		VirtualTerminal: virtualTerm,
		Aggregator:    agg,
		StartedAt:     state.StartedAt,
		Status:        PodStatusRunning,
	}

	// Wire up output handler (same pipeline as OnCreatePod)
	podKey := state.PodKey
	term.SetOutputHandler(func(data []byte) {
		defer func() {
			if rec := recover(); rec != nil {
				logger.Terminal().Error("PANIC in recovered OutputHandler",
					"pod_key", podKey, "panic", fmt.Sprintf("%v", rec))
			}
		}()

		var screenLines []string
		if virtualTerm != nil {
			screenLines = virtualTerm.Feed(data)
		}
		go pod.NotifyStateDetectorWithScreen(len(data), screenLines)
		agg.Write(data)
	})

	// Set exit handler
	term.SetExitHandler(r.messageHandler.createExitHandler(podKey))

	// Start Terminal I/O (readOutput + waitExit goroutines)
	if err := term.Start(); err != nil {
		dpty.Close()
		return nil, fmt.Errorf("start terminal: %w", err)
	}

	// Register with MCP and monitor
	if mcpSrv := r.GetMCPServer(); mcpSrv != nil {
		mcpSrv.RegisterPod(podKey, r.conn.GetOrgSlug(), nil, nil, state.AgentType)
	}
	if agentMon := r.GetAgentMonitor(); agentMon != nil {
		agentMon.RegisterPod(podKey, term.PID())
	}

	return pod, nil
}

// recoverSessionStartedAt parses the started_at field, falling back to now.
func recoverSessionStartedAt(startedAt time.Time) time.Time {
	if startedAt.IsZero() {
		return time.Now()
	}
	return startedAt
}
