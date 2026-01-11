package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"regexp"
	"sync"
	"unicode/utf8"

	"github.com/anthropics/agentmesh/backend/internal/infra/eventbus"
	"github.com/anthropics/agentmesh/backend/internal/infra/terminal"
	"github.com/gorilla/websocket"
)

const (
	// Default scrollback buffer size (100KB)
	DefaultScrollbackSize = 100 * 1024
)

// TerminalMessage represents a message to send to the frontend client
type TerminalMessage struct {
	Data   []byte
	IsJSON bool // true for JSON control messages, false for binary terminal output
}

// TerminalClient represents a frontend WebSocket client connected to a terminal
type TerminalClient struct {
	Conn   *websocket.Conn
	PodKey string
	Send   chan TerminalMessage
}

// PtySize represents the current PTY terminal size
type PtySize struct {
	Cols int
	Rows int
}

// OSC 777 notification pattern: ESC ] 777 ; notify ; <title> ; <body> BEL
// Also matches: ESC ] 777 ; <title> ; <body> BEL (without "notify" keyword)
var osc777Pattern = regexp.MustCompile(`\x1b\]777;(?:notify;)?([^;]*);([^\x07]*)\x07`)

// OSC 9 notification pattern: ESC ] 9 ; <message> BEL (iTerm2/Windows Terminal style)
// Used by Claude Code for notifications
var osc9Pattern = regexp.MustCompile(`\x1b\]9;([^\x07]*)\x07`)

// PodInfoGetter interface for getting pod information
type PodInfoGetter interface {
	GetPodOrganizationAndCreator(ctx context.Context, podKey string) (orgID, creatorID int64, err error)
}

// TerminalRouter routes terminal data between frontend clients and runners
type TerminalRouter struct {
	connectionManager *ConnectionManager
	logger            *slog.Logger

	// EventBus for publishing notifications
	eventBus      *eventbus.EventBus
	podInfoGetter PodInfoGetter

	// Pod -> Runner mapping
	podRunnerMap map[string]int64
	podRunnerMu  sync.RWMutex

	// Pod -> Frontend clients
	terminalClients   map[string]map[*TerminalClient]bool
	terminalClientsMu sync.RWMutex

	// Scrollback buffers for reconnection (raw output for frontend)
	scrollbackBuffers map[string]*ScrollbackBuffer
	scrollbackMu      sync.RWMutex

	// Virtual terminals for agent observation (processed output)
	virtualTerminals map[string]*terminal.VirtualTerminal
	virtualTermMu    sync.RWMutex

	// Current PTY size for each pod (for broadcasting to clients)
	ptySize   map[string]*PtySize
	ptySizeMu sync.RWMutex

	// Buffer size configuration
	scrollbackSize int
}

// ScrollbackBuffer stores terminal output for reconnection
type ScrollbackBuffer struct {
	data     []byte
	maxSize  int
	mu       sync.RWMutex
}

// NewScrollbackBuffer creates a new scrollback buffer
func NewScrollbackBuffer(maxSize int) *ScrollbackBuffer {
	return &ScrollbackBuffer{
		data:    make([]byte, 0, maxSize),
		maxSize: maxSize,
	}
}

// Write appends data to the buffer, trimming old data if necessary
func (sb *ScrollbackBuffer) Write(data []byte) {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	sb.data = append(sb.data, data...)

	// Trim if exceeded max size
	if len(sb.data) > sb.maxSize {
		// Keep only the last maxSize bytes
		sb.data = sb.data[len(sb.data)-sb.maxSize:]
		// Ensure we start at a valid UTF-8 boundary
		sb.data = trimToValidUTF8Start(sb.data)
	}
}

// trimToValidUTF8Start ensures data starts with a valid UTF-8 character.
// If the data begins with continuation bytes (10xxxxxx pattern), it skips them
// to find the start of a valid UTF-8 sequence.
func trimToValidUTF8Start(data []byte) []byte {
	if len(data) == 0 {
		return data
	}

	// Check up to utf8.UTFMax (4) bytes for a valid start
	for i := 0; i < len(data) && i < utf8.UTFMax; i++ {
		// Check if remaining data is valid UTF-8
		if utf8.Valid(data[i:]) {
			return data[i:]
		}
		// Also check if this byte starts a valid UTF-8 sequence
		// (not a continuation byte: 10xxxxxx)
		if data[i]&0xC0 != 0x80 {
			// This is a leading byte, check if the sequence starting here is valid
			if r, _ := utf8.DecodeRune(data[i:]); r != utf8.RuneError {
				return data[i:]
			}
		}
	}

	// Fallback: return original data (shouldn't normally reach here)
	return data
}

