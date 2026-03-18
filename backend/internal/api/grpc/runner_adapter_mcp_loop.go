package grpc

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	loopDomain "github.com/anthropics/agentsmesh/backend/internal/domain/loop"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	loopService "github.com/anthropics/agentsmesh/backend/internal/service/loop"
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

// ==================== Loop MCP Methods ====================

// mcpListLoops handles the "list_loops" MCP method.
func (a *GRPCRunnerAdapter) mcpListLoops(ctx context.Context, tc *middleware.TenantContext, payload []byte) (interface{}, *mcpError) {
	if a.loopService == nil {
		return nil, newMcpError(500, "loop service not available")
	}

	var params struct {
		Status string `json:"status"`
		Query  string `json:"query"`
		Limit  int    `json:"limit"`
		Offset int    `json:"offset"`
	}
	if err := unmarshalPayload(payload, &params); err != nil {
		return nil, err
	}

	limit := params.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset := params.Offset
	if offset < 0 {
		offset = 0
	}

	loops, _, err := a.loopService.List(ctx, &loopDomain.ListFilter{
		OrganizationID: tc.OrganizationID,
		Status:         params.Status,
		Query:          params.Query,
		Limit:          limit,
		Offset:         offset,
	})
	if err != nil {
		return nil, newMcpError(500, "failed to list loops")
	}

	// Enrich with active run counts
	if len(loops) > 0 && a.loopRunService != nil {
		loopIDs := make([]int64, len(loops))
		for i, l := range loops {
			loopIDs[i] = l.ID
		}
		if counts, err := a.loopRunService.CountActiveRunsByLoopIDs(ctx, loopIDs); err == nil {
			for _, l := range loops {
				if count, ok := counts[l.ID]; ok {
					l.ActiveRunCount = int(count)
				}
			}
		}
	}

	summaries := make([]*mcpLoopSummary, len(loops))
	for i, l := range loops {
		summaries[i] = toMCPLoopSummary(l)
	}

	return map[string]interface{}{"loops": summaries}, nil
}

// mcpTriggerLoop handles the "trigger_loop" MCP method.
func (a *GRPCRunnerAdapter) mcpTriggerLoop(ctx context.Context, tc *middleware.TenantContext, payload []byte) (interface{}, *mcpError) {
	if a.loopService == nil || a.loopOrchestrator == nil {
		return nil, newMcpError(500, "loop service not available")
	}

	var params struct {
		LoopSlug  string          `json:"loop_slug"`
		Variables json.RawMessage `json:"variables"`
	}
	if err := unmarshalPayload(payload, &params); err != nil {
		return nil, err
	}

	if params.LoopSlug == "" {
		return nil, newMcpError(400, "loop_slug is required")
	}

	loop, err := a.loopService.GetBySlug(ctx, tc.OrganizationID, params.LoopSlug)
	if err != nil {
		if errors.Is(err, loopService.ErrLoopNotFound) {
			return nil, newMcpError(404, "loop not found")
		}
		return nil, newMcpError(500, "failed to get loop")
	}

	result, err := a.loopOrchestrator.TriggerRun(ctx, &loopService.TriggerRunRequest{
		LoopID:        loop.ID,
		TriggerType:   loopDomain.RunTriggerManual,
		TriggerSource: "pod:" + strconv.FormatInt(tc.UserID, 10),
		TriggerParams: params.Variables,
	})
	if err != nil {
		if errors.Is(err, loopService.ErrLoopDisabled) {
			return nil, newMcpError(400, "loop is disabled")
		}
		return nil, newMcpError(500, "failed to trigger loop")
	}

	if result.Skipped {
		return map[string]interface{}{
			"run":     toMCPRunSummary(result.Run),
			"skipped": true,
			"reason":  result.Reason,
		}, nil
	}

	// Start run asynchronously (same pattern as loop_handler.go)
	startCtx, startCancel := context.WithTimeout(context.Background(), 5*time.Minute)
	go func() {
		defer startCancel()
		a.loopOrchestrator.StartRun(startCtx, result.Loop, result.Run, tc.UserID)
	}()

	return map[string]interface{}{
		"run": toMCPRunSummary(result.Run),
	}, nil
}
