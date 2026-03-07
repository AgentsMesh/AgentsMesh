package loop

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	loopDomain "github.com/anthropics/agentsmesh/backend/internal/domain/loop"
	"github.com/anthropics/agentsmesh/backend/internal/infra/eventbus"
	agentpodSvc "github.com/anthropics/agentsmesh/backend/internal/service/agentpod"
	ticketSvc "github.com/anthropics/agentsmesh/backend/internal/service/ticket"
)

// PodTerminator defines the minimal interface needed by LoopOrchestrator
// to terminate Pods (used for timeout handling).
type PodTerminator interface {
	TerminatePod(ctx context.Context, podKey string) error
}

// LoopOrchestrator orchestrates the full lifecycle of a Loop run:
//   - TriggerRun:          atomic run record creation (FOR UPDATE + SSOT concurrency check)
//   - StartRun:            Pod creation + optional Autopilot setup
//   - HandleRunCompleted:  stats update, runtime state (last_pod_key), event publishing
//
// Architecture: Pod is the Single Source of Truth (SSOT) for execution status.
// The orchestrator creates LoopRun records and associates them with Pods,
// but does NOT maintain run status independently. Status is always derived
// from Pod state when queried.
type LoopOrchestrator struct {
	loopService    *LoopService
	loopRunService *LoopRunService
	eventBus       *eventbus.EventBus
	logger         *slog.Logger

	// External dependencies (injected after construction)
	podOrchestrator *agentpodSvc.PodOrchestrator
	autopilotSvc    *agentpodSvc.AutopilotControllerService
	podTerminator   PodTerminator // for terminating timed-out Pods
	ticketService   *ticketSvc.Service

	// HTTP client for webhook callbacks (reused across calls)
	httpClient *http.Client
}

