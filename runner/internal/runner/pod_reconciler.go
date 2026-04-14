package runner

import (
	"context"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/process"
)

// PodReconciler periodically verifies that each pod's underlying process is
// still alive. If a pod's process has exited but the exit handler never fired
// (race condition, unexpected crash), the reconciler cleans it up. This
// prevents zombie relay connections from accumulating in long-running Runners.
type PodReconciler struct {
	podStore PodStore
	cleanup  func(podKey string, exitCode int, stopIO bool)
	interval time.Duration
}

// NewPodReconciler creates a PodReconciler.
// cleanup should be RunnerMessageHandler.cleanupPodExit.
func NewPodReconciler(
	store PodStore,
	cleanup func(podKey string, exitCode int, stopIO bool),
	interval time.Duration,
) *PodReconciler {
	if interval == 0 {
		interval = 60 * time.Second
	}
	return &PodReconciler{
		podStore: store,
		cleanup:  cleanup,
		interval: interval,
	}
}

// Serve implements suture.Service.
func (r *PodReconciler) Serve(ctx context.Context) error {
	log := logger.Runner()
	log.Info("PodReconciler starting", "interval", r.interval)

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			r.reconcile()
		}
	}
}

func (r *PodReconciler) String() string { return "PodReconciler" }

func (r *PodReconciler) reconcile() {
	log := logger.Runner()

	for _, pod := range r.podStore.All() {
		// Skip pods in intermediate states: initializing pods haven't started
		// their process yet, and stopped pods are already being cleaned up.
		status := pod.GetStatus()
		if status != PodStatusRunning {
			continue
		}

		if pod.IO == nil {
			continue
		}

		pid := pod.IO.GetPID()
		if pid <= 0 {
			continue // ACP mode or not yet started
		}

		if err := process.IsAlive(pid); err != nil {
			log.Warn("Reconciler detected dead pod process, cleaning up",
				"pod_key", pod.PodKey, "pid", pid, "reason", err)
			r.cleanup(pod.PodKey, -1, true)
		}
	}
}
