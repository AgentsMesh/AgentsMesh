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
			// Perpetual pod: daemon died → re-create it from the existing sandbox
			if state.Perpetual {
				pod, restartErr := r.restartDeadPerpetualDaemon(state)
				if restartErr != nil {
					log.Warn("failed to restart perpetual daemon, cleaning up",
						"pod_key", state.PodKey, "error", restartErr)
					_ = r.podDaemonManager.CleanupSession(state.SandboxPath)
					continue
				}
				log.Info("perpetual daemon restarted",
					"pod_key", pod.PodKey, "sandbox", pod.SandboxPath)
				continue
			}

			log.Warn("failed to recover session, cleaning up",
				"pod_key", state.PodKey, "error", err)
			_ = r.podDaemonManager.CleanupSession(state.SandboxPath)
			continue
		}

		current, stillOwned := r.podStore.Get(pod.PodKey)
		if !stillOwned || current != pod {
			log.Info("session exited while recovery was committing", "pod_key", pod.PodKey)
			continue
		}
		log.Info("session recovered",
			"pod_key", pod.PodKey,
			"pid", func() int {
				pid := 0
				pod.WithActiveIO(func(io PodIO) { pid = io.GetPID() })
				return pid
			}(),
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

	virtualTerm := vt.NewVirtualTerminal(state.Cols, state.Rows, state.VTHistoryLimit)
	virtualTerm.SetOSCHandler(r.messageHandler.createOSCHandler(state.PodKey))

	agg := aggregator.NewSmartAggregator(nil,
		aggregator.WithFullRedrawThrottling(),
	)

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
		LaunchEnv:       state.Env,
		Perpetual:       state.Perpetual,
		TicketSlug:      state.TicketSlug,
		StartedAt:       state.StartedAt,
		Status:          PodStatusInitializing,
	}

	runtime := assemblePTYRuntime(
		pod, term, virtualTerm, agg, ptyLogger, r.localServer,
	)
	pod.installRuntime(runtime)

	// Set exit and error handlers via PodIO
	runtime.IO.SetExitHandler(r.messageHandler.createExitHandler(podKey))
	runtime.IO.SetIOErrorHandler(r.messageHandler.createPTYErrorHandler(podKey, pod))
	r.podStore.Put(podKey, pod)
	pod.lifecycleMu.Lock()

	// Start Terminal I/O (readOutput + waitExit goroutines) and commit the
	// recovered owner. The helper owns rollback and releases lifecycleMu only
	// on failure; the success path keeps the lock through final registration.
	if err := startRecoveredRuntime(r.podStore, pod, runtime, func() { _ = dpty.Close() }); err != nil {
		return nil, err
	}

	pod.SetStatus(PodStatusRunning)

	// Register with MCP and monitor
	if mcpSrv := r.GetMCPServer(); mcpSrv != nil {
		mcpSrv.RegisterPod(podKey, r.conn.GetOrgSlug(), nil, nil, state.Agent)
	}
	if agentMon := r.GetAgentMonitor(); agentMon != nil {
		agentMon.RegisterPod(podKey, runtime.IO.GetPID())
	}

	// Subscribe to VT state detection events, bridge to gRPC (shared with OnCreatePod)
	pod.SubscribeAgentStatusBridge(r.conn.SendAgentStatus)
	pod.lifecycleMu.Unlock()

	return pod, nil
}

// startRecoveredRuntime starts a runtime while the caller holds pod.lifecycleMu.
// On failure it rolls back store ownership, tears the runtime down, closes the
// attached session, and releases lifecycleMu. On success ownership of the lock
// remains with the caller so status and sidecar registration commit atomically.
func startRecoveredRuntime(
	store PodStore,
	pod *Pod,
	runtime PodRuntime,
	closeAttached func(),
) error {
	if err := runtime.IO.Start(); err != nil {
		store.DeleteIf(pod.PodKey, pod)
		pod.lifecycleMu.Unlock()
		runtime.IO.Stop()
		runtime.IO.Teardown()
		closeAttached()
		return fmt.Errorf("start terminal: %w", err)
	}
	return nil
}

// restartDeadPerpetualDaemon re-creates a daemon session for a perpetual pod
// whose daemon died. Uses the existing sandbox and state to spawn a new daemon.
func (r *Runner) restartDeadPerpetualDaemon(state *poddaemon.PodDaemonState) (*Pod, error) {
	_, updatedState, err := r.podDaemonManager.CreateSession(poddaemon.CreateOpts{
		PodKey:         state.PodKey,
		Agent:          state.Agent,
		Command:        state.Command,
		Args:           state.Args,
		WorkDir:        state.WorkDir,
		Env:            state.Env,
		Cols:           state.Cols,
		Rows:           state.Rows,
		SandboxPath:    state.SandboxPath,
		RepositoryURL:  state.RepositoryURL,
		Branch:         state.Branch,
		TicketSlug:     state.TicketSlug,
		VTHistoryLimit: state.VTHistoryLimit,
		Perpetual:      true,
	})
	if err != nil {
		return nil, fmt.Errorf("create daemon session: %w", err)
	}

	// recoverSingleSession will AttachSession (new TCP conn). The CreateSession
	// connection is implicitly replaced by daemon's single-client model.
	return r.recoverSingleSession(updatedState)
}
