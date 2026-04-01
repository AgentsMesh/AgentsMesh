package runner

import (
	"fmt"

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
			_ = r.podDaemonManager.CleanupSession(state.SandboxPath)
			continue
		}

		r.podStore.Put(pod.PodKey, pod)
		log.Info("session recovered",
			"pod_key", pod.PodKey,
			"pid", pod.IO.GetPID(),
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
	virtualTerm.SetOSCHandler(r.messageHandler.createOSCHandler(state.PodKey))

	agg := aggregator.NewSmartAggregator(nil,
		aggregator.WithFullRedrawThrottling(),
	)

	// Enable PTY logging if configured (same as pod_builder_build.go)
	var ptyLogger *aggregator.PTYLogger
	cfg := r.GetConfig()
	if cfg.LogPTY {
		var logErr error
		ptyLogger, logErr = aggregator.NewPTYLogger(cfg.GetLogPTYDir(), state.PodKey)
		if logErr != nil {
			logger.Runner().Warn("Failed to create PTY logger for recovered pod",
				"pod_key", state.PodKey, "error", logErr)
		} else {
			agg.SetPTYLogger(ptyLogger)
		}
	}

	// Build Pod
	podKey := state.PodKey
	pod := &Pod{
		ID:              podKey,
		PodKey:          podKey,
		Agent:           state.Agent,
		InteractionMode: InteractionModePTY,
		RepositoryURL:   state.RepositoryURL,
		Branch:          state.Branch,
		SandboxPath:     state.SandboxPath,
		LaunchCommand:   state.Command,
		LaunchArgs:      state.Args,
		WorkDir:         state.WorkDir,
		TicketSlug:      state.TicketSlug,
		StartedAt:       state.StartedAt,
		Status:          PodStatusInitializing,
		vtProvider:      func() *vt.VirtualTerminal { return virtualTerm },
	}

	comps := &PTYComponents{Terminal: term, VirtualTerminal: virtualTerm, Aggregator: agg, PTYLogger: ptyLogger}

	// Wire up output handler (shared implementation with circuit breaker + inline recover)
	term.SetOutputHandler(NewPTYOutputHandler(podKey, comps, pod.NotifyStateDetectorWithScreen))

	// Create PodIO before Start so consumers can use it immediately
	ptyIO := NewPTYPodIO(podKey, comps, PTYPodIODeps{
		GetOrCreateDetector: pod.GetOrCreateStateDetector,
		SubscribeState:      pod.SubscribeStateChange,
		UnsubscribeState:    pod.UnsubscribeStateChange,
		GetPTYError:         pod.GetPTYError,
	})
	pod.IO = ptyIO

	// Wire PodRelay for mode-specific relay behavior
	pod.Relay = NewPTYPodRelay(podKey, pod.IO, comps)

	// Set exit and error handlers via PodIO
	pod.IO.SetExitHandler(r.messageHandler.createExitHandler(podKey))
	pod.IO.SetIOErrorHandler(r.messageHandler.createPTYErrorHandler(podKey, pod))

	// Start Terminal I/O (readOutput + waitExit goroutines)
	if err := pod.IO.Start(); err != nil {
		if pod.IO != nil {
			pod.IO.Teardown()
		}
		dpty.Close()
		return nil, fmt.Errorf("start terminal: %w", err)
	}

	pod.SetStatus(PodStatusRunning)

	// Register with MCP and monitor
	if mcpSrv := r.GetMCPServer(); mcpSrv != nil {
		mcpSrv.RegisterPod(podKey, r.conn.GetOrgSlug(), nil, nil, state.Agent)
	}
	if agentMon := r.GetAgentMonitor(); agentMon != nil {
		agentMon.RegisterPod(podKey, pod.IO.GetPID())
	}

	// Subscribe to VT state detection events, bridge to gRPC (shared with OnCreatePod)
	pod.SubscribeAgentStatusBridge(r.conn.SendAgentStatus)

	return pod, nil
}
