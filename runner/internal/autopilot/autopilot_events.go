package autopilot

import (
	runnerv1 "github.com/AgentsMesh/AgentsMesh/proto/gen/go/runner/v1"
)

// reportThinkingEvent sends an AutopilotThinkingEvent to expose the Control Agent's decision process.
func (ac *AutopilotController) reportThinkingEvent(decision *ControlDecision, iteration int) {
	if ac.reporter == nil {
		return
	}

	event := &runnerv1.AutopilotThinkingEvent{
		AutopilotKey: ac.key,
		Iteration:    int32(iteration),
		DecisionType: string(decision.Type),
		Reasoning:    decision.Reasoning,
		Confidence:   decision.Confidence,
	}

	// Add action if present
	if decision.Action != nil {
		event.Action = &runnerv1.AutopilotAction{
			Type:    decision.Action.Type,
			Content: decision.Action.Content,
			Reason:  decision.Action.Reason,
		}
	}

	// Add progress if present
	if decision.Progress != nil {
		event.Progress = &runnerv1.AutopilotProgress{
			Summary:        decision.Progress.Summary,
			CompletedSteps: decision.Progress.CompletedSteps,
			RemainingSteps: decision.Progress.RemainingSteps,
			Percent:        int32(decision.Progress.Percent),
		}
	}

	// Add help request if present
	if decision.HelpRequest != nil {
		event.HelpRequest = &runnerv1.AutopilotHelpRequest{
			Reason:          decision.HelpRequest.Reason,
			Context:         decision.HelpRequest.Context,
			TerminalExcerpt: decision.HelpRequest.TerminalExcerpt,
		}
		for _, s := range decision.HelpRequest.Suggestions {
			event.HelpRequest.Suggestions = append(event.HelpRequest.Suggestions, &runnerv1.AutopilotHelpSuggestion{
				Action: s.Action,
				Label:  s.Label,
			})
		}
	}

	ac.reporter.ReportAutopilotThinking(event)
}

// buildAutopilotStatus builds an AutopilotStatus proto from current state.
// Used as a callback by PhaseManager for status reporting.
func (ac *AutopilotController) buildAutopilotStatus() *runnerv1.AutopilotStatus {
	status := ac.iterCtrl.GetStatus()
	status.Phase = string(ac.phaseMgr.GetPhase())
	status.PodStatus = ac.podCtrl.GetAgentStatus()
	return status
}
