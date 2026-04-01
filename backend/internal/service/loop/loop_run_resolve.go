package loop

import (
	"context"
	"log/slog"

	loopDomain "github.com/anthropics/agentsmesh/backend/internal/domain/loop"
)

// resolveRunStatuses resolves status for a batch of runs from Pod (SSOT).
func (s *LoopRunService) resolveRunStatuses(ctx context.Context, runs []*loopDomain.LoopRun) {
	podKeys := make([]string, 0)
	autopilotKeys := make([]string, 0)
	for _, run := range runs {
		if run.PodKey != nil {
			podKeys = append(podKeys, *run.PodKey)
		}
		if run.AutopilotControllerKey != nil {
			autopilotKeys = append(autopilotKeys, *run.AutopilotControllerKey)
		}
	}

	if len(podKeys) == 0 {
		return
	}

	podInfos, err := s.repo.BatchGetPodStatuses(ctx, podKeys)
	if err != nil {
		slog.Error("failed to batch get pod statuses for SSOT resolution", "error", err, "count", len(podKeys))
	}
	podMap := make(map[string]*loopDomain.PodStatusInfo, len(podInfos))
	for i := range podInfos {
		podMap[podInfos[i].PodKey] = &podInfos[i]
	}

	autopilotMap, err := s.repo.BatchGetAutopilotPhases(ctx, autopilotKeys)
	if err != nil {
		slog.Error("failed to batch get autopilot phases for SSOT resolution", "error", err, "count", len(autopilotKeys))
	}

	for _, run := range runs {
		resolveOneRunStatus(run, podMap, autopilotMap)
	}
}

// resolveOneRunStatus resolves the effective status for a single run using pod and autopilot data.
func resolveOneRunStatus(run *loopDomain.LoopRun, podMap map[string]*loopDomain.PodStatusInfo, autopilotMap map[string]string) {
	if run.PodKey == nil {
		return
	}
	// If finished_at is set, the run has been finalized by HandleRunCompleted.
	// Use the persisted status/duration directly instead of re-deriving from Pod.
	if run.FinishedAt != nil {
		return
	}
	pod, ok := podMap[*run.PodKey]
	if !ok {
		// Pod not found in DB - treat as failed (orphaned reference)
		run.Status = loopDomain.RunStatusFailed
		return
	}

	autopilotPhase := ""
	if run.AutopilotControllerKey != nil && autopilotMap != nil {
		autopilotPhase = autopilotMap[*run.AutopilotControllerKey]
	}

	ResolveRunStatus(run, pod.Status, autopilotPhase, pod.FinishedAt)
}

// resolveRunStatus resolves status for a single run from Pod (SSOT).
func (s *LoopRunService) resolveRunStatus(ctx context.Context, run *loopDomain.LoopRun) {
	if run.PodKey == nil {
		return
	}
	s.resolveRunStatuses(ctx, []*loopDomain.LoopRun{run})
}
