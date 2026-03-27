package tokenusage

import (
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// Collect gathers token usage for a pod session.
// agent is the LaunchCommand (e.g., "claude", "aider").
// sandboxPath is the pod's sandbox root directory.
// podStartedAt is the pod's start time; only files modified after this time are processed.
// Returns nil if no parser is available or no usage data is found.
func Collect(agent, sandboxPath string, podStartedAt time.Time) *TokenUsage {
	log := logger.Pod()

	parser := GetParser(agent)
	if parser == nil {
		log.Debug("No token usage parser for agent", "agent", agent)
		return nil
	}

	usage, err := parser.Parse(sandboxPath, podStartedAt)
	if err != nil {
		log.Warn("Token usage collection failed",
			"agent", agent,
			"sandbox_path", sandboxPath,
			"error", err,
		)
		return nil
	}

	if usage == nil || usage.IsEmpty() {
		log.Debug("No token usage data found", "agent", agent)
		return nil
	}

	// Log summary
	for _, m := range usage.Sorted() {
		log.Info("Token usage collected",
			"agent", agent,
			"model", m.Model,
			"input", m.InputTokens,
			"output", m.OutputTokens,
			"cache_creation", m.CacheCreationTokens,
			"cache_read", m.CacheReadTokens,
		)
	}

	return usage
}
