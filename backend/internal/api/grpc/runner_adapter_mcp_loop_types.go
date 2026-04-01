package grpc

import (
	"time"

	loopDomain "github.com/anthropics/agentsmesh/backend/internal/domain/loop"
)

// mcpLoopSummary is a token-efficient Loop representation for MCP responses.
type mcpLoopSummary struct {
	Slug           string  `json:"slug"`
	Name           string  `json:"name"`
	Description    string  `json:"description,omitempty"`
	Status         string  `json:"status"`
	ExecutionMode  string  `json:"execution_mode"`
	CronExpression string  `json:"cron_expression,omitempty"`
	TotalRuns      int     `json:"total_runs"`
	SuccessfulRuns int     `json:"successful_runs"`
	FailedRuns     int     `json:"failed_runs"`
	ActiveRunCount int     `json:"active_run_count"`
	LastRunAt      string  `json:"last_run_at,omitempty"`
	NextRunAt      string  `json:"next_run_at,omitempty"`
	CreatedAt      string  `json:"created_at"`
}

func toMCPLoopSummary(l *loopDomain.Loop) *mcpLoopSummary {
	s := &mcpLoopSummary{
		Slug:           l.Slug,
		Name:           l.Name,
		Status:         l.Status,
		ExecutionMode:  l.ExecutionMode,
		TotalRuns:      l.TotalRuns,
		SuccessfulRuns: l.SuccessfulRuns,
		FailedRuns:     l.FailedRuns,
		ActiveRunCount: l.ActiveRunCount,
		CreatedAt:      l.CreatedAt.Format(time.RFC3339),
	}
	if l.Description != nil {
		s.Description = *l.Description
	}
	if l.CronExpression != nil {
		s.CronExpression = *l.CronExpression
	}
	if l.LastRunAt != nil {
		s.LastRunAt = l.LastRunAt.Format(time.RFC3339)
	}
	if l.NextRunAt != nil {
		s.NextRunAt = l.NextRunAt.Format(time.RFC3339)
	}
	return s
}

// mcpRunSummary is a token-efficient LoopRun representation for MCP responses.
type mcpRunSummary struct {
	ID          int64  `json:"id"`
	RunNumber   int    `json:"run_number"`
	Status      string `json:"status"`
	TriggerType string `json:"trigger_type"`
	PodKey      string `json:"pod_key,omitempty"`
	StartedAt   string `json:"started_at,omitempty"`
	FinishedAt  string `json:"finished_at,omitempty"`
	DurationSec *int   `json:"duration_sec,omitempty"`
	CreatedAt   string `json:"created_at"`
}

func toMCPRunSummary(r *loopDomain.LoopRun) *mcpRunSummary {
	s := &mcpRunSummary{
		ID:          r.ID,
		RunNumber:   r.RunNumber,
		Status:      r.Status,
		TriggerType: r.TriggerType,
		DurationSec: r.DurationSec,
		CreatedAt:   r.CreatedAt.Format(time.RFC3339),
	}
	if r.PodKey != nil {
		s.PodKey = *r.PodKey
	}
	if r.StartedAt != nil {
		s.StartedAt = r.StartedAt.Format(time.RFC3339)
	}
	if r.FinishedAt != nil {
		s.FinishedAt = r.FinishedAt.Format(time.RFC3339)
	}
	return s
}
