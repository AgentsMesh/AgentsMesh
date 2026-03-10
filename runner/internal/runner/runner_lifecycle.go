package runner

import (
	"context"
	"sync"
	"time"

	"github.com/thejerf/suture/v4"

	"github.com/anthropics/agentsmesh/runner/internal/autopilot"
	"github.com/anthropics/agentsmesh/runner/internal/lifecycle"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
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

	if r.mcpServer != nil {
		supervisor.Add(&lifecycle.MCPServerService{Server: r.mcpServer})
	}
	if r.agentMonitor != nil {
		supervisor.Add(&lifecycle.MonitorService{Monitor: r.agentMonitor})
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
		r.stopAllPods()    // Pods stop while gRPC still connected
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

	// Stop all autopilot controllers first (they depend on pods)
	r.autopilotsMu.Lock()
	if len(r.autopilots) > 0 {
		log.Info("Stopping all autopilots", "count", len(r.autopilots))
	}
	for key, ac := range r.autopilots {
		log.Debug("Stopping autopilot", "autopilot_key", key)
		ac.Stop()
	}
	r.autopilots = make(map[string]*autopilot.AutopilotController)
	r.autopilotsMu.Unlock()

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
			log.Debug("Stopping pod", "pod_key", p.PodKey)
			p.DisconnectRelay()
			p.StopStateDetector()
			if p.Aggregator != nil {
				p.Aggregator.Stop()
			}
			if p.Terminal != nil {
				p.Terminal.Stop()
			}
			r.podStore.Delete(p.PodKey)
		}(pod)
	}

	// Total timeout: fits within Windows SCM 20s limit
	// (leaves headroom for autopilot stop + supervisor teardown)
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()

	select {
	case <-done:
		log.Info("All pods stopped successfully")
	case <-time.After(15 * time.Second):
		log.Warn("Timeout waiting for pods to stop", "count", len(pods))
	}
}

// IsDraining returns true if the runner is waiting for pods to finish before update.
func (r *Runner) IsDraining() bool {
	r.drainingMu.RLock()
	defer r.drainingMu.RUnlock()
	return r.draining
}

// SetDraining sets the draining state.
func (r *Runner) SetDraining(draining bool) {
	r.drainingMu.Lock()
	defer r.drainingMu.Unlock()
	r.draining = draining
	if draining {
		logger.Runner().Info("Entering draining mode - no new pods will be accepted")
	} else {
		logger.Runner().Info("Exiting draining mode - accepting pods again")
	}
}

// CanAcceptPod returns true if the runner can accept new pods.
func (r *Runner) CanAcceptPod() bool {
	r.drainingMu.RLock()
	draining := r.draining
	r.drainingMu.RUnlock()

	if draining {
		logger.Runner().Debug("Cannot accept pod: runner is draining")
		return false
	}

	currentCount := r.GetActivePodCount()
	if currentCount >= r.cfg.MaxConcurrentPods {
		logger.Runner().Debug("Cannot accept pod: max capacity reached",
			"current", currentCount, "max", r.cfg.MaxConcurrentPods)
		return false
	}

	return true
}

// GetActivePodCount returns the number of currently active pods.
func (r *Runner) GetActivePodCount() int {
	return r.podStore.Count()
}

// GetPodCounter returns a function that counts active pods.
func (r *Runner) GetPodCounter() func() int {
	return func() int {
		return r.GetActivePodCount()
	}
}