// GetData returns a copy of the buffer data
func (sb *ScrollbackBuffer) GetData() []byte {
	sb.mu.RLock()
	defer sb.mu.RUnlock()

	result := make([]byte, len(sb.data))
	copy(result, sb.data)
	return result
}

// GetRecentLines returns the last N lines from the buffer
func (sb *ScrollbackBuffer) GetRecentLines(lines int) []byte {
	sb.mu.RLock()
	defer sb.mu.RUnlock()

	if len(sb.data) == 0 {
		return nil
	}

	// Split by newlines and return last N lines
	allLines := bytes.Split(sb.data, []byte("\n"))
	if len(allLines) <= lines {
		return sb.data
	}

	recentLines := allLines[len(allLines)-lines:]
	return bytes.Join(recentLines, []byte("\n"))
}

// Clear clears the buffer
func (sb *ScrollbackBuffer) Clear() {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	sb.data = sb.data[:0]
}

// NewTerminalRouter creates a new terminal router
func NewTerminalRouter(cm *ConnectionManager, logger *slog.Logger) *TerminalRouter {
	tr := &TerminalRouter{
		connectionManager: cm,
		logger:            logger,
		podRunnerMap:      make(map[string]int64),
		terminalClients:   make(map[string]map[*TerminalClient]bool),
		scrollbackBuffers: make(map[string]*ScrollbackBuffer),
		virtualTerminals:  make(map[string]*terminal.VirtualTerminal),
		ptySize:           make(map[string]*PtySize),
		scrollbackSize:    DefaultScrollbackSize,
	}

	// Set up callbacks from connection manager
	cm.SetTerminalOutputCallback(tr.handleTerminalOutput)
	cm.SetPtyResizedCallback(tr.handlePtyResized)

	return tr
}

// SetEventBus sets the event bus for publishing terminal notifications
func (tr *TerminalRouter) SetEventBus(eb *eventbus.EventBus) {
	tr.eventBus = eb
}

// SetPodInfoGetter sets the pod info getter for retrieving pod organization and creator
func (tr *TerminalRouter) SetPodInfoGetter(getter PodInfoGetter) {
	tr.podInfoGetter = getter
}

// DefaultTerminalCols is the default terminal width
const DefaultTerminalCols = 80

// DefaultTerminalRows is the default terminal height
const DefaultTerminalRows = 24

// DefaultVirtualTerminalHistory is the default scrollback history lines
const DefaultVirtualTerminalHistory = 10000

// RegisterPod registers a pod's runner mapping
func (tr *TerminalRouter) RegisterPod(podKey string, runnerID int64) {
	tr.RegisterPodWithSize(podKey, runnerID, DefaultTerminalCols, DefaultTerminalRows)
}

// RegisterPodWithSize registers a pod with specific terminal size
func (tr *TerminalRouter) RegisterPodWithSize(podKey string, runnerID int64, cols, rows int) {
	tr.podRunnerMu.Lock()
	tr.podRunnerMap[podKey] = runnerID
	tr.podRunnerMu.Unlock()

	// Initialize scrollback buffer
	tr.scrollbackMu.Lock()
	if _, exists := tr.scrollbackBuffers[podKey]; !exists {
		tr.scrollbackBuffers[podKey] = NewScrollbackBuffer(tr.scrollbackSize)
	}
	tr.scrollbackMu.Unlock()

	// Initialize virtual terminal for agent observation
	tr.virtualTermMu.Lock()
	if vt, exists := tr.virtualTerminals[podKey]; !exists {
		tr.virtualTerminals[podKey] = terminal.NewVirtualTerminal(cols, rows, DefaultVirtualTerminalHistory)
	} else {
		vt.Resize(cols, rows)
	}
	tr.virtualTermMu.Unlock()

	tr.logger.Debug("pod registered",
		"pod_key", podKey,
		"runner_id", runnerID,
		"cols", cols,
		"rows", rows)
}

