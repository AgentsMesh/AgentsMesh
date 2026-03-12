package tokenusage

import (
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// Collect gathers token usage for a pod session.
// agentType is the LaunchCommand (e.g., "claude", "aider").
// sandboxPath is the pod's sandbox root directory.
// podStartedAt is the pod's start time; only files modified after this time are processed.
// Returns nil if no parser is available or no usage data is found.
func Collect(agentType, sandboxPath string, podStartedAt time.Time) *TokenUsage {
	log := logger.Pod()

	parser := GetParser(agentType)
	if parser == nil {
		log.Debug("No token usage parser for agent type", "agent_type", agentType)
		return nil
	}

	usage, err := parser.Parse(sandboxPath, podStartedAt)
	if err != nil {
		log.Warn("Token usage collection failed",
			"agent_type", agentType,
			"sandbox_path", sandboxPath,
			"error", err,
		)
		return nil
	}

	if usage == nil || usage.IsEmpty() {
		log.Debug("No token usage data found", "agent_type", agentType)
		return nil
	}

	// Log summary
	for _, m := range usage.Sorted() {
		log.Info("Token usage collected",
			"agent_type", agentType,
			"model", m.Model,
			"input", m.InputTokens,
			"output", m.OutputTokens,
			"cache_creation", m.CacheCreationTokens,
			"cache_read", m.CacheReadTokens,
		)
	}

	return usage
}
