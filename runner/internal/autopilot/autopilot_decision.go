package autopilot

import (
	"context"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// runSingleDecision executes a single decision cycle using the control process.
// On error, it retries internally without consuming additional iteration quota.
func (ac *AutopilotController) runSingleDecision(iteration int) {
	ac.log.Info("Running single decision", "autopilot_key", ac.key, "iteration", iteration)

	// Internal retry loop - errors don't consume iteration quota
	for {
		startTime := time.Now()

		// Create timeout context
		timeout := time.Duration(ac.config.IterationTimeoutSeconds) * time.Second
		if timeout == 0 {
			timeout = DefaultIterationTimeout
		}
		ctx, cancel := context.WithTimeout(ac.ctx, timeout)

		// Run control process
		decision, err := ac.controlRunner.RunControlProcess(ctx, iteration)
		duration := time.Since(startTime)
		cancel()

		if err != nil {
			if ac.handleDecisionError(err, iteration) {
				return // Stop retrying
			}
			continue // Retry the loop
		}

		// Success - reset consecutive errors
		ac.iterCtrl.ResetErrors()

		// Process successful decision
		ac.processSuccessfulDecision(decision, iteration, duration)
		return
	}
}

// handleDecisionError handles errors from control process execution.
// Returns true if we should stop retrying.
func (ac *AutopilotController) handleDecisionError(err error, iteration int) bool {
	// Control execution failure - log and report
	ac.log.Error("Control process failed", "error", err, "iteration", iteration)
	ac.iterCtrl.ReportIterationEvent(iteration, "error", err.Error(), nil)

	// Record error and check if max consecutive errors exceeded
	if ac.iterCtrl.RecordError() {
		ac.log.Error("Max consecutive errors exceeded, stopping autopilot",
			"autopilot_key", ac.key,
			"consecutive_errors", ac.iterCtrl.GetConsecutiveErrors())
		ac.phaseMgr.SetPhase(PhaseFailed)
		if ac.reporter != nil {
			ac.reporter.ReportAutopilotTerminated(&runnerv1.AutopilotTerminatedEvent{
				AutopilotKey: ac.key,
				Reason:       "max_consecutive_errors",
			})
		}
		return true
	}

	// Check if Pod is still waiting and we should retry
	if ac.podCtrl == nil || ac.podCtrl.GetAgentStatus() != "waiting" {
		ac.log.Info("Pod not waiting, skipping retry", "autopilot_key", ac.key)
		return true
	}

	// Retry with exponential backoff (doesn't consume iteration quota)
	retryDelay := ac.iterCtrl.GetRetryDelay()
	ac.log.Info("Retrying control process with backoff (same iteration)",
		"autopilot_key", ac.key,
		"iteration", iteration,
		"retry_delay", retryDelay,
		"consecutive_errors", ac.iterCtrl.GetConsecutiveErrors())

	// Wait with context cancellation check
	select {
	case <-time.After(retryDelay):
		return false // Continue retrying
	case <-ac.ctx.Done():
		return true // Controller stopped
	}
}

// processSuccessfulDecision handles the aftermath of a successful control process run.
func (ac *AutopilotController) processSuccessfulDecision(decision *ControlDecision, iteration int, duration time.Duration) {
	// Capture progress snapshot AFTER iteration to detect actual changes
	if ac.progressTracker != nil {
		snapshot := ac.progressTracker.CaptureSnapshot()
		logger.AutopilotTrace().Trace("Progress snapshot captured after iteration",
			"iteration", iteration,
			"files_changed", len(snapshot.FilesModified),
			"has_changes", snapshot.GitDiff.HasChanges)

		// Use detected file changes if decision doesn't provide them
		if len(decision.FilesChanged) == 0 && len(snapshot.FilesModified) > 0 {
			decision.FilesChanged = snapshot.FilesModified
		}
	}

	// Update last decision
	ac.decisionMu.Lock()
	ac.lastDecision = string(decision.Type)
	ac.lastDecisionMsg = decision.Summary
	ac.decisionMu.Unlock()

	// Handle Control decision
	ac.handleDecision(decision, iteration, duration)
}

// handleDecision processes a Control decision and updates state accordingly.
func (ac *AutopilotController) handleDecision(decision *ControlDecision, iteration int, duration time.Duration) {
	// Report thinking event to expose Control Agent's decision process
	ac.reportThinkingEvent(decision, iteration)

	switch decision.Type {
	case DecisionCompleted:
		ac.phaseMgr.SetPhase(PhaseCompleted)
		ac.log.Info("Task completed", "autopilot_key", ac.key)
		ac.iterCtrl.ReportIterationEvent(iteration, "completed", decision.Summary, decision.FilesChanged)
		if ac.reporter != nil {
			ac.reporter.ReportAutopilotTerminated(&runnerv1.AutopilotTerminatedEvent{
				AutopilotKey: ac.key,
				Reason:       "completed",
			})
		}

	case DecisionNeedHumanHelp:
		ac.phaseMgr.SetPhase(PhaseWaitingApproval)
		ac.log.Warn("Control requests human help",
			"autopilot_key", ac.key,
			"reason", decision.Summary)
		ac.iterCtrl.ReportIterationEvent(iteration, "need_human_help", decision.Summary, nil)

	case DecisionGiveUp:
		ac.phaseMgr.SetPhase(PhaseFailed)
		ac.log.Warn("Control gave up",
			"autopilot_key", ac.key,
			"reason", decision.Summary)
		ac.iterCtrl.ReportIterationEvent(iteration, "give_up", decision.Summary, nil)
		if ac.reporter != nil {
			ac.reporter.ReportAutopilotTerminated(&runnerv1.AutopilotTerminatedEvent{
				AutopilotKey: ac.key,
				Reason:       "failed",
			})
		}

	case DecisionContinue:
		// Normal case - waiting for next Pod waiting state
		ac.iterCtrl.ReportIterationEvent(iteration, "action_sent", decision.Summary, decision.FilesChanged)
		ac.log.Info("Decision completed",
			"autopilot_key", ac.key,
			"iteration", iteration,
			"duration_ms", duration.Milliseconds(),
			"files_changed", len(decision.FilesChanged))
		// StateDetector will trigger next iteration when Pod is ready
	}
}