// NewLoopOrchestrator creates a new LoopOrchestrator
func NewLoopOrchestrator(
	loopService *LoopService,
	loopRunService *LoopRunService,
	eventBus *eventbus.EventBus,
	logger *slog.Logger,
) *LoopOrchestrator {
	return &LoopOrchestrator{
		loopService:    loopService,
		loopRunService: loopRunService,
		eventBus:       eventBus,
		logger:         logger.With("component", "loop_orchestrator"),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			// Disable redirects to prevent SSRF bypass via HTTP redirect to internal IPs.
			// The initial callback_url is validated against private ranges at create/update time,
			// but a redirect could point to an internal address, bypassing that check.
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

// SetPodDependencies injects Pod-related dependencies after construction.
// Called from main.go after PodOrchestrator and PodCoordinator are available.
func (o *LoopOrchestrator) SetPodDependencies(
	podOrch *agentpodSvc.PodOrchestrator,
	autopilot *agentpodSvc.AutopilotControllerService,
	podTerminator PodTerminator,
	ticket *ticketSvc.Service,
) {
	o.podOrchestrator = podOrch
	o.autopilotSvc = autopilot
	o.podTerminator = podTerminator
	o.ticketService = ticket
}

// ========== Trigger ==========

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

// ========== Start Run (Pod + Autopilot creation) ==========

// StartRun creates a Pod and optionally an AutopilotController for the loop run.
// Should be called asynchronously (in a goroutine) after TriggerRun returns successfully.
func (o *LoopOrchestrator) StartRun(ctx context.Context, loop *loopDomain.Loop, run *loopDomain.LoopRun, userID int64) {
	// Panic recovery — this method is always called in a goroutine, so panics would crash the process
	defer func() {
		if r := recover(); r != nil {
			o.logger.Error("panic in StartRun", "run_id", run.ID, "loop_id", loop.ID, "panic", r)
			_ = o.MarkRunFailed(ctx, run.ID, fmt.Sprintf("Internal error: %v", r))
		}
	}()

	if o.podOrchestrator == nil {
		o.logger.Error("pod orchestrator not set, cannot start run", "run_id", run.ID)
		_ = o.MarkRunFailed(ctx, run.ID, "Pod orchestrator not configured")
		return
	}

	// Check if the run was cancelled between TriggerRun and StartRun
	// (e.g., user cancelled a pending run before the goroutine started)
	currentRun, err := o.loopRunService.GetByID(ctx, run.ID)
	if err != nil {
		o.logger.Error("failed to check run status before start", "run_id", run.ID, "error", err)
		return
	}
	if currentRun.FinishedAt != nil || currentRun.IsTerminal() {
		o.logger.Info("run already finished/cancelled before StartRun, skipping",
			"run_id", run.ID, "status", currentRun.Status)
		return
	}

	// Determine runner ID
	var runnerID int64
	if loop.RunnerID != nil {
		runnerID = *loop.RunnerID
	}

	// Determine permission mode
	permissionMode := loop.PermissionMode
	if permissionMode == "" {
		permissionMode = "bypassPermissions"
	}

	// Build config overrides
	var configOverrides map[string]interface{}
	if loop.ConfigOverrides != nil {
		_ = json.Unmarshal(loop.ConfigOverrides, &configOverrides)
	}

	// Resolve prompt: merge default variables with trigger-time overrides, then substitute {{key}} placeholders
	resolvedPrompt := resolvePrompt(loop.PromptTemplate, loop.PromptVariables, run.TriggerParams)

	// Persist resolved prompt on the run record
	if err := o.loopRunService.UpdateStatus(ctx, run.ID, map[string]interface{}{
		"resolved_prompt": resolvedPrompt,
	}); err != nil {
		o.logger.Error("failed to persist resolved prompt", "run_id", run.ID, "error", err)
	}

	// Determine source pod key for resume (persistent sandbox strategy)
	var sourcePodKey string
	resumeSession := loop.SessionPersistence
	if loop.IsPersistent() && loop.LastPodKey != nil {
		sourcePodKey = *loop.LastPodKey
	}

	// Create Pod via PodOrchestrator
	podResult, err := o.podOrchestrator.CreatePod(ctx, &agentpodSvc.OrchestrateCreatePodRequest{
		OrganizationID:      loop.OrganizationID,
		UserID:              userID,
		RunnerID:            runnerID,
		AgentTypeID:         loop.AgentTypeID,
		CustomAgentTypeID:   loop.CustomAgentTypeID,
		RepositoryID:        loop.RepositoryID,
		TicketID:            loop.TicketID,
		InitialPrompt:       resolvedPrompt,
		BranchName:          loop.BranchName,
		PermissionMode:      &permissionMode,
		CredentialProfileID: loop.CredentialProfileID,
		ConfigOverrides:     configOverrides,
		Cols:                120,
		Rows:                40,
		SourcePodKey:        sourcePodKey,
		ResumeAgentSession:  &resumeSession,
	})
	if err != nil {
		// M3: If resume mode failed, retry without resume (degrade to fresh sandbox)
		if sourcePodKey != "" {
			o.logger.Warn("persistent sandbox resume failed, degrading to fresh",
				"loop_id", loop.ID, "run_id", run.ID, "source_pod_key", sourcePodKey, "error", err)

			// Notify frontend about the degradation
			o.publishWarningEvent(loop.OrganizationID, loop.ID, run.ID, run.RunNumber,
				"sandbox_resume_degraded",
				fmt.Sprintf("Resume from pod %s failed: %v. Degraded to fresh sandbox.", sourcePodKey, err))

			podResult, err = o.podOrchestrator.CreatePod(ctx, &agentpodSvc.OrchestrateCreatePodRequest{
				OrganizationID:      loop.OrganizationID,
				UserID:              userID,
				RunnerID:            runnerID,
				AgentTypeID:         loop.AgentTypeID,
				CustomAgentTypeID:   loop.CustomAgentTypeID,
				RepositoryID:        loop.RepositoryID,
				TicketID:            loop.TicketID,
				InitialPrompt:       resolvedPrompt,
				BranchName:          loop.BranchName,
				PermissionMode:      &permissionMode,
				CredentialProfileID: loop.CredentialProfileID,
				ConfigOverrides:     configOverrides,
				Cols:                120,
				Rows:                40,
				// No SourcePodKey — fresh start
			})
			if err != nil {
				_ = o.MarkRunFailed(ctx, run.ID, fmt.Sprintf("Pod creation failed (after resume degradation): %v", err))
				return
			}

			// Clear the stale resume chain so future runs don't keep failing
			_ = o.loopService.ClearRuntimeState(ctx, loop.ID)
		} else {
			_ = o.MarkRunFailed(ctx, run.ID, fmt.Sprintf("Pod creation failed: %v", err))
			return
		}
	}

	pod := podResult.Pod
	autopilotKey := ""

	// If autopilot mode, create AutopilotController via the encapsulated service method
	if loop.IsAutopilot() && o.autopilotSvc != nil {
		var err error
		autopilotKey, err = o.startAutopilot(ctx, loop, run, pod, resolvedPrompt)
		if err != nil {
			o.logger.Error("autopilot creation failed, terminating Pod",
				"run_id", run.ID, "pod_key", pod.PodKey, "error", err)
			// Terminate the orphan Pod — nothing will drive it without Autopilot
			if o.podTerminator != nil {
				_ = o.podTerminator.TerminatePod(ctx, pod.PodKey)
			}
			_ = o.MarkRunFailed(ctx, run.ID, fmt.Sprintf("Autopilot creation failed: %v", err))
			return
		}
	}

	// Associate Pod with run — after this, run status is derived from Pod (SSOT)
	if err := o.SetRunPodKey(ctx, run.ID, pod.PodKey, autopilotKey); err != nil {
		o.logger.Error("failed to set run pod key", "run_id", run.ID, "error", err)
	}

	o.logger.Info("loop run started",
		"loop_id", loop.ID,
		"run_id", run.ID,
		"pod_key", pod.PodKey,
		"autopilot_key", autopilotKey,
		"execution_mode", loop.ExecutionMode,
	)
}

// startAutopilot delegates Autopilot creation to AutopilotControllerService.CreateAndStart.
// Returns the autopilot controller key and any error.
func (o *LoopOrchestrator) startAutopilot(ctx context.Context, loop *loopDomain.Loop, run *loopDomain.LoopRun, pod *agentpod.Pod, resolvedPrompt string) (string, error) {
	// Extract autopilot config via typed struct (all zeros → domain defaults apply)
	apCfg := loop.ParseAutopilotConfig()

	controller, err := o.autopilotSvc.CreateAndStart(ctx, &agentpodSvc.CreateAndStartRequest{
		OrganizationID:      loop.OrganizationID,
		Pod:                 pod,
		InitialPrompt:       resolvedPrompt,
		MaxIterations:       apCfg.MaxIterations,
		IterationTimeoutSec: apCfg.IterationTimeoutSec,
		NoProgressThreshold: apCfg.NoProgressThreshold,
		SameErrorThreshold:  apCfg.SameErrorThreshold,
		ApprovalTimeoutMin:  apCfg.ApprovalTimeoutMin,
		KeyPrefix:           fmt.Sprintf("loop-%s-run%d", loop.Slug, run.RunNumber),
	})
	if err != nil {
		return "", fmt.Errorf("failed to create autopilot controller: %w", err)
	}

	return controller.AutopilotControllerKey, nil
}

// ========== Run Lifecycle ==========

// SetRunPodKey associates a Pod with a run and marks it as running.
// After this, the run's effective status is derived from Pod (SSOT).
func (o *LoopOrchestrator) SetRunPodKey(ctx context.Context, runID int64, podKey string, autopilotKey string) error {
	updates := map[string]interface{}{
		"pod_key": podKey,
	}
	if autopilotKey != "" {
		updates["autopilot_controller_key"] = autopilotKey
	}
	return o.loopRunService.UpdateStatus(ctx, runID, updates)
}

// MarkRunFailed marks a run as failed when Pod creation or Autopilot setup fails.
// This is only used when no Pod exists (or Pod was cleaned up) — because no SSOT is available.
func (o *LoopOrchestrator) MarkRunFailed(ctx context.Context, runID int64, errorMessage string) error {
	return o.markRunTerminal(ctx, runID, loopDomain.RunStatusFailed, errorMessage)
}

// MarkRunCancelled marks a run as cancelled (e.g. user-initiated cancellation of a pending run).
func (o *LoopOrchestrator) MarkRunCancelled(ctx context.Context, runID int64, reason string) error {
	return o.markRunTerminal(ctx, runID, loopDomain.RunStatusCancelled, reason)
}

// markRunTerminal sets a run to a terminal state directly (bypassing Pod SSOT).
// Used only when no Pod exists to derive status from.
// Uses FinishRun (WHERE finished_at IS NULL) for idempotency — concurrent calls are no-ops.
func (o *LoopOrchestrator) markRunTerminal(ctx context.Context, runID int64, status string, errorMessage string) error {
	now := time.Now()
	updates := map[string]interface{}{
		"status":        status,
		"finished_at":   now,
		"error_message": errorMessage,
	}
	updated, err := o.loopRunService.FinishRun(ctx, runID, updates)
	if err != nil {
		return err
	}
	if !updated {
		// Already finished by a concurrent call — skip side-effects
		return nil
	}

	// Publish event — cancelled counts as failed (consistent with stats)
	run, _ := o.loopRunService.GetByID(ctx, runID)
	if run != nil {
		o.publishRunEvent(run.OrganizationID, eventbus.EventLoopRunFailed, run)
		// Update loop stats (incremental)
		_ = o.loopService.UpdateRunStats(ctx, run.LoopID, status, now)
	}
	return nil
}

// HandleRunCompleted processes a completed loop run.
// Called when a Pod or Autopilot reaches a terminal state.
//
// Responsibilities:
//  1. Update Loop run statistics (incremental counters)
//  2. Update runtime state (last_pod_key for persistent sandbox resume)
//  3. Publish completion event
func (o *LoopOrchestrator) HandleRunCompleted(ctx context.Context, run *loopDomain.LoopRun, effectiveStatus string) {
	now := time.Now()

	// 0. Atomically mark the run as finished (optimistic locking via WHERE finished_at IS NULL).
	// If another concurrent event already finished this run, FinishRun returns false and we skip
	// all side-effects (stats, events, webhooks) to prevent double-counting.
	runUpdates := map[string]interface{}{
		"status":      effectiveStatus,
		"finished_at": now,
	}
	if run.StartedAt != nil {
		durationSec := int(now.Sub(*run.StartedAt).Seconds())
		runUpdates["duration_sec"] = durationSec
	}
	updated, err := o.loopRunService.FinishRun(ctx, run.ID, runUpdates)
	if err != nil {
		o.logger.Error("failed to mark run as finished",
			"run_id", run.ID, "error", err)
		return
	}
	if !updated {
		// Already finished by a concurrent event — skip all side-effects
		o.logger.Debug("run already finished, skipping duplicate completion",
			"run_id", run.ID)
		return
	}

	// Sync the in-memory struct so downstream consumers (publishRunEvent, webhook, ticket comment)
	// see the resolved status instead of the stale DB value (e.g. "pending").
	run.Status = effectiveStatus
	run.FinishedAt = &now

	// 1. Update loop run statistics (incremental)
	if err := o.loopService.UpdateRunStats(ctx, run.LoopID, effectiveStatus, now); err != nil {
		o.logger.Error("failed to update loop run stats",
			"loop_id", run.LoopID, "run_id", run.ID, "error", err)
	}

	// 2. Update runtime state for persistent sandbox resume
	loop, _ := o.loopService.GetByID(ctx, run.LoopID)
	if run.PodKey != nil && loop != nil && loop.IsPersistent() {
		if err := o.loopService.UpdateRuntimeState(ctx, run.LoopID, nil, run.PodKey); err != nil {
			o.logger.Error("failed to update loop runtime state",
				"loop_id", run.LoopID, "error", err)
		}
	}

	// 3. Publish completion event — cancelled/failed/timeout all emit RunFailed
	eventType := eventbus.EventLoopRunCompleted
	if effectiveStatus == loopDomain.RunStatusFailed || effectiveStatus == loopDomain.RunStatusTimeout || effectiveStatus == loopDomain.RunStatusCancelled {
		eventType = eventbus.EventLoopRunFailed
	}
	o.publishRunEvent(run.OrganizationID, eventType, run)

	// 4. Webhook callback (async, fire-and-forget)
	if loop != nil && loop.CallbackURL != nil && *loop.CallbackURL != "" {
		go o.sendWebhookCallback(*loop.CallbackURL, loop, run, effectiveStatus)
	}

	// 5. Ticket comment (async — use background context since the parent ctx may be cancelled)
	if loop != nil && loop.TicketID != nil && o.ticketService != nil {
		go o.postTicketComment(context.Background(), *loop.TicketID, loop.CreatedByID, loop, run, effectiveStatus)
	}

	// 6. Data retention — trim old finished runs if max_retained_runs is configured
	if loop != nil && loop.MaxRetainedRuns > 0 {
		if deleted, err := o.loopRunService.DeleteOldFinishedRuns(ctx, loop.ID, loop.MaxRetainedRuns); err != nil {
			o.logger.Error("failed to trim old loop runs",
				"loop_id", loop.ID, "max_retained", loop.MaxRetainedRuns, "error", err)
		} else if deleted > 0 {
			o.logger.Info("trimmed old loop runs",
				"loop_id", loop.ID, "deleted", deleted, "max_retained", loop.MaxRetainedRuns)
		}
	}

	o.logger.Info("loop run completed",
		"loop_id", run.LoopID,
		"run_id", run.ID,
		"effective_status", effectiveStatus,
		"pod_key", run.PodKey,
	)
}

// HandlePodTerminated is called when a Pod reaches a terminal state.
// It looks up the associated LoopRun and processes completion.
//
// Uses FindActiveRunByPodKey (no status resolution) because the event payload
// carries the authoritative podStatus — re-querying Pod status would be redundant.
func (o *LoopOrchestrator) HandlePodTerminated(ctx context.Context, podKey string, podStatus string, podFinishedAt *time.Time) {
	run, err := o.loopRunService.FindActiveRunByPodKey(ctx, podKey)
	if err != nil {
		// Not a loop-associated pod, ignore
		return
	}

	// Derive effective status using SSOT logic
	autopilotPhase := ""
	if run.AutopilotControllerKey != nil {
		autopilotPhase = o.loopRunService.GetAutopilotPhase(ctx, *run.AutopilotControllerKey)
	}
	effectiveStatus := loopDomain.DeriveRunStatus(podStatus, autopilotPhase)

	// Only process if the run reached a terminal state
	if effectiveStatus == loopDomain.RunStatusRunning {
		return
	}

	o.HandleRunCompleted(ctx, run, effectiveStatus)
}

// HandleAutopilotTerminated is called when an Autopilot reaches a terminal phase.
// It looks up the associated LoopRun and processes completion.
//
// Uses FindActiveRunByAutopilotKey (no status resolution) because the event payload
// carries the authoritative phase — re-querying would be redundant.
// Delegates to DeriveRunStatus for status mapping (SSOT — single mapping location).
func (o *LoopOrchestrator) HandleAutopilotTerminated(ctx context.Context, autopilotKey string, phase string) {
	if !agentpod.IsAutopilotPhaseTerminal(phase) {
		return // Not terminal, ignore
	}

	run, err := o.loopRunService.FindActiveRunByAutopilotKey(ctx, autopilotKey)
	if err != nil {
		// Not a loop-associated autopilot, ignore
		return
	}

	// Delegate to DeriveRunStatus for consistent mapping (SSOT)
	// Pod status is irrelevant when autopilot phase is terminal — DeriveRunStatus handles this.
	effectiveStatus := loopDomain.DeriveRunStatus("", phase)

	o.HandleRunCompleted(ctx, run, effectiveStatus)
}

// ========== Timeout Detection ==========

// CheckTimeoutRuns detects loop runs that have exceeded their timeout and marks them as timed out.
// orgIDs filters to specific organizations; nil means all orgs (single-instance mode).
// Called periodically by the LoopScheduler.
func (o *LoopOrchestrator) CheckTimeoutRuns(ctx context.Context, orgIDs []int64) error {
	runs, err := o.loopRunService.GetTimedOutRuns(ctx, orgIDs)
	if err != nil {
		o.logger.Error("failed to get timed out runs", "error", err)
		return err
	}

	if len(runs) == 0 {
		return nil
	}

	o.logger.Info("found timed out loop runs", "count", len(runs))

	for _, run := range runs {
		o.HandleRunCompleted(ctx, run, loopDomain.RunStatusTimeout)

		// Terminate the Pod if podTerminator is available
		if run.PodKey != nil && o.podTerminator != nil {
			if termErr := o.podTerminator.TerminatePod(ctx, *run.PodKey); termErr != nil {
				o.logger.Error("failed to terminate timed out pod",
					"pod_key", *run.PodKey,
					"run_id", run.ID,
					"error", termErr,
				)
			}
		}

		o.logger.Info("marked loop run as timed out",
			"run_id", run.ID,
			"loop_id", run.LoopID,
			"pod_key", run.PodKey,
		)
	}

	return nil
}

// ========== Orphan Cleanup ==========

// CleanupOrphanPendingRuns marks pending runs with no Pod that are stuck for > 5 minutes as failed.
// These can occur when StartRun goroutine crashes or the server restarts between TriggerRun and StartRun.
// orgIDs filters to specific organizations; nil means all orgs (single-instance mode).
func (o *LoopOrchestrator) CleanupOrphanPendingRuns(ctx context.Context, orgIDs []int64) error {
	runs, err := o.loopRunService.GetOrphanPendingRuns(ctx, orgIDs)
	if err != nil {
		return err
	}
	if len(runs) == 0 {
		return nil
	}

	o.logger.Info("cleaning up orphan pending runs", "count", len(runs))
	for _, run := range runs {
		_ = o.MarkRunFailed(ctx, run.ID, "Orphan pending run: Pod was never created (server restart or StartRun failure)")
		o.logger.Warn("marked orphan pending run as failed", "run_id", run.ID, "loop_id", run.LoopID)
	}
	return nil
}

// ========== Stats ==========

// RefreshLoopStats recomputes loop statistics from Pod status (SSOT).
// Call this periodically or after significant events.
func (o *LoopOrchestrator) RefreshLoopStats(ctx context.Context, loopID int64) error {
	total, successful, failed, err := o.loopRunService.ComputeLoopStats(ctx, loopID)
	if err != nil {
		return fmt.Errorf("failed to compute loop stats: %w", err)
	}

	return o.loopService.UpdateStats(ctx, loopID, total, successful, failed)
}

// GetLastPodKey returns the pod_key from the most recent run that has one.
// Used for persistent sandbox resume.
func (o *LoopOrchestrator) GetLastPodKey(ctx context.Context, loopID int64) *string {
	return o.loopRunService.GetLatestPodKey(ctx, loopID)
}

// ========== Events ==========

// publishRunEvent publishes a loop run event to the event bus
func (o *LoopOrchestrator) publishRunEvent(orgID int64, eventType eventbus.EventType, run *loopDomain.LoopRun) {
	if o.eventBus == nil {
		return
	}

	data, _ := json.Marshal(map[string]interface{}{
		"loop_id":    run.LoopID,
		"run_id":     run.ID,
		"run_number": run.RunNumber,
		"status":     run.Status,
		"pod_key":    run.PodKey,
	})

	_ = o.eventBus.Publish(context.Background(), &eventbus.Event{
		Type:           eventType,
		Category:       eventbus.CategoryEntity,
		OrganizationID: orgID,
		EntityType:     "loop_run",
		EntityID:       fmt.Sprintf("%d", run.ID),
		Data:           data,
		Timestamp:      time.Now().UnixMilli(),
	})
}

// publishWarningEvent publishes a loop run warning event to the event bus
func (o *LoopOrchestrator) publishWarningEvent(orgID int64, loopID int64, runID int64, runNumber int, warning string, detail string) {
	if o.eventBus == nil {
		return
	}

	data, _ := json.Marshal(eventbus.LoopRunWarningData{
		LoopID:    loopID,
		RunID:     runID,
		RunNumber: runNumber,
		Warning:   warning,
		Detail:    detail,
	})

	_ = o.eventBus.Publish(context.Background(), &eventbus.Event{
		Type:           eventbus.EventLoopRunWarning,
		Category:       eventbus.CategoryEntity,
		OrganizationID: orgID,
		EntityType:     "loop_run",
		EntityID:       fmt.Sprintf("%d", runID),
		Data:           data,
		Timestamp:      time.Now().UnixMilli(),
	})
}

// ========== Prompt Resolution ==========

// resolvePrompt merges default variables with trigger-time overrides,
// then substitutes {{key}} placeholders in the template.
func resolvePrompt(template string, defaults json.RawMessage, overrides json.RawMessage) string {
	vars := make(map[string]interface{})
	if len(defaults) > 0 {
		_ = json.Unmarshal(defaults, &vars)
	}
	if len(overrides) > 0 {
		var ov map[string]interface{}
		if err := json.Unmarshal(overrides, &ov); err == nil {
			for k, v := range ov {
				vars[k] = v
			}
		}
	}

	result := template
	for k, v := range vars {
		placeholder := "{{" + k + "}}"
		result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", v))
	}
	return result
}

// ========== Webhook Callback ==========

// sendWebhookCallback POSTs run result to the loop's callback URL.
// Fire-and-forget: errors are logged but do not affect the run.
func (o *LoopOrchestrator) sendWebhookCallback(callbackURL string, loop *loopDomain.Loop, run *loopDomain.LoopRun, status string) {
	payload, _ := json.Marshal(map[string]interface{}{
		"loop_id":      loop.ID,
		"loop_slug":    loop.Slug,
		"loop_name":    loop.Name,
		"run_id":       run.ID,
		"run_number":   run.RunNumber,
		"status":       status,
		"trigger":      run.TriggerType,
		"exit_summary": run.ExitSummary,
		"started_at": func() string {
			if run.StartedAt != nil {
				return run.StartedAt.Format(time.RFC3339)
			}
			return ""
		}(),
		"finished_at": func() string {
			if run.FinishedAt != nil {
				return run.FinishedAt.Format(time.RFC3339)
			}
			return time.Now().Format(time.RFC3339)
		}(),
	})

	resp, err := o.httpClient.Post(callbackURL, "application/json", strings.NewReader(string(payload)))
	if err != nil {
		o.logger.Warn("webhook callback failed",
			"loop_id", loop.ID, "run_id", run.ID, "url", callbackURL, "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		o.logger.Warn("webhook callback returned error",
			"loop_id", loop.ID, "run_id", run.ID, "url", callbackURL, "status", resp.StatusCode)
	}
}

// ========== Ticket Comment ==========

// postTicketComment creates a comment on the associated ticket with run results.
func (o *LoopOrchestrator) postTicketComment(ctx context.Context, ticketID int64, userID int64, loop *loopDomain.Loop, run *loopDomain.LoopRun, status string) {
	statusEmoji := "✅"
	switch status {
	case loopDomain.RunStatusFailed:
		statusEmoji = "❌"
	case loopDomain.RunStatusTimeout:
		statusEmoji = "⏰"
	case loopDomain.RunStatusCancelled:
		statusEmoji = "⊘"
	}

	durationStr := "-"
	if run.StartedAt != nil && run.FinishedAt != nil {
		durationStr = fmt.Sprintf("%.0fs", run.FinishedAt.Sub(*run.StartedAt).Seconds())
	}

	content := fmt.Sprintf(
		"%s **Loop Run #%d** — %s\n\nLoop: **%s** (`%s`)\nDuration: %s\nTrigger: %s",
		statusEmoji, run.RunNumber, status, loop.Name, loop.Slug, durationStr, run.TriggerType,
	)

	if _, err := o.ticketService.CreateComment(ctx, ticketID, userID, content, nil, nil); err != nil {
		o.logger.Warn("failed to create ticket comment for loop run",
			"loop_id", loop.ID, "run_id", run.ID, "ticket_id", ticketID, "error", err)
	}
}
