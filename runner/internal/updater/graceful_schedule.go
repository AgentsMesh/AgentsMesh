package updater

import (
	"context"
	"fmt"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// ScheduleUpdate checks for updates and schedules a graceful update if available.
// It waits for all pods to finish before applying. If maxWaitTime is reached,
// the update is postponed to the next check cycle.
func (g *GracefulUpdater) ScheduleUpdate(ctx context.Context) error {
	log := logger.Updater()

	// Atomically check and set state to avoid race condition
	g.mu.Lock()
	if g.state != StateIdle {
		currentState := g.state
		g.mu.Unlock()
		log.Warn("Update already in progress", "state", currentState)
		return fmt.Errorf("update already in progress (state: %s)", currentState)
	}
	g.state = StateChecking
	g.mu.Unlock()

	log.Info("Checking for updates")

	// Notify status change
	if g.onStatus != nil {
		g.onStatus(StateChecking, nil, 0)
	}
	info, err := g.updater.CheckForUpdate(ctx)
	if err != nil {
		g.setState(StateIdle)
		log.Error("Update check failed", "error", err)
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	if !info.HasUpdate {
		g.setState(StateIdle)
		return nil
	}

	log.Info("New version available", "current", info.CurrentVersion, "latest", info.LatestVersion)

	g.mu.Lock()
	g.pendingInfo = info
	g.mu.Unlock()

	// Wait for pods to drain before applying
	if err := g.drainPods(ctx); err != nil {
		g.mu.Lock()
		g.pendingInfo = nil
		g.mu.Unlock()
		return err
	}

	return g.executeUpdate(ctx)
}

// drainPods waits for all active pods to finish before proceeding.
func (g *GracefulUpdater) drainPods(ctx context.Context) error {
	log := logger.Updater()
	g.setState(StateDraining)
	g.mu.Lock()
	g.draining = true

	drainCtx, cancel := context.WithTimeout(ctx, g.maxWaitTime)
	g.cancelDrain = cancel
	g.mu.Unlock()

	defer func() {
		cancel()
		g.mu.Lock()
		g.draining = false
		g.cancelDrain = nil
		g.mu.Unlock()
	}()

	ticker := time.NewTicker(g.pollInterval)
	defer ticker.Stop()

	for {
		activePods := 0
		if g.podCounter != nil {
			activePods = g.podCounter()
		}

		if activePods == 0 {
			log.Info("No active pods, applying update")
			return nil
		}

		log.Info("Waiting for active pods to finish", "count", activePods)

		if g.onStatus != nil {
			g.mu.RLock()
			info := g.pendingInfo
			g.mu.RUnlock()
			g.onStatus(StateDraining, info, activePods)
		}

		select {
		case <-drainCtx.Done():
			if drainCtx.Err() == context.DeadlineExceeded {
				log.Warn("Max wait time reached, postponing update", "active_pods", activePods)
				g.mu.Lock()
				g.pendingInfo = nil
				g.mu.Unlock()
				g.setState(StateIdle)
				return fmt.Errorf("update postponed: max wait time reached with active pods")
			}
			g.mu.Lock()
			g.pendingInfo = nil
			g.mu.Unlock()
			g.setState(StateIdle)
			return fmt.Errorf("update cancelled")
		case <-ticker.C:
			// Continue polling
		}
	}
}

// ForceUpdate applies the update immediately without waiting for pods.
func (g *GracefulUpdater) ForceUpdate(ctx context.Context) error {
	log := logger.Updater()
	log.Info("Force update requested")

	g.mu.Lock()
	if g.state != StateIdle && g.state != StateDraining {
		currentState := g.state
		g.mu.Unlock()
		return fmt.Errorf("cannot force update in state: %s", currentState)
	}

	// Cancel ongoing drain if any
	if g.cancelDrain != nil {
		g.cancelDrain()
	}

	// Check if we already have pending info (from a ScheduleUpdate that was draining)
	hasPending := g.pendingInfo != nil
	if !hasPending {
		g.state = StateChecking
	}
	g.mu.Unlock()

	if hasPending {
		return g.executeUpdate(ctx)
	}

	// Notify status change (state already set above)
	if g.onStatus != nil {
		g.onStatus(StateChecking, nil, 0)
	}
	info, err := g.updater.CheckForUpdate(ctx)
	if err != nil {
		g.setState(StateIdle)
		return err
	}

	if !info.HasUpdate {
		g.setState(StateIdle)
		return fmt.Errorf("no update available")
	}

	g.mu.Lock()
	g.pendingInfo = info
	g.mu.Unlock()

	return g.executeUpdate(ctx)
}
