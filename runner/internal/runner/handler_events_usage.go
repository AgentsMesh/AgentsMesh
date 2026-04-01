package runner

import (
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/tokenusage"
)

// collectAndSendTokenUsage collects token usage and sends it to the backend.
// This is called asynchronously after pod termination and must never panic.
func (h *RunnerMessageHandler) collectAndSendTokenUsage(podKey, agent, sandboxPath string, podStartedAt time.Time) {
	log := logger.Pod()

	if h == nil || h.conn == nil {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			log.Error("Panic in token usage collection", "pod_key", podKey, "panic", r)
		}
	}()

	usage := tokenusage.Collect(agent, sandboxPath, podStartedAt)
	if usage == nil {
		return
	}

	models := make([]*runnerv1.TokenModelUsage, 0, len(usage.Models))
	for _, m := range usage.Sorted() {
		models = append(models, &runnerv1.TokenModelUsage{
			Model:               m.Model,
			InputTokens:         m.InputTokens,
			OutputTokens:        m.OutputTokens,
			CacheCreationTokens: m.CacheCreationTokens,
			CacheReadTokens:     m.CacheReadTokens,
		})
	}

	if err := h.conn.SendTokenUsage(podKey, models); err != nil {
		log.Warn("Failed to send token usage report", "pod_key", podKey, "error", err)
	} else {
		log.Info("Token usage report sent", "pod_key", podKey, "models", len(models))
	}
}
