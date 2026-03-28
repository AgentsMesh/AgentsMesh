package runner

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	runnerDomain "github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// Terminate command backoff constants
const (
	terminateCooldown     = 5 * time.Minute
	terminateCacheCleanup = 30 * time.Minute

	// orphanMissThreshold: consecutive heartbeat misses before marking pod orphaned.
	// At ~30s per heartbeat cycle, 3 misses ~ 90s.
	orphanMissThreshold = 3

	// initRecoverThreshold: consecutive heartbeat reports of "initializing" pod
	// before recovering to "running".
	initRecoverThreshold = 2

	ErrCodeRunnerDisconnected = "RUNNER_DISCONNECTED"
)

// PodCoordinator coordinates pod lifecycle events between backend and runners.
type PodCoordinator struct {
	podRepo           agentpod.PodRepository
	runnerRepo        runnerDomain.RunnerRepository
	autopilotRepo     agentpod.AutopilotRepository
	connectionManager *RunnerConnectionManager
	podRouter         *PodRouter
	heartbeatBatcher  *HeartbeatBatcher
	logger            *slog.Logger

	commandSender        RunnerCommandSender
	relayConnectionCache *RelayConnectionCache

	terminateSentCache map[string]time.Time
	terminateCacheMu   sync.Mutex

	podMissCount map[string]int
	podMissOwner map[string]int64
	podMissMu    sync.Mutex

	initReportCount   map[string]int
	initReportCountMu sync.Mutex

	ackTracker *AckTracker

	onStatusChange   func(podKey string, status string, agentStatus string)
	onInitProgress   func(podKey string, phase string, progress int, message string)

	onAutopilotStatusChange    AutopilotStatusChangeFunc
	onAutopilotIterationChange AutopilotIterationChangeFunc
	onAutopilotThinkingChange  AutopilotThinkingChangeFunc
}

func NewPodCoordinator(
	podRepo agentpod.PodRepository,
	runnerRepo runnerDomain.RunnerRepository,
	cm *RunnerConnectionManager,
	tr *PodRouter,
	hb *HeartbeatBatcher,
	logger *slog.Logger,
) *PodCoordinator {
	pc := &PodCoordinator{
		podRepo:              podRepo,
		runnerRepo:           runnerRepo,
		connectionManager:    cm,
		podRouter:            tr,
		heartbeatBatcher:     hb,
		logger:               logger,
		commandSender:        NewNoOpCommandSender(logger),
		relayConnectionCache: NewRelayConnectionCache(),
		terminateSentCache:   make(map[string]time.Time),
		podMissCount:         make(map[string]int),
		podMissOwner:         make(map[string]int64),
		initReportCount:      make(map[string]int),
		ackTracker:           NewAckTracker(),
	}

	cm.SetHeartbeatCallback(pc.handleHeartbeat)
	cm.SetPodCreatedCallback(pc.handlePodCreated)
	cm.SetPodTerminatedCallback(pc.handlePodTerminated)
	cm.SetAgentStatusCallback(pc.handleAgentStatus)
	cm.SetPodErrorCallback(pc.handlePodError)
	cm.SetDisconnectCallback(pc.handleRunnerDisconnect)
	cm.SetPodInitProgressCallback(pc.handlePodInitProgress)

	cm.SetAutopilotStatusCallback(pc.handleAutopilotControllerStatus)
	cm.SetAutopilotCreatedCallback(pc.handleAutopilotControllerCreated)
	cm.SetAutopilotTerminatedCallback(pc.handleAutopilotControllerTerminated)
	cm.SetAutopilotIterationCallback(pc.handleAutopilotIteration)
	cm.SetAutopilotThinkingCallback(pc.handleAutopilotThinking)

	return pc
}

func (pc *PodCoordinator) SetCommandSender(sender RunnerCommandSender) {
	pc.commandSender = sender
	pc.logger.Info("command sender configured", "type", fmt.Sprintf("%T", sender))
}

