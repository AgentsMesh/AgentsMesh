package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/infra/eventbus"
	"github.com/anthropics/agentsmesh/backend/internal/infra/terminal"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/gorilla/websocket"
)

// TerminalRouter routes terminal data between frontend clients and runners using sharded locks
// Uses 64 shards to minimize lock contention for high-scale deployments (500K+ pods)
type TerminalRouter struct {
	connectionManager *RunnerConnectionManager
	logger            *slog.Logger

	// Command sender for sending terminal input/resize to runners.
	// Must be set via SetCommandSender before use.
	commandSender RunnerCommandSender

	// OSC notification detector
	oscDetector *OSCDetector

	// Sharded storage for all pod-related data
	shards [terminalShards]*terminalShard

	// Buffer size configuration
	scrollbackSize int
}

// NewTerminalRouter creates a new terminal router with sharded locks.
// By default, uses NoOpCommandSender which logs warnings. Call SetCommandSender
// to configure a real command sender (e.g., GRPCCommandSender).
func NewTerminalRouter(cm *RunnerConnectionManager, logger *slog.Logger) *TerminalRouter {
	tr := &TerminalRouter{
		connectionManager: cm,
		logger:            logger,
		scrollbackSize:    DefaultScrollbackSize,
		commandSender:     NewNoOpCommandSender(logger), // Default to no-op
	}

	// Initialize all shards
	for i := 0; i < terminalShards; i++ {
		tr.shards[i] = newTerminalShard()
	}

	// Set up callbacks from connection manager
	cm.SetTerminalOutputCallback(tr.handleTerminalOutput)
	cm.SetPtyResizedCallback(tr.handlePtyResized)

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

// RegisterPod registers a pod's runner mapping
func (tr *TerminalRouter) RegisterPod(podKey string, runnerID int64) {
	tr.RegisterPodWithSize(podKey, runnerID, DefaultTerminalCols, DefaultTerminalRows)
}

// RegisterPodWithSize registers a pod with specific terminal size
func (tr *TerminalRouter) RegisterPodWithSize(podKey string, runnerID int64, cols, rows int) {
	shard := tr.getShard(podKey)

	shard.mu.Lock()
	defer shard.mu.Unlock()

	shard.podRunnerMap[podKey] = runnerID

	// Initialize scrollback buffer
	if _, exists := shard.scrollbackBuffers[podKey]; !exists {
		shard.scrollbackBuffers[podKey] = NewScrollbackBuffer(tr.scrollbackSize)
	}

	// Initialize virtual terminal for agent observation
	if vt, exists := shard.virtualTerminals[podKey]; !exists {
		shard.virtualTerminals[podKey] = terminal.NewVirtualTerminal(cols, rows, DefaultVirtualTerminalHistory)
	} else {
		vt.Resize(cols, rows)
	}

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
	delete(shard.scrollbackBuffers, podKey)
	delete(shard.virtualTerminals, podKey)
	delete(shard.ptySize, podKey)
	clients := shard.terminalClients[podKey]
	delete(shard.terminalClients, podKey)
	shard.mu.Unlock()

	// Close client connections outside the lock
	for client := range clients {
		close(client.Send)
		client.Conn.Close()
	}

	tr.logger.Debug("pod unregistered", "pod_key", podKey)
}

// ConnectClient connects a frontend client to a pod
func (tr *TerminalRouter) ConnectClient(podKey string, conn *websocket.Conn) (*TerminalClient, error) {
	client := &TerminalClient{
		Conn:   conn,
		PodKey: podKey,
		Send:   make(chan TerminalMessage, 256),
	}

	shard := tr.getShard(podKey)

	shard.mu.Lock()
	if shard.terminalClients[podKey] == nil {
		shard.terminalClients[podKey] = make(map[*TerminalClient]bool)
	}
	shard.terminalClients[podKey][client] = true

	// Get current PTY size while holding lock
	currentSize := shard.ptySize[podKey]
	shard.mu.Unlock()

	tr.logger.Info("terminal client connected", "pod_key", podKey)

	// Send current PTY size to the newly connected client
	if currentSize != nil {
		tr.sendPtyResizedToClient(client, currentSize.Cols, currentSize.Rows)
	}

	// Note: We intentionally do NOT send scrollback/history data from backend.
	// The frontend uses xterm-addon-serialize to save/restore terminal state locally.
	// This avoids issues with raw scrollback (duplicate display from readline sequences)
	// and processed VirtualTerminal output (layout issues without ANSI positioning).

	return client, nil
}

// DisconnectClient disconnects a frontend client
func (tr *TerminalRouter) DisconnectClient(client *TerminalClient) {
	shard := tr.getShard(client.PodKey)

	shard.mu.Lock()
	if clients, ok := shard.terminalClients[client.PodKey]; ok {
		delete(clients, client)
		if len(clients) == 0 {
			delete(shard.terminalClients, client.PodKey)
		}
	}
	shard.mu.Unlock()

	close(client.Send)
	tr.logger.Info("terminal client disconnected", "pod_key", client.PodKey)
}

// handleTerminalOutput handles terminal output from a runner (Proto type)
func (tr *TerminalRouter) handleTerminalOutput(runnerID int64, data *runnerv1.TerminalOutputEvent) {
	podKey := data.PodKey
	shard := tr.getShard(podKey)

	// Get buffer and virtual terminal under read lock
	shard.mu.RLock()
	buffer := shard.scrollbackBuffers[podKey]
	vt := shard.virtualTerminals[podKey]
	clients := shard.terminalClients[podKey]
	shard.mu.RUnlock()

	// Store in scrollback buffer (raw data for frontend)
	if buffer != nil {
		buffer.Write(data.Data)
	}

	// Feed to virtual terminal (processed data for agent observation)
	if vt != nil {
		vt.Feed(data.Data)
	}

	// Check for OSC 777/9 notifications and publish events
	if tr.oscDetector != nil {
		tr.oscDetector.DetectAndPublish(context.Background(), podKey, data.Data)
		// Check for OSC 0/2 title changes and publish events
		tr.oscDetector.DetectAndPublishTitle(context.Background(), podKey, data.Data)
	}

	// Route to all connected clients
	if len(clients) == 0 {
		tr.logger.Debug("no clients for terminal output", "pod_key", podKey)
		return
	}

	// Broadcast to all clients
	var deadClients []*TerminalClient
	for client := range clients {
		select {
		case client.Send <- TerminalMessage{Data: data.Data, IsJSON: false}:
		default:
			// Client buffer full, mark for removal
			deadClients = append(deadClients, client)
		}
	}

	// Clean up dead clients
	if len(deadClients) > 0 {
		shard.mu.Lock()
		for _, client := range deadClients {
			delete(shard.terminalClients[podKey], client)
		}
		shard.mu.Unlock()
	}
}

// handlePtyResized handles PTY resize notifications from runner (Proto type)
func (tr *TerminalRouter) handlePtyResized(runnerID int64, data *runnerv1.PtyResizedEvent) {
	podKey := data.PodKey
	shard := tr.getShard(podKey)

	cols := int(data.Cols)
	rows := int(data.Rows)

	shard.mu.Lock()
	// Update local PTY size record
	shard.ptySize[podKey] = &PtySize{Cols: cols, Rows: rows}

	// Update virtual terminal size
	if vt, exists := shard.virtualTerminals[podKey]; exists {
		vt.Resize(cols, rows)
		tr.logger.Debug("virtual terminal resized",
			"pod_key", podKey,
			"cols", cols,
			"rows", rows)
	}

	// Get clients while holding lock
	clients := shard.terminalClients[podKey]
	shard.mu.Unlock()

	// Broadcast pty_resized to all connected frontend clients
	for client := range clients {
		tr.sendPtyResizedToClient(client, cols, rows)
	}
}

// sendPtyResizedToClient sends pty_resized message to a single client
func (tr *TerminalRouter) sendPtyResizedToClient(client *TerminalClient, cols, rows int) {
	msg, err := json.Marshal(map[string]interface{}{
		"type": "pty_resized",
		"cols": cols,
		"rows": rows,
	})
	if err != nil {
		tr.logger.Error("failed to marshal pty_resized message", "error", err)
		return
	}

	select {
	case client.Send <- TerminalMessage{Data: msg, IsJSON: true}:
		tr.logger.Debug("sent pty_resized to client",
			"pod_key", client.PodKey,
			"cols", cols,
			"rows", rows)
	default:
		tr.logger.Warn("failed to send pty_resized, channel full", "pod_key", client.PodKey)
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

// GetClientCount returns the number of clients connected to a pod
func (tr *TerminalRouter) GetClientCount(podKey string) int {
	shard := tr.getShard(podKey)

	shard.mu.RLock()
	defer shard.mu.RUnlock()
	return len(shard.terminalClients[podKey])
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

