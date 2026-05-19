package loop

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	loopDomain "github.com/anthropics/agentsmesh/backend/internal/domain/loop"
	"github.com/anthropics/agentsmesh/backend/internal/infra/eventbus"
)

type TriggerRunRequest struct {
	LoopID        int64
	TriggerType   string
	TriggerSource string
	TriggerParams json.RawMessage
}

type TriggerRunResult struct {
	Run     *loopDomain.LoopRun
	Loop    *loopDomain.Loop
	Skipped bool
	Reason  string
}

func (o *LoopOrchestrator) TriggerRun(ctx context.Context, req *TriggerRunRequest) (*TriggerRunResult, error) {
	atomicResult, err := o.loopRunService.TriggerRunAtomic(ctx, &loopDomain.TriggerRunAtomicParams{
		LoopID:        req.LoopID,
		TriggerType:   req.TriggerType,
		TriggerSource: req.TriggerSource,
		TriggerParams: req.TriggerParams,
	})
	if err != nil {
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

	if result.Run != nil && atomicResult.Loop != nil {
		if result.Skipped {
			// Skipped runs count toward total_runs so the denormalized counter stays in sync with ComputeLoopStats (SSOT).
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