func (pc *PodCoordinator) SetAutopilotRepo(repo agentpod.AutopilotRepository) {
	pc.autopilotRepo = repo
}

func (pc *PodCoordinator) GetCommandSender() RunnerCommandSender {
	return pc.commandSender
}

func (pc *PodCoordinator) SetStatusChangeCallback(fn func(podKey string, status string, agentStatus string)) {
	pc.onStatusChange = fn
}

func (pc *PodCoordinator) SetInitProgressCallback(fn func(podKey string, phase string, progress int, message string)) {
	pc.onInitProgress = fn
}

func (pc *PodCoordinator) IncrementPods(ctx context.Context, runnerID int64) error {
	return pc.runnerRepo.IncrementPods(ctx, runnerID)
}

func (pc *PodCoordinator) DecrementPods(ctx context.Context, runnerID int64) error {
	return pc.runnerRepo.DecrementPods(ctx, runnerID)
}

// CreatePod creates a new pod on a runner (uses Proto type directly for zero-copy).
func (pc *PodCoordinator) CreatePod(ctx context.Context, runnerID int64, cmd *runnerv1.CreatePodCommand) error {
	if err := pc.IncrementPods(ctx, runnerID); err != nil {
		return err
	}

	if err := pc.commandSender.SendCreatePod(ctx, runnerID, cmd); err != nil {
		_ = pc.DecrementPods(ctx, runnerID)
		return err
	}
	pc.logger.Info("pod dispatched to runner", "pod_key", cmd.PodKey, "runner_id", runnerID)
	return nil
}

// TerminatePod terminates a pod on a runner.
func (pc *PodCoordinator) TerminatePod(ctx context.Context, podKey string) error {
	pod, err := pc.podRepo.GetByKey(ctx, podKey)
	if err != nil {
		return err
	}

	if err := pc.commandSender.SendTerminatePod(ctx, pod.RunnerID, podKey); err != nil {
		pc.logger.Warn("failed to send terminate to runner, marking as completed",
			"pod_key", podKey, "error", err)
	}

	now := time.Now()
	if _, err := pc.podRepo.UpdateByKey(ctx, podKey, map[string]interface{}{
		"status":      agentpod.StatusCompleted,
		"finished_at": now,
	}); err != nil {
		return err
	}

	pc.podRouter.UnregisterPod(podKey)
	pc.clearMissCount(podKey)

	pc.logger.Info("pod terminate sent", "pod_key", podKey, "runner_id", pod.RunnerID)

	return pc.DecrementPods(ctx, pod.RunnerID)
}

func (pc *PodCoordinator) UpdateActivity(ctx context.Context, podKey string) error {
	return pc.podRepo.UpdateField(ctx, podKey, "last_activity", time.Now())
}

func (pc *PodCoordinator) MarkDisconnected(ctx context.Context, podKey string) error {
	return pc.podRepo.UpdateByKeyAndStatus(ctx, podKey, agentpod.StatusRunning, map[string]interface{}{
		"status": agentpod.StatusDisconnected,
	})
}

func (pc *PodCoordinator) MarkReconnected(ctx context.Context, podKey string) error {
	return pc.podRepo.UpdateByKeyAndStatus(ctx, podKey, agentpod.StatusDisconnected, map[string]interface{}{
		"status":        agentpod.StatusRunning,
		"last_activity": time.Now(),
	})
}

// ==================== AutopilotController Commands ====================

func (pc *PodCoordinator) SendCreateAutopilot(runnerID int64, cmd *runnerv1.CreateAutopilotCommand) error {
	return pc.commandSender.SendCreateAutopilot(runnerID, cmd)
}

func (pc *PodCoordinator) SendAutopilotControl(runnerID int64, cmd *runnerv1.AutopilotControlCommand) error {
	return pc.commandSender.SendAutopilotControl(runnerID, cmd)
}

func (pc *PodCoordinator) GetRelayConnections(runnerID int64) []RelayConnectionInfo {
	return pc.relayConnectionCache.Get(runnerID)
}
