package autopilot

import (
	"os"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// Start initializes and starts the AutopilotController.
// It checks if the Pod is waiting and sends the initial prompt if so.
func (ac *AutopilotController) Start() error {
	ac.log.Info("Starting AutopilotController", "autopilot_key", ac.key, "pod_key", ac.podKey)

	// Start state detection (deferred from constructor to avoid goroutines in New*)
	ac.stateCoordinator.Start()

	// Report created event
	if ac.reporter != nil {
		ac.reporter.ReportAutopilotCreated(&runnerv1.AutopilotCreatedEvent{
			AutopilotKey: ac.key,
			PodKey:       ac.podKey,
		})
	}

	// Check Pod current status
	agentStatus := ac.podCtrl.GetAgentStatus()
	ac.log.Info("Pod current status", "status", agentStatus)

	if agentStatus == "waiting" {
		// Pod is waiting for input, send initial_prompt
		ac.sendInitialPrompt()
	}
	// If executing, we'll wait for the next waiting event

	ac.phaseMgr.SetPhase(PhaseRunning)
	return nil
}

// Stop stops the AutopilotController and waits for all goroutines to complete.
// This method is safe to call multiple times - cleanup only runs once,
// but it always waits for goroutines to finish.
func (ac *AutopilotController) Stop() {
	// Run cleanup logic only once
	ac.stopOnce.Do(func() {
		// Set phase to stopped (may already be stopped by Approve(false, _))
		ac.phaseMgr.SetPhaseWithoutReport(PhaseStopped)

		ac.stateCoordinator.Stop()

		// Stop control process (ACP mode: shuts down long-lived session)
		if ac.controlRunner != nil {
			ac.controlRunner.Stop()
		}

		// Acquire wgMu to ensure no new wg.Add() can happen while we set stopped=true.
		// This guarantees that after this block, no new goroutines will be added to wg.
		ac.wgMu.Lock()
		ac.stopped = true
		ac.wgMu.Unlock()

		ac.cancel()

		// Wait for all running goroutines to complete with a timeout.
		// In rare edge cases (e.g., process stuck in D-state), wg.Wait() could
		// block indefinitely. A 30-second timeout prevents the entire Runner
		// cleanup from hanging.
		done := make(chan struct{})
		go func() {
			ac.wg.Wait()
			close(done)
		}()
		select {
		case <-done:
			// All goroutines finished cleanly
		case <-time.After(30 * time.Second):
			ac.log.Error("Timed out waiting for goroutines to finish during Stop()",
				"autopilot_key", ac.key,
				"timeout", "30s")
		}

		// Cleanup MCP config file
		if ac.mcpConfigPath != "" {
			if err := os.Remove(ac.mcpConfigPath); err != nil && !os.IsNotExist(err) {
				ac.log.Warn("Failed to cleanup MCP config file",
					"path", ac.mcpConfigPath,
					"error", err)
			}
		}

		ac.log.Info("AutopilotController stopped", "autopilot_key", ac.key)

		if ac.reporter != nil {
			ac.reporter.ReportAutopilotTerminated(&runnerv1.AutopilotTerminatedEvent{
				AutopilotKey: ac.key,
				Reason:       "stopped",
			})
		}
	})
}

// Pause pauses the AutopilotController.
func (ac *AutopilotController) Pause() {
	if ac.phaseMgr.GetPhase() == PhaseRunning {
		ac.phaseMgr.SetPhase(PhasePaused)
		ac.log.Info("AutopilotController paused", "autopilot_key", ac.key)
	}
}

// Resume resumes a paused AutopilotController.
func (ac *AutopilotController) Resume() {
	if ac.phaseMgr.GetPhase() == PhasePaused {
		ac.phaseMgr.SetPhase(PhaseRunning)
		ac.log.Info("AutopilotController resumed", "autopilot_key", ac.key)
	}
}

// Takeover allows the user to take control.
func (ac *AutopilotController) Takeover() {
	ac.userHandler.Takeover()
	ac.log.Info("User takeover", "autopilot_key", ac.key)
}

// Handback returns control to AutopilotController.
func (ac *AutopilotController) Handback() {
	ac.userHandler.Handback()
	ac.log.Info("User handback", "autopilot_key", ac.key)
}

// Approve handles approval when Control requests human help (NEED_HUMAN_HELP).
func (ac *AutopilotController) Approve(continueExecution bool, additionalIterations int32) {
	ac.userHandler.Approve(continueExecution, additionalIterations)
}

// onResumeFromUserInteraction is called after user handback or approve.
// It checks if the Pod is waiting and triggers an iteration if needed.
func (ac *AutopilotController) onResumeFromUserInteraction() {
	// Small delay to allow state to stabilize, with context cancellation check
	select {
	case <-time.After(DefaultResumeDelay):
		// Continue after delay
	case <-ac.ctx.Done():
		// Controller stopped during delay, abort
		return
	}

	// Check if Pod is waiting for input
	if ac.podCtrl != nil && ac.podCtrl.GetAgentStatus() == "waiting" {
		ac.log.Info("Pod is waiting after resume, triggering iteration",
			"autopilot_key", ac.key)
		ac.OnPodWaiting()
	}
}
