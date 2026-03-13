package loop

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	loopDomain "github.com/anthropics/agentsmesh/backend/internal/domain/loop"
	"github.com/anthropics/agentsmesh/backend/internal/infra/eventbus"
)

// TriggerRunRequest represents a request to trigger a loop run
type TriggerRunRequest struct {
	LoopID        int64
	TriggerType   string // "cron", "api", "manual"
	TriggerSource string
	TriggerParams json.RawMessage // Optional runtime variable overrides
}

// TriggerRunResult represents the result of triggering a loop run
type TriggerRunResult struct {
	Run     *loopDomain.LoopRun
	Loop    *loopDomain.Loop // the Loop definition (for callers that need it)
	Skipped bool
	Reason  string
}

// TriggerRun triggers a new run of a Loop.
//
// Delegates to the repository's TriggerRunAtomic for transactional safety
// (FOR UPDATE lock, concurrency check via Pod SSOT, atomic run_number generation).
func (o *LoopOrchestrator) TriggerRun(ctx context.Context, req *TriggerRunRequest) (*TriggerRunResult, error) {
	atomicResult, err := o.loopRunService.TriggerRunAtomic(ctx, &loopDomain.TriggerRunAtomicParams{
		LoopID:        req.LoopID,
		TriggerType:   req.TriggerType,
		TriggerSource: req.TriggerSource,
		TriggerParams: req.TriggerParams,
	})
	if err != nil {
		// Map domain error to service-level sentinel for handler consumption
		if errors.Is(err, loopDomain.ErrLoopDisabled) {
			return nil, ErrLoopDisabled
		}
		return nil, err
	}

	result := &TriggerRunResult{
		Run:     atomicResult.Run,
		Loop:    atomicResult.Loop,
		Skipped: atomicResult.Skipped,
		Reason:  atomicResult.Reason,
	}

	// Publish event outside transaction to avoid holding locks during event dispatch
	if result.Run != nil && atomicResult.Loop != nil {
		if result.Skipped {
			// Skipped runs still count toward total_runs so denormalized counters stay in sync
			// with SSOT (ComputeLoopStats). The "skipped" status doesn't increment
			// successful_runs or failed_runs — only total_runs and last_run_at.
			_ = o.loopService.UpdateRunStats(ctx, atomicResult.Loop.ID, loopDomain.RunStatusSkipped, time.Now())
		} else {
			o.publishRunEvent(atomicResult.Loop.OrganizationID, eventbus.EventLoopRunStarted, result.Run)
			o.logger.Info("loop run triggered",
				"loop_id", atomicResult.Loop.ID,
				"loop_slug", atomicResult.Loop.Slug,
				"run_id", result.Run.ID,
				"run_number", result.Run.RunNumber,
				"trigger_type", req.TriggerType,
			)
		}
	}

	return result, nil
}