// UnregisterPod unregisters a pod
func (tr *TerminalRouter) UnregisterPod(podKey string) {
	tr.podRunnerMu.Lock()
	delete(tr.podRunnerMap, podKey)
	tr.podRunnerMu.Unlock()

	// Clean up scrollback buffer
	tr.scrollbackMu.Lock()
	delete(tr.scrollbackBuffers, podKey)
	tr.scrollbackMu.Unlock()

	// Clean up virtual terminal
	tr.virtualTermMu.Lock()
	delete(tr.virtualTerminals, podKey)
	tr.virtualTermMu.Unlock()

	// Clean up PTY size record
	tr.ptySizeMu.Lock()
	delete(tr.ptySize, podKey)
	tr.ptySizeMu.Unlock()

	// Disconnect all clients
	tr.terminalClientsMu.Lock()
	clients := tr.terminalClients[podKey]
	delete(tr.terminalClients, podKey)
	tr.terminalClientsMu.Unlock()

	// Close client connections
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

	tr.terminalClientsMu.Lock()
	if tr.terminalClients[podKey] == nil {
		tr.terminalClients[podKey] = make(map[*TerminalClient]bool)
	}
	tr.terminalClients[podKey][client] = true
	tr.terminalClientsMu.Unlock()

	tr.logger.Info("terminal client connected", "pod_key", podKey)

	// Send current PTY size to the newly connected client
	tr.ptySizeMu.RLock()
	currentSize := tr.ptySize[podKey]
	tr.ptySizeMu.RUnlock()

	if currentSize != nil {
		tr.sendPtyResizedToClient(client, currentSize.Cols, currentSize.Rows)
	}

	// Send scrollback data to the newly connected client
	tr.scrollbackMu.RLock()
	buffer := tr.scrollbackBuffers[podKey]
	tr.scrollbackMu.RUnlock()

	if buffer != nil {
		data := buffer.GetData()
		if len(data) > 0 {
			select {
			case client.Send <- TerminalMessage{Data: data, IsJSON: false}:
				tr.logger.Debug("sent scrollback to client",
					"pod_key", podKey,
					"size", len(data))
			default:
				// Channel full, skip scrollback
			}
		}
	}

	return client, nil
}

// DisconnectClient disconnects a frontend client
func (tr *TerminalRouter) DisconnectClient(client *TerminalClient) {
	tr.terminalClientsMu.Lock()
	if clients, ok := tr.terminalClients[client.PodKey]; ok {
		delete(clients, client)
		if len(clients) == 0 {
			delete(tr.terminalClients, client.PodKey)
		}
	}
	tr.terminalClientsMu.Unlock()

	close(client.Send)
	tr.logger.Info("terminal client disconnected", "pod_key", client.PodKey)
}

// handleTerminalOutput handles terminal output from a runner
func (tr *TerminalRouter) handleTerminalOutput(runnerID int64, data *TerminalOutputData) {
	podKey := data.PodKey

	// Store in scrollback buffer (raw data for frontend)
	tr.scrollbackMu.RLock()
	buffer := tr.scrollbackBuffers[podKey]
	tr.scrollbackMu.RUnlock()

	if buffer != nil {
		buffer.Write(data.Data)
	}

	// Feed to virtual terminal (processed data for agent observation)
	tr.virtualTermMu.RLock()
	vt := tr.virtualTerminals[podKey]
	tr.virtualTermMu.RUnlock()

	if vt != nil {
		vt.Feed(data.Data)
	}

	// Check for OSC 777 notifications and publish events
	tr.detectAndPublishOSC777(podKey, data.Data)

	// Route to all connected clients
	tr.terminalClientsMu.RLock()
	clients := tr.terminalClients[podKey]
	tr.terminalClientsMu.RUnlock()

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
		tr.terminalClientsMu.Lock()
		for _, client := range deadClients {
			delete(tr.terminalClients[podKey], client)
		}
		tr.terminalClientsMu.Unlock()
	}
}

// handlePtyResized handles PTY resize notifications from runner
func (tr *TerminalRouter) handlePtyResized(runnerID int64, data *PtyResizedData) {
	podKey := data.PodKey

	// Update local PTY size record
	tr.ptySizeMu.Lock()
	tr.ptySize[podKey] = &PtySize{Cols: data.Cols, Rows: data.Rows}
	tr.ptySizeMu.Unlock()

	// Update virtual terminal size
	tr.virtualTermMu.Lock()
	if vt, exists := tr.virtualTerminals[podKey]; exists {
		vt.Resize(data.Cols, data.Rows)
		tr.logger.Debug("virtual terminal resized",
			"pod_key", podKey,
			"cols", data.Cols,
			"rows", data.Rows)
	}
	tr.virtualTermMu.Unlock()

	// Broadcast pty_resized to all connected frontend clients
	tr.broadcastPtyResized(podKey, data.Cols, data.Rows)
}

