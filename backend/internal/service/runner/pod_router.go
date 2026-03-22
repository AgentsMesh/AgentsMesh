package runner

import (
	"context"
	"fmt"
	"hash/fnv"
	"log/slog"
	"sync"

	"github.com/anthropics/agentsmesh/backend/internal/infra/eventbus"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// PodRouter routes pod commands between frontend and runners using sharded locks.
// After Relay migration, this component only handles:
// - Pod-Runner mapping for routing commands
// - OSC notification detection
//
// Terminal data streaming and resize are handled by Relay (Browser <-> Relay <-> Runner).
type PodRouter struct {
	connectionManager *RunnerConnectionManager
	logger            *slog.Logger

	// Command sender for sending pod commands to runners.
	// Must be set via SetCommandSender before use.
	commandSender RunnerCommandSender

	// OSC notification detector
	oscDetector *OSCDetector

	// Sharded storage for pod-related data
	shards [routerShards]*routerShard

	// Pending pod observation queries (shared with pod_router_query.go)
	pendingQueries sync.Map      // map[requestID]*pendingPodQuery
	done           chan struct{} // signal channel for graceful shutdown of cleanup goroutine
}

// NewPodRouter creates a new pod router with sharded locks.
// By default, uses NoOpCommandSender which logs warnings. Call SetCommandSender
// to configure a real command sender (e.g., GRPCCommandSender).
func NewPodRouter(cm *RunnerConnectionManager, logger *slog.Logger) *PodRouter {
	tr := &PodRouter{
		connectionManager: cm,
		logger:            logger,
		commandSender:     NewNoOpCommandSender(logger), // Default to no-op
	}

	// Initialize all shards
	for i := 0; i < routerShards; i++ {
		tr.shards[i] = newRouterShard()
	}

	// Set up OSC notification callbacks
	// OSC events are sent directly by Runner (bypassing terminal output throttling)
	cm.SetOSCNotificationCallback(tr.handleOSCNotification)
	cm.SetOSCTitleCallback(tr.handleOSCTitle)

	// Initialize pod observation query support (callback + cleanup goroutine)
	initQuerySupport(tr, cm, make(chan struct{}))

	return tr
}

// getShard returns the shard for a given pod key using FNV-1a hashing
func (tr *PodRouter) getShard(podKey string) *routerShard {
	h := fnv.New32a()
	h.Write([]byte(podKey))
	return tr.shards[h.Sum32()%routerShards]
}

// SetCommandSender sets the command sender for sending pod input/resize to runners.
// This should be called to configure a real command sender (e.g., GRPCCommandSender).
func (tr *PodRouter) SetCommandSender(sender RunnerCommandSender) {
	tr.commandSender = sender
	tr.logger.Info("command sender configured", "type", fmt.Sprintf("%T", sender))
}

// SetEventBus sets the event bus for publishing pod notifications
func (tr *PodRouter) SetEventBus(eb *eventbus.EventBus) {
	if tr.oscDetector == nil {
		tr.oscDetector = &OSCDetector{}
	}
	tr.oscDetector.eventBus = eb
}

// SetPodInfoGetter sets the pod info getter for retrieving pod organization and creator
func (tr *PodRouter) SetPodInfoGetter(getter PodInfoGetter) {
	if tr.oscDetector == nil {
		tr.oscDetector = &OSCDetector{}
	}
	tr.oscDetector.podInfoGetter = getter
}

// SetNotifyFunc sets the notification callback for OSC notifications.
// When set, OSC notifications are routed through the NotificationDispatcher
// for preference-aware delivery instead of direct EventBus publish.
func (tr *PodRouter) SetNotifyFunc(fn NotifyFunc) {
	if tr.oscDetector == nil {
		tr.oscDetector = &OSCDetector{}
	}
	tr.oscDetector.notifyFunc = fn
}

// RegisterPod registers a pod's runner mapping.
func (tr *PodRouter) RegisterPod(podKey string, runnerID int64) {
	shard := tr.getShard(podKey)

	shard.mu.Lock()
	defer shard.mu.Unlock()

	shard.podRunnerMap[podKey] = runnerID

	tr.logger.Debug("pod registered",
		"pod_key", podKey,
		"runner_id", runnerID)
}

// EnsurePodRegistered ensures the pod is registered with the pod router.
// Use this for heartbeat re-registration to preserve existing state.
func (tr *PodRouter) EnsurePodRegistered(podKey string, runnerID int64) {
	shard := tr.getShard(podKey)

	shard.mu.Lock()
	defer shard.mu.Unlock()

	shard.podRunnerMap[podKey] = runnerID
	tr.logger.Debug("pod registered (ensure)", "pod_key", podKey, "runner_id", runnerID)
}

// UnregisterPod unregisters a pod
func (tr *PodRouter) UnregisterPod(podKey string) {
	shard := tr.getShard(podKey)

	shard.mu.Lock()
	delete(shard.podRunnerMap, podKey)
	shard.mu.Unlock()

	tr.logger.Debug("pod unregistered", "pod_key", podKey)
}

// handleOSCNotification handles OSC notification events directly from Runner.
// OSC data flow (high priority, bypasses terminal output):
//
//	PTY → VT.Feed() → OSCHandler → gRPC controlCh → Backend → EventBus → WebSocket → Browser
//
// OSC events are sent as discrete messages via controlCh, ensuring they are
// never lost due to SmartAggregator throttling (terminal output uses separate Relay path).
func (tr *PodRouter) handleOSCNotification(runnerID int64, data *runnerv1.OSCNotificationEvent) {
	podKey := data.PodKey

	tr.logger.Info("received OSC notification",
		"pod_key", podKey,
		"runner_id", runnerID,
		"title", data.Title,
		"body", data.Body,
	)

	// Publish to EventBus using OSCDetector
	if tr.oscDetector != nil && tr.oscDetector.eventBus != nil {
		tr.oscDetector.PublishNotification(context.Background(), podKey, data.Title, data.Body)
	}
}

// handleOSCTitle handles OSC title change events directly from Runner.
// This is the new OSC data flow that bypasses terminal output throttling.
func (tr *PodRouter) handleOSCTitle(runnerID int64, data *runnerv1.OSCTitleEvent) {
	podKey := data.PodKey

	tr.logger.Debug("received OSC title change",
		"pod_key", podKey,
		"runner_id", runnerID,
		"title", data.Title,
	)

	// Publish to EventBus using OSCDetector
	if tr.oscDetector != nil && tr.oscDetector.eventBus != nil {
		tr.oscDetector.PublishTitle(context.Background(), podKey, data.Title)
	}
}

// RoutePodInput routes pod input from frontend to runner
func (tr *PodRouter) RoutePodInput(podKey string, data []byte) error {
	shard := tr.getShard(podKey)

	shard.mu.RLock()
	runnerID, ok := shard.podRunnerMap[podKey]
	shard.mu.RUnlock()

	if !ok {
		tr.logger.Warn("no runner for pod", "pod_key", podKey)
		return ErrRunnerNotConnected
	}

	return tr.commandSender.SendPodInput(context.Background(), runnerID, podKey, data)
}

// RoutePrompt routes a prompt to a pod via SendPrompt (mode-agnostic: PTY or ACP).
// For PTY pods, the runner writes the prompt to stdin.
// For ACP pods, the runner sends the prompt via the ACP protocol.
func (tr *PodRouter) RoutePrompt(podKey string, prompt string) error {
	shard := tr.getShard(podKey)

	shard.mu.RLock()
	runnerID, ok := shard.podRunnerMap[podKey]
	shard.mu.RUnlock()

	if !ok {
		tr.logger.Warn("no runner for pod", "pod_key", podKey)
		return ErrRunnerNotConnected
	}

	return tr.commandSender.SendPrompt(context.Background(), runnerID, podKey, prompt)
}

// IsPodRegistered checks if a pod is registered
func (tr *PodRouter) IsPodRegistered(podKey string) bool {
	shard := tr.getShard(podKey)

	shard.mu.RLock()
	defer shard.mu.RUnlock()
	_, ok := shard.podRunnerMap[podKey]
	return ok
}

// GetRunnerID returns the runner ID for a pod
func (tr *PodRouter) GetRunnerID(podKey string) (int64, bool) {
	shard := tr.getShard(podKey)

	shard.mu.RLock()
	defer shard.mu.RUnlock()
	id, ok := shard.podRunnerMap[podKey]
	return id, ok
}

// GetRegisteredPodCount returns the total number of registered pods across all shards
func (tr *PodRouter) GetRegisteredPodCount() int {
	total := 0
	for i := 0; i < routerShards; i++ {
		shard := tr.shards[i]
		shard.mu.RLock()
		total += len(shard.podRunnerMap)
		shard.mu.RUnlock()
	}
	return total
}

// Stop gracefully stops the PodRouter's background goroutines (query cleanup).
func (tr *PodRouter) Stop() {
	close(tr.done)
}
