package agentpod

import (
	"context"
	"log/slog"
	"time"

	podDomain "github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	otelinit "github.com/anthropics/agentsmesh/backend/internal/infra/otel"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

func (o *PodOrchestrator) dispatchPod(
	ctx context.Context,
	req *OrchestrateCreatePodRequest,
	pod *podDomain.Pod,
	command *runnerv1.CreatePodCommand,
	sessionID string,
	isResumeMode bool,
) error {
	if o.podCoordinator == nil {
		slog.WarnContext(ctx, "PodCoordinator is nil, cannot dispatch create_pod", "pod_key", pod.PodKey)
		return nil
	}

	slog.InfoContext(ctx, "dispatching create_pod to runner", "runner_id", req.RunnerID, "pod_key", pod.PodKey, "session_id", sessionID, "resume", isResumeMode)
	recordDuration := otelinit.PodDispatchAttemptDuration.Enabled(ctx)
	dispatchStart := time.Time{}
	if recordDuration {
		dispatchStart = time.Now()
	}
	dispatchErr := o.podCoordinator.CreatePod(ctx, req.RunnerID, command)
	if recordDuration {
		otelinit.RecordPodDispatchAttemptDuration(ctx, time.Since(dispatchStart), dispatchErr)
	}
	if dispatchErr == nil {
		slog.InfoContext(ctx, "create_pod dispatched", "pod_key", pod.PodKey)
		return nil
	}

	slog.ErrorContext(ctx, "failed to dispatch create_pod", "pod_key", pod.PodKey, "error", dispatchErr)
	if markErr := o.podService.MarkInitFailed(ctx, pod.PodKey, errCodeRunnerUnreachable,
		"Failed to dispatch pod to runner: "+dispatchErr.Error()); markErr != nil {
		slog.ErrorContext(ctx, "failed to mark pod as init failed", "pod_key", pod.PodKey, "error", markErr)
	}
	return ErrRunnerDispatchFailed
}
