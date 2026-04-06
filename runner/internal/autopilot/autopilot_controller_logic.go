package autopilot

import (
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// OnPodWaiting is called when the Pod transitions to waiting state.
// This is the main event-driven entry point triggered by StateDetectorCoordinator.
// Includes deduplication to prevent rapid re-triggering.
func (ac *AutopilotController) OnPodWaiting() {
	ac.log.Debug("OnPodWaiting triggered", "autopilot_key", ac.key)

	// Check trigger deduplication
	if !ac.iterCtrl.CheckTriggerDedup() {
		ac.log.Debug("Skipping iteration - deduplication", "autopilot_key", ac.key)
		return
	}

	// Check if user has taken over
	if ac.userHandler.IsUserTakeover() {
		ac.log.Debug("Skipping iteration - user takeover", "autopilot_key", ac.key)
		return
	}

	// Check if phase allows iteration
	if !ac.phaseMgr.CanProcessIteration() {
		ac.log.Debug("Skipping iteration - phase not ready", "autopilot_key", ac.key, "phase", ac.phaseMgr.GetPhase())
		return
	}

	// Check max iterations (the only hard protection)
	if ac.iterCtrl.HasReachedMaxIterations() {
		ac.phaseMgr.SetPhase(PhaseMaxIterations)
		ac.log.Info("Max iterations reached", "autopilot_key", ac.key)
		if ac.reporter != nil {
			ac.reporter.ReportAutopilotTerminated(&runnerv1.AutopilotTerminatedEvent{
				AutopilotKey: ac.key,
				Reason:       "max_iterations",
			})
		}
		return
	}

	// Increment iteration
	iteration, ok := ac.iterCtrl.IncrementIteration()
	if !ok {
		// Max iterations reached during increment
		ac.phaseMgr.SetPhase(PhaseMaxIterations)
		ac.log.Info("Max iterations reached", "autopilot_key", ac.key)
		if ac.reporter != nil {
			ac.reporter.ReportAutopilotTerminated(&runnerv1.AutopilotTerminatedEvent{
				AutopilotKey: ac.key,
				Reason:       "max_iterations",
			})
		}
		return
	}

	// Report iteration started
	ac.iterCtrl.ReportIterationEvent(iteration, "started", "", nil)

	// Run single decision in a goroutine.
	// Acquire wgMu to ensure atomicity of the stopped check + wg.Add(1).
	// Without this lock, Stop() could set stopped=true and call wg.Wait()
	// between our ctx check and wg.Add(1), causing a panic.
	ac.wgMu.Lock()
	if ac.stopped {
		ac.wgMu.Unlock()
		ac.log.Debug("Controller stopped before starting decision goroutine", "autopilot_key", ac.key)
		return
	}
	ac.wg.Add(1)
	ac.wgMu.Unlock()

	go func() {
		defer ac.wg.Done()
		ac.runSingleDecision(iteration)
	}()
}

// sendPrompt starts the first iteration when Pod is waiting.
// This launches the control process which will use MCP tools to interact with Pod.
func (ac *AutopilotController) sendPrompt() {
	ac.log.Info("Starting initial iteration", "autopilot_key", ac.key)

	// Update trigger time to prevent OnPodWaiting from double-triggering
	ac.iterCtrl.UpdateTriggerTime()

	// Set initial iteration
	iteration := ac.iterCtrl.SetInitialIteration()

	// Report iteration started
	ac.iterCtrl.ReportIterationEvent(iteration, "started", "", nil)

	// Run the control process.
	// Acquire wgMu to ensure atomicity of the stopped check + wg.Add(1).
	ac.wgMu.Lock()
	if ac.stopped {
		ac.wgMu.Unlock()
		ac.log.Debug("Controller stopped before starting initial decision goroutine", "autopilot_key", ac.key)
		return
	}
	ac.wg.Add(1)
	ac.wgMu.Unlock()

	go func() {
		defer ac.wg.Done()
		ac.runSingleDecision(iteration)
	}()
}
