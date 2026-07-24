package runner

import (
	"context"
	"time"

	"github.com/thejerf/suture/v4"

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

	// Bring up the local browser-facing relay server. Failure here is non-fatal:
	// runner falls back to cloud-only mode if 127.0.0.1 binding is blocked.
	if r.localServer != nil {
		if _, err := r.localServer.Start(ctx); err != nil {
			log.Warn("Local relay server failed to start, continuing without it", "error", err)
			r.localServer = nil
		}
	}

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
