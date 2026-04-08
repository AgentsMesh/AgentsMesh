package main

import (
	"github.com/anthropics/agentsmesh/backend/internal/infra/eventbus"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

func protoToEventbusThinking(data *runnerv1.AutopilotThinkingEvent) *eventbus.AutopilotThinkingData {
	result := &eventbus.AutopilotThinkingData{
		AutopilotControllerKey: data.GetAutopilotKey(),
		Iteration:              data.GetIteration(),
		DecisionType:           data.GetDecisionType(),
		Reasoning:              data.GetReasoning(),
		Confidence:             data.GetConfidence(),
	}

	if action := data.GetAction(); action != nil {
		result.Action = &eventbus.AutopilotActionData{
			Type:    action.GetType(),
			Content: action.GetContent(),
			Reason:  action.GetReason(),
		}
	}

	if progress := data.GetProgress(); progress != nil {
		result.Progress = &eventbus.AutopilotProgressData{
			Summary:        progress.GetSummary(),
			CompletedSteps: progress.GetCompletedSteps(),
			RemainingSteps: progress.GetRemainingSteps(),
			Percent:        progress.GetPercent(),
		}
	}

	if helpReq := data.GetHelpRequest(); helpReq != nil {
		result.HelpRequest = &eventbus.AutopilotHelpRequestData{
			Reason:          helpReq.GetReason(),
			Context:         helpReq.GetContext(),
			TerminalExcerpt: helpReq.GetTerminalExcerpt(),
		}
		for _, s := range helpReq.GetSuggestions() {
			result.HelpRequest.Suggestions = append(result.HelpRequest.Suggestions, eventbus.AutopilotHelpSuggestionData{
				Action: s.GetAction(),
				Label:  s.GetLabel(),
			})
		}
	}

	return result
}
