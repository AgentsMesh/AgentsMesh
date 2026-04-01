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