// broadcastPtyResized sends pty_resized message to all connected clients for a pod
func (tr *TerminalRouter) broadcastPtyResized(podKey string, cols, rows int) {
	tr.terminalClientsMu.RLock()
	clients := tr.terminalClients[podKey]
	tr.terminalClientsMu.RUnlock()

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
	tr.podRunnerMu.RLock()
	runnerID, ok := tr.podRunnerMap[podKey]
	tr.podRunnerMu.RUnlock()

	if !ok {
		tr.logger.Warn("no runner for pod", "pod_key", podKey)
		return ErrRunnerNotConnected
	}

	return tr.connectionManager.SendTerminalInput(nil, runnerID, podKey, data)
}

// RouteResize routes terminal resize from frontend to runner
func (tr *TerminalRouter) RouteResize(podKey string, cols, rows int) error {
	tr.podRunnerMu.RLock()
	runnerID, ok := tr.podRunnerMap[podKey]
	tr.podRunnerMu.RUnlock()

	if !ok {
		tr.logger.Warn("no runner for pod", "pod_key", podKey)
		return ErrRunnerNotConnected
	}

	return tr.connectionManager.SendTerminalResize(nil, runnerID, podKey, cols, rows)
}

// GetRecentOutput returns recent terminal output for observation
// If raw is true, returns raw scrollback data; otherwise returns processed output from virtual terminal
func (tr *TerminalRouter) GetRecentOutput(podKey string, lines int, raw bool) []byte {
	if raw {
		// Return raw scrollback data
		tr.scrollbackMu.RLock()
		buffer := tr.scrollbackBuffers[podKey]
		tr.scrollbackMu.RUnlock()

		if buffer == nil {
			return nil
		}
		return buffer.GetRecentLines(lines)
	}

	// Try to return processed output from virtual terminal
	tr.virtualTermMu.RLock()
	vt := tr.virtualTerminals[podKey]
	tr.virtualTermMu.RUnlock()

	if vt != nil {
		output := vt.GetOutput(lines)
		if output != "" {
			return []byte(output)
		}
	}

	// Fallback: if virtual terminal has no data, strip ANSI from raw scrollback
	tr.scrollbackMu.RLock()
	buffer := tr.scrollbackBuffers[podKey]
	tr.scrollbackMu.RUnlock()

	if buffer == nil {
		return nil
	}

	rawData := buffer.GetRecentLines(lines)
	if rawData == nil {
		return nil
	}

	// Strip ANSI escape sequences as fallback
	return []byte(terminal.StripANSI(string(rawData)))
}

// GetScreenSnapshot returns the current screen snapshot for agent observation
func (tr *TerminalRouter) GetScreenSnapshot(podKey string) string {
	tr.virtualTermMu.RLock()
	vt := tr.virtualTerminals[podKey]
	tr.virtualTermMu.RUnlock()

	if vt != nil {
		display := vt.GetDisplay()
		if display != "" {
			return display
		}
	}

	// Fallback: strip ANSI from raw scrollback and return last screen worth of lines
	tr.scrollbackMu.RLock()
	buffer := tr.scrollbackBuffers[podKey]
	tr.scrollbackMu.RUnlock()

	if buffer == nil {
		return ""
	}

	// Get approximately one screen worth of lines (default 24 lines)
	rawData := buffer.GetRecentLines(24)
	if rawData == nil {
		return ""
	}

	return terminal.StripANSI(string(rawData))
}

// GetCursorPosition returns the current cursor position (row, col) for a pod
func (tr *TerminalRouter) GetCursorPosition(podKey string) (row, col int) {
	tr.virtualTermMu.RLock()
	vt := tr.virtualTerminals[podKey]
	tr.virtualTermMu.RUnlock()

	if vt == nil {
		return 0, 0
	}
	return vt.CursorPosition()
}

// GetClientCount returns the number of clients connected to a pod
func (tr *TerminalRouter) GetClientCount(podKey string) int {
	tr.terminalClientsMu.RLock()
	defer tr.terminalClientsMu.RUnlock()
	return len(tr.terminalClients[podKey])
}

