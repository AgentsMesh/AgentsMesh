package tokenusage

import (
	"log/slog"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// shortPodGracePeriod suppresses empty-usage warnings for very short pod
// sessions where 0 tokens is the normal outcome (open-then-immediate-close).
// MUST stay in sync with backend/internal/service/tokenusage/service.go's
// constant of the same name — separate processes, separate compilation
// units, no shared SSOT possible without a circular dependency.
const shortPodGracePeriod = 5 * time.Second

// Collect gathers token usage for a pod session.
// Returns nil when there is no parser, parsing fails, or the parser found
// no data — callers should treat nil as "do not send a report".
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
		logEmptyUsage(log, agent, sandboxPath, podStartedAt)
		return nil
	}

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

// logEmptyUsage promotes "no token usage" to Info for sessions that ran long
// enough that zero tokens is suspicious (e.g. parser format drift), keeping
// short-lived pods at Debug to avoid noise. A zero podStartedAt is treated
// as a caller bug (Warn) rather than collapsed into the short-pod case —
// silent downgrade defeats the whole purpose of this signal.
func logEmptyUsage(log *slog.Logger, agent, sandboxPath string, podStartedAt time.Time) {
	if podStartedAt.IsZero() {
		log.Warn("Token usage empty after parse: caller passed zero podStartedAt",
			"agent", agent,
			"sandbox_path", sandboxPath,
		)
		return
	}
	runtime := time.Since(podStartedAt)
	args := []any{
		"agent", agent,
		"sandbox_path", sandboxPath,
		"pod_runtime_seconds", runtime.Seconds(),
	}
	if runtime < shortPodGracePeriod {
		log.Debug("Token usage empty after parse", args...)
		return
	}
	log.Info("Token usage empty after parse", args...)
}
