// Package tokenusage provides token usage collection from AI agent session files.
// Parsers are best-effort: errors are logged but never block pod termination.
package tokenusage

import "time"

// ModelUsage holds aggregated token counts for a single model.
type ModelUsage struct {
	Model               string
	InputTokens         int64
	OutputTokens        int64
	CacheCreationTokens int64
	CacheReadTokens     int64
}

// TokenUsage holds the aggregated token usage across all models for a pod session.
type TokenUsage struct {
	// Models maps model name to its aggregated usage.
	Models map[string]*ModelUsage
}

// NewTokenUsage creates an empty TokenUsage.
func NewTokenUsage() *TokenUsage {
	return &TokenUsage{Models: make(map[string]*ModelUsage)}
}

// Add accumulates usage for a given model.
func (t *TokenUsage) Add(model string, input, output, cacheCreation, cacheRead int64) {
	m, ok := t.Models[model]
	if !ok {
		m = &ModelUsage{Model: model}
		t.Models[model] = m
	}
	m.InputTokens += input
	m.OutputTokens += output
	m.CacheCreationTokens += cacheCreation
	m.CacheReadTokens += cacheRead
}

// IsEmpty returns true if no usage was collected.
func (t *TokenUsage) IsEmpty() bool {
	return len(t.Models) == 0
}

// Sorted returns models sorted by name for deterministic output.
func (t *TokenUsage) Sorted() []*ModelUsage {
	if len(t.Models) == 0 {
		return nil
	}
	result := make([]*ModelUsage, 0, len(t.Models))
	for _, m := range t.Models {
		result = append(result, m)
	}
	// Sort by model name for deterministic ordering
	for i := 1; i < len(result); i++ {
		for j := i; j > 0 && result[j].Model < result[j-1].Model; j-- {
			result[j], result[j-1] = result[j-1], result[j]
		}
	}
	return result
}

// TokenParser defines the interface for agent-specific token usage parsers.
type TokenParser interface {
	// Parse collects token usage from session files.
	// sandboxPath is the pod's sandbox root directory.
	// podStartedAt is the pod's start time; only files modified after this time are processed.
	// Returns nil usage if no session files are found (not an error).
	Parse(sandboxPath string, podStartedAt time.Time) (*TokenUsage, error)
}
