package runner

import (
	"context"
	"fmt"
	"hash/fnv"
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/infra/eventbus"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// TerminalRouter routes terminal commands between frontend and runners using sharded locks.
// After Relay migration, this component only handles:
// - Pod-Runner mapping for routing commands
// - PTY size tracking for resize events
// - OSC notification detection
//
// Terminal data streaming is handled by Relay (Browser <-> Relay <-> Runner).
type TerminalRouter struct {
	connectionManager *RunnerConnectionManager
	logger            *slog.Logger

	// Command sender for sending terminal input/resize to runners.
	// Must be set via SetCommandSender before use.
	commandSender RunnerCommandSender

	// OSC notification detector
	oscDetector *OSCDetector

	// Sharded storage for pod-related data
	shards [terminalShards]*terminalShard
}

// NewTerminalRouter creates a new terminal router with sharded locks.
// By default, uses NoOpCommandSender which logs warnings. Call SetCommandSender
// to configure a real command sender (e.g., GRPCCommandSender).
func NewTerminalRouter(cm *RunnerConnectionManager, logger *slog.Logger) *TerminalRouter {
	tr := &TerminalRouter{
		connectionManager: cm,
		logger:            logger,
		commandSender:     NewNoOpCommandSender(logger), // Default to no-op
	}

	// Initialize all shards
	for i := 0; i < terminalShards; i++ {
		tr.shards[i] = newTerminalShard()
	}

	// Set up callbacks from connection manager
	cm.SetPtyResizedCallback(tr.handlePtyResized)

	// Set up OSC notification callbacks
	// OSC events are sent directly by Runner (bypassing terminal output throttling)
	cm.SetOSCNotificationCallback(tr.handleOSCNotification)
	cm.SetOSCTitleCallback(tr.handleOSCTitle)

	return tr
}

// getShard returns the shard for a given pod key using FNV-1a hashing
func (tr *TerminalRouter) getShard(podKey string) *terminalShard {
	h := fnv.New32a()
	h.Write([]byte(podKey))
	return tr.shards[h.Sum32()%terminalShards]
}

// SetCommandSender sets the command sender for sending terminal input/resize to runners.
// This should be called to configure a real command sender (e.g., GRPCCommandSender).
func (tr *TerminalRouter) SetCommandSender(sender RunnerCommandSender) {
	tr.commandSender = sender
	tr.logger.Info("command sender configured", "type", fmt.Sprintf("%T", sender))
}

// SetEventBus sets the event bus for publishing terminal notifications
func (tr *TerminalRouter) SetEventBus(eb *eventbus.EventBus) {
	if tr.oscDetector == nil {
		tr.oscDetector = &OSCDetector{}
	}
	tr.oscDetector.eventBus = eb
}

// SetPodInfoGetter sets the pod info getter for retrieving pod organization and creator
func (tr *TerminalRouter) SetPodInfoGetter(getter PodInfoGetter) {
	if tr.oscDetector == nil {
		tr.oscDetector = &OSCDetector{}
	}
	tr.oscDetector.podInfoGetter = getter
}

// RegisterPod registers a pod's runner mapping with default terminal size.
func (tr *TerminalRouter) RegisterPod(podKey string, runnerID int64) {
	tr.RegisterPodWithSize(podKey, runnerID, DefaultTerminalCols, DefaultTerminalRows)
}

// EnsurePodRegistered ensures the pod is registered with the terminal router.
// Use this for heartbeat re-registration to preserve existing state.
func (tr *TerminalRouter) EnsurePodRegistered(podKey string, runnerID int64) {
	shard := tr.getShard(podKey)

	shard.mu.Lock()
	defer shard.mu.Unlock()

	shard.podRunnerMap[podKey] = runnerID
	tr.logger.Debug("pod registered (ensure)", "pod_key", podKey, "runner_id", runnerID)
}

// RegisterPodWithSize registers a pod with specific terminal size
func (tr *TerminalRouter) RegisterPodWithSize(podKey string, runnerID int64, cols, rows int) {
	shard := tr.getShard(podKey)

	shard.mu.Lock()
	defer shard.mu.Unlock()

	shard.podRunnerMap[podKey] = runnerID
	shard.ptySize[podKey] = &PtySize{Cols: cols, Rows: rows}

	tr.logger.Debug("pod registered",
		"pod_key", podKey,
		"runner_id", runnerID,
		"cols", cols,
		"rows", rows)
}

// UnregisterPod unregisters a pod
func (tr *TerminalRouter) UnregisterPod(podKey string) {
	shard := tr.getShard(podKey)

	shard.mu.Lock()
	delete(shard.podRunnerMap, podKey)
	delete(shard.ptySize, podKey)
	shard.mu.Unlock()

	tr.logger.Debug("pod unregistered", "pod_key", podKey)
}

// handlePtyResized handles PTY resize notifications from runner
func (tr *TerminalRouter) handlePtyResized(runnerID int64, data *runnerv1.PtyResizedEvent) {
	podKey := data.PodKey
	shard := tr.getShard(podKey)

	cols := int(data.Cols)
	rows := int(data.Rows)

	shard.mu.Lock()
	// Update local PTY size record
	shard.ptySize[podKey] = &PtySize{Cols: cols, Rows: rows}

	// Ensure pod is registered
	if _, exists := shard.podRunnerMap[podKey]; !exists {
		shard.podRunnerMap[podKey] = runnerID
		tr.logger.Info("pod auto-registered on resize",
			"pod_key", podKey,
			"runner_id", runnerID)
	}
	shard.mu.Unlock()

	tr.logger.Debug("pty resized",
		"pod_key", podKey,
		"cols", cols,
		"rows", rows)
}

// handleOSCNotification handles OSC notification events directly from Runner.
// OSC data flow (high priority, bypasses terminal output):
//
//	PTY → VT.Feed() → OSCHandler → gRPC controlCh → Backend → EventBus → WebSocket → Browser
//
// OSC events are sent as discrete messages via controlCh, ensuring they are
// never lost due to SmartAggregator throttling (terminal output uses separate Relay path).
func (tr *TerminalRouter) handleOSCNotification(runnerID int64, data *runnerv1.OSCNotificationEvent) {
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
func (tr *TerminalRouter) handleOSCTitle(runnerID int64, data *runnerv1.OSCTitleEvent) {
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

// RouteInput routes terminal input from frontend to runner
func (tr *TerminalRouter) RouteInput(podKey string, data []byte) error {
	shard := tr.getShard(podKey)

	shard.mu.RLock()
	runnerID, ok := shard.podRunnerMap[podKey]
	shard.mu.RUnlock()

	if !ok {
		tr.logger.Warn("no runner for pod", "pod_key", podKey)
		return ErrRunnerNotConnected
	}

	return tr.commandSender.SendTerminalInput(context.Background(), runnerID, podKey, data)
}

// RouteResize routes terminal resize from frontend to runner
func (tr *TerminalRouter) RouteResize(podKey string, cols, rows int) error {
	shard := tr.getShard(podKey)

	shard.mu.RLock()
	runnerID, ok := shard.podRunnerMap[podKey]
	shard.mu.RUnlock()

	if !ok {
		tr.logger.Warn("no runner for pod", "pod_key", podKey)
		return ErrRunnerNotConnected
	}

	return tr.commandSender.SendTerminalResize(context.Background(), runnerID, podKey, cols, rows)
}

// IsPodRegistered checks if a pod is registered
func (tr *TerminalRouter) IsPodRegistered(podKey string) bool {
	shard := tr.getShard(podKey)

	shard.mu.RLock()
	defer shard.mu.RUnlock()
	_, ok := shard.podRunnerMap[podKey]
	return ok
}

// GetRunnerID returns the runner ID for a pod
func (tr *TerminalRouter) GetRunnerID(podKey string) (int64, bool) {
	shard := tr.getShard(podKey)

	shard.mu.RLock()
	defer shard.mu.RUnlock()
	id, ok := shard.podRunnerMap[podKey]
	return id, ok
}

// GetRegisteredPodCount returns the total number of registered pods across all shards
func (tr *TerminalRouter) GetRegisteredPodCount() int {
	total := 0
	for i := 0; i < terminalShards; i++ {
		shard := tr.shards[i]
		shard.mu.RLock()
		total += len(shard.podRunnerMap)
		shard.mu.RUnlock()
	}
	return total
}

// GetRecentOutput returns recent terminal output for a pod.
// After Relay migration, terminal output is streamed directly via Relay,
// so this returns empty. Retained for TerminalRouterInterface compatibility.
func (tr *TerminalRouter) GetRecentOutput(podKey string, lines int) []byte {
	return nil
}

// GetScreenSnapshot returns a text snapshot of the terminal screen.
// After Relay migration, this is not available server-side.
func (tr *TerminalRouter) GetScreenSnapshot(podKey string) string {
	return ""
}

// GetCursorPosition returns the cursor position in the terminal.
// After Relay migration, this is not available server-side.
func (tr *TerminalRouter) GetCursorPosition(podKey string) (row, col int) {
	return 0, 0
}

// GetPtySize returns the PTY size for a pod
func (tr *TerminalRouter) GetPtySize(podKey string) (cols, rows int, ok bool) {
	shard := tr.getShard(podKey)

	shard.mu.RLock()
	defer shard.mu.RUnlock()

	if size, exists := shard.ptySize[podKey]; exists {
		return size.Cols, size.Rows, true
	}
	return DefaultTerminalCols, DefaultTerminalRows, false
}