// IsPodRegistered checks if a pod is registered
func (tr *TerminalRouter) IsPodRegistered(podKey string) bool {
	tr.podRunnerMu.RLock()
	defer tr.podRunnerMu.RUnlock()
	_, ok := tr.podRunnerMap[podKey]
	return ok
}

// GetRunnerID returns the runner ID for a pod
func (tr *TerminalRouter) GetRunnerID(podKey string) (int64, bool) {
	tr.podRunnerMu.RLock()
	defer tr.podRunnerMu.RUnlock()
	id, ok := tr.podRunnerMap[podKey]
	return id, ok
}

// GetAllScrollbackData returns all scrollback buffer data
func (tr *TerminalRouter) GetAllScrollbackData(podKey string) []byte {
	tr.scrollbackMu.RLock()
	buffer := tr.scrollbackBuffers[podKey]
	tr.scrollbackMu.RUnlock()

	if buffer == nil {
		return nil
	}

	return buffer.GetData()
}

// ClearScrollback clears the scrollback buffer for a pod
func (tr *TerminalRouter) ClearScrollback(podKey string) {
	tr.scrollbackMu.RLock()
	buffer := tr.scrollbackBuffers[podKey]
	tr.scrollbackMu.RUnlock()

	if buffer != nil {
		buffer.Clear()
	}
}

// detectAndPublishOSC777 detects OSC 777/OSC 9 notification sequences and publishes events
// OSC 777 format: ESC ] 777 ; notify ; <title> ; <body> BEL
// OSC 9 format: ESC ] 9 ; <message> BEL (used by Claude Code, iTerm2, Windows Terminal)
func (tr *TerminalRouter) detectAndPublishOSC777(podKey string, data []byte) {
	// Skip if EventBus or PodInfoGetter is not configured
	if tr.eventBus == nil || tr.podInfoGetter == nil {
		return
	}

	// Collect notifications from both OSC 777 and OSC 9
	type notification struct {
		title string
		body  string
	}
	var notifications []notification

	// Find OSC 777 matches (title ; body format)
	matches777 := osc777Pattern.FindAllSubmatch(data, -1)
	for _, match := range matches777 {
		if len(match) >= 3 {
			notifications = append(notifications, notification{
				title: string(match[1]),
				body:  string(match[2]),
			})
		}
	}

	// Find OSC 9 matches (single message format, used by Claude Code)
	matches9 := osc9Pattern.FindAllSubmatch(data, -1)
	for _, match := range matches9 {
		if len(match) >= 2 {
			msg := string(match[1])
			notifications = append(notifications, notification{
				title: "Terminal Notification",
				body:  msg,
			})
		}
	}

	if len(notifications) == 0 {
		return
	}

	// Get pod organization and creator info
	ctx := context.Background()
	orgID, creatorID, err := tr.podInfoGetter.GetPodOrganizationAndCreator(ctx, podKey)
	if err != nil {
		tr.logger.Warn("failed to get pod info for terminal notification",
			"pod_key", podKey,
			"error", err)
		return
	}

	// Process each notification
	for _, n := range notifications {
		tr.logger.Info("detected terminal notification (OSC 777/9)",
			"pod_key", podKey,
			"title", n.title,
			"body", n.body)

		// Publish terminal notification event
		tr.eventBus.Publish(ctx, &eventbus.Event{
			Type:           eventbus.EventTerminalNotification,
			Category:       eventbus.CategoryNotification,
			OrganizationID: orgID,
			TargetUserID:   &creatorID, // Send only to pod creator
			EntityType:     "pod",
			EntityID:       podKey,
			Data: json.RawMessage(`{
				"title": "` + escapeJSON(n.title) + `",
				"body": "` + escapeJSON(n.body) + `",
				"pod_key": "` + podKey + `"
			}`),
		})
	}
}

// escapeJSON escapes special characters in JSON string values
func escapeJSON(s string) string {
	var result bytes.Buffer
	for _, r := range s {
		switch r {
		case '"':
			result.WriteString(`\"`)
		case '\\':
			result.WriteString(`\\`)
		case '\n':
			result.WriteString(`\n`)
		case '\r':
			result.WriteString(`\r`)
		case '\t':
			result.WriteString(`\t`)
		default:
			result.WriteRune(r)
		}
	}
	return result.String()
}
