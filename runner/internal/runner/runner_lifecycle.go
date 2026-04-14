package runner

import (
	"context"
	"sync"
	"time"

	"github.com/thejerf/suture/v4"

	"github.com/anthropics/agentsmesh/runner/internal/autopilot"
	"github.com/anthropics/agentsmesh/runner/internal/lifecycle"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/terminal"
)

// AddService registers an additional suture.Service to be managed by the Supervisor.
// Must be called before Run().
func (r *Runner) AddService(svc suture.Service) {
	r.additionalServices = append(r.additionalServices, svc)
}

// Run starts the runner with a suture Supervisor tree and blocks until context is cancelled.
// All core components (gRPC connection, MCP server, Monitor, etc.) are managed by the Supervisor,
// which automatically restarts them on failure.
//
// Shutdown order: when ctx is cancelled, pods are stopped first (while gRPC is still alive
// to send final output/status), then the supervisor is torn down.
func (r *Runner) Run(ctx context.Context) error {
	log := logger.Runner()
	log.Info("Runner starting", "node_id", r.cfg.NodeID, "org", r.cfg.OrgSlug)

	// Store lifecycle context so message handlers can derive cancellable contexts
	// for long-running operations (e.g., git clone in OnCreatePod).
	r.runCtx = ctx

	// Clean up stale stack dump files from previous runs (>24h old)
	terminal.CleanupOldStackDumps(24 * time.Hour)

	// Recover daemon sessions from previous Runner lifecycle
	r.recoverDaemonSessions()

	// Create top-level Supervisor
	supervisor := suture.New("runner", suture.Spec{
		EventHook: func(e suture.Event) {
			log.Warn("Supervisor event", "event", e.String())
		},
		FailureThreshold: 5,
		FailureDecay:     60,
		FailureBackoff:   5 * time.Second,
	})

	// Register core services
	supervisor.Add(&lifecycle.ConnectionService{Conn: r.conn})

	for _, svc := range r.sidecars.Services() {
		supervisor.Add(svc)
	}

	// Register Watchdog health monitor
	watchdogCfg := lifecycle.WatchdogConfig{
		Interval: 15 * time.Second,
	}
	// Wire up connection activity monitoring if GRPCConnection supports it
	if am, ok := r.conn.(lifecycle.ActivityMonitor); ok {
		watchdogCfg.ConnMonitor = am
	}
	supervisor.Add(lifecycle.NewWatchdogService(watchdogCfg))

	// Register Pod Reconciler: periodically detects dead pod processes
	// whose exit handlers never fired, preventing zombie relay connections.
	supervisor.Add(NewPodReconciler(
		r.podStore,
		r.messageHandler.cleanupPodExit,
		60*time.Second,
	))

	// Register additional services (Console, etc.)
	for _, svc := range r.additionalServices {
		supervisor.Add(svc)
	}

	// Decouple supervisor lifecycle from external shutdown signal.
	// When ctx is cancelled, stop pods first (while gRPC is still alive),
	// then cancel the supervisor.
	supervisorCtx, supervisorCancel := context.WithCancel(context.Background())
	defer supervisorCancel()

	shutdownDone := make(chan struct{})
	go func() {
		<-ctx.Done()
		log.Info("Shutting down runner...")
		r.stopAllPods() // Pods stop while gRPC still connected
		close(shutdownDone)
		supervisorCancel() // Now tear down gRPC + other services
	}()

	// Supervisor.Serve() blocks until supervisorCtx is cancelled
	err := supervisor.Serve(supervisorCtx)

	// If supervisor exited on its own (not from our shutdown goroutine),
	// also clean up pods
	select {
	case <-shutdownDone:
		// Normal shutdown — pods already stopped
	default:
		r.stopAllPods()
	}

	return err
}

// stopAllPods stops all active autopilots and pods during shutdown.
// Pods are stopped in parallel with a global timeout to fit within
// Windows SCM's 20s shutdown limit.
func (r *Runner) stopAllPods() {
	log := logger.Runner()

	// Stop all autopilot controllers first (they depend on pods).
	// DrainAll atomically removes all controllers and returns them
	// so we can stop them in parallel without holding the lock.
	var autopilotsCopy []*autopilot.AutopilotController
	if r.autopilotStore != nil {
		autopilotsCopy = r.autopilotStore.DrainAll()
	}

	if len(autopilotsCopy) > 0 {
		log.Info("Stopping all autopilots in parallel", "count", len(autopilotsCopy))
		var apWg sync.WaitGroup
		for _, ac := range autopilotsCopy {
			apWg.Add(1)
			go func(a *autopilot.AutopilotController) {
				defer apWg.Done()
				a.Stop()
			}(ac)
		}
		apWg.Wait()
	}

	// Stop all pods in parallel
	pods := r.podStore.All()
	if len(pods) == 0 {
		return
	}
	log.Info("Stopping all pods in parallel", "count", len(pods))

	var wg sync.WaitGroup
	for _, pod := range pods {
		wg.Add(1)
		go func(p *Pod) {
			defer wg.Done()
			log.Debug("Detaching pod (daemon stays alive)", "pod_key", p.PodKey)

			// Capture metadata before cleanup for token usage collection.
			podKey := p.PodKey
			agent := p.Agent
			sandboxPath := p.SandboxPath
			podStartedAt := p.StartedAt

			p.StopStateDetector()
			// Mode-specific infrastructure cleanup (aggregator, loggers).
			// Must happen BEFORE DisconnectRelay so the final flush reaches the browser.
			if p.IO != nil {
				p.IO.Teardown()
			}
			p.DisconnectRelay()
			if p.IO != nil {
				// Detach instead of Stop: daemon + child process stay alive
				// so the session can be recovered after Runner restart.
				p.IO.Detach()
			}
			r.podStore.Delete(p.PodKey)

			// Collect token usage synchronously (best-effort).
			// Token files live in HOME (~/.claude/, ~/.codex/), not in sandbox,
			// so they survive Terminal.Detach(). gRPC is still alive at this point.
			if r.messageHandler != nil {
				r.messageHandler.collectAndSendTokenUsage(podKey, agent, sandboxPath, podStartedAt)
			}
		}(pod)
	}

	// Total timeout: fits within Windows SCM 20s limit
	// (leaves headroom for autopilot stop + supervisor teardown).
	// Use a timer so we can stop it on success, preventing the
	// wg.Wait goroutine from leaking on timeout.
	timer := time.NewTimer(15 * time.Second)
	defer timer.Stop()

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()

	select {
	case <-done:
		log.Info("All pods stopped successfully")
	case <-timer.C:
		log.Warn("Timeout waiting for pods to stop — some token usage data may be lost", "count", len(pods))
	}
}
