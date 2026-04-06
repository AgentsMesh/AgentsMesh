package autopilot

import "time"

// =============================================================================
// Test Helper Methods (unexported)
// These methods provide access to internal components for testing purposes.
// They delegate to the appropriate component and are not part of the public API.
// =============================================================================

func (ac *AutopilotController) extractSessionID(output string) {
	if sessionID := ExtractSessionID(output); sessionID != "" {
		ac.controlRunner.SetSessionID(sessionID)
	}
}

func (ac *AutopilotController) getSessionID() string {
	return ac.controlRunner.GetSessionID()
}

func (ac *AutopilotController) buildPrompt() string {
	return ac.promptBuilder.BuildPrompt()
}

func (ac *AutopilotController) buildResumePrompt(iteration int) string {
	return ac.promptBuilder.BuildResumePrompt(iteration)
}

func (ac *AutopilotController) parseDecision(output string) *ControlDecision {
	return NewDecisionParser().ParseDecision(output)
}

func (ac *AutopilotController) setPhaseForTest(phase Phase) {
	ac.phaseMgr.SetPhaseWithoutReport(phase)
}

func (ac *AutopilotController) setIterationForTest(iteration int) {
	ac.iterCtrl.mu.Lock()
	ac.iterCtrl.currentIter = iteration
	ac.iterCtrl.mu.Unlock()
}

// GetProgressSummary returns a human-readable summary of task progress.
func (ac *AutopilotController) GetProgressSummary() string {
	if ac.progressTracker == nil {
		return "No progress tracking available"
	}
	return ac.progressTracker.GenerateSummary()
}

// IsStuck checks if no progress has been made for the specified duration.
func (ac *AutopilotController) IsStuck(threshold time.Duration) bool {
	if ac.progressTracker == nil {
		return false
	}
	return ac.progressTracker.IsStuck(threshold)
}

// GetChangedFiles returns files changed during the autopilot session.
func (ac *AutopilotController) GetChangedFiles() []string {
	if ac.progressTracker == nil {
		return nil
	}
	startedAt := ac.iterCtrl.GetStartedAt()
	return ac.progressTracker.GetChangedFilesSince(startedAt)
}
