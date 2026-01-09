package runner

import (
	"bytes"
	"log/slog"
	"sync"
	"unicode/utf8"

	"github.com/anthropics/agentmesh/backend/internal/infra/terminal"
	"github.com/gorilla/websocket"
)

const (
	// Default scrollback buffer size (100KB)
	DefaultScrollbackSize = 100 * 1024
)

// TerminalClient represents a frontend WebSocket client connected to a terminal
type TerminalClient struct {
	Conn      *websocket.Conn
	SessionID string
	Send      chan []byte
}

// TerminalRouter routes terminal data between frontend clients and runners
type TerminalRouter struct {
	connectionManager *ConnectionManager
	logger            *slog.Logger

	// Session -> Runner mapping
	sessionRunnerMap map[string]int64
	sessionRunnerMu  sync.RWMutex

	// Session -> Frontend clients
	terminalClients   map[string]map[*TerminalClient]bool
	terminalClientsMu sync.RWMutex

	// Scrollback buffers for reconnection (raw output for frontend)
	scrollbackBuffers map[string]*ScrollbackBuffer
	scrollbackMu      sync.RWMutex

	// Virtual terminals for agent observation (processed output)
	virtualTerminals map[string]*terminal.VirtualTerminal
	virtualTermMu    sync.RWMutex

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
		sessionRunnerMap:  make(map[string]int64),
		terminalClients:   make(map[string]map[*TerminalClient]bool),
		scrollbackBuffers: make(map[string]*ScrollbackBuffer),
		virtualTerminals:  make(map[string]*terminal.VirtualTerminal),
		scrollbackSize:    DefaultScrollbackSize,
	}

	// Set up callbacks from connection manager
	cm.SetTerminalOutputCallback(tr.handleTerminalOutput)
	cm.SetPtyResizedCallback(tr.handlePtyResized)

	return tr
}

// DefaultTerminalCols is the default terminal width
const DefaultTerminalCols = 80

// DefaultTerminalRows is the default terminal height
const DefaultTerminalRows = 24

// DefaultVirtualTerminalHistory is the default scrollback history lines
const DefaultVirtualTerminalHistory = 10000

// RegisterSession registers a session's runner mapping
func (tr *TerminalRouter) RegisterSession(sessionID string, runnerID int64) {
	tr.RegisterSessionWithSize(sessionID, runnerID, DefaultTerminalCols, DefaultTerminalRows)
}

// RegisterSessionWithSize registers a session with specific terminal size
func (tr *TerminalRouter) RegisterSessionWithSize(sessionID string, runnerID int64, cols, rows int) {
	tr.sessionRunnerMu.Lock()
	tr.sessionRunnerMap[sessionID] = runnerID
	tr.sessionRunnerMu.Unlock()

	// Initialize scrollback buffer
	tr.scrollbackMu.Lock()
	if _, exists := tr.scrollbackBuffers[sessionID]; !exists {
		tr.scrollbackBuffers[sessionID] = NewScrollbackBuffer(tr.scrollbackSize)
	}
	tr.scrollbackMu.Unlock()

	// Initialize virtual terminal for agent observation
	tr.virtualTermMu.Lock()
	if vt, exists := tr.virtualTerminals[sessionID]; !exists {
		tr.virtualTerminals[sessionID] = terminal.NewVirtualTerminal(cols, rows, DefaultVirtualTerminalHistory)
	} else {
		vt.Resize(cols, rows)
	}
	tr.virtualTermMu.Unlock()

	tr.logger.Debug("session registered",
		"session_id", sessionID,
		"runner_id", runnerID,
		"cols", cols,
		"rows", rows)
}

// UnregisterSession unregisters a session
func (tr *TerminalRouter) UnregisterSession(sessionID string) {
	tr.sessionRunnerMu.Lock()
	delete(tr.sessionRunnerMap, sessionID)
	tr.sessionRunnerMu.Unlock()

	// Clean up scrollback buffer
	tr.scrollbackMu.Lock()
	delete(tr.scrollbackBuffers, sessionID)
	tr.scrollbackMu.Unlock()

	// Clean up virtual terminal
	tr.virtualTermMu.Lock()
	delete(tr.virtualTerminals, sessionID)
	tr.virtualTermMu.Unlock()

	// Disconnect all clients
	tr.terminalClientsMu.Lock()
	clients := tr.terminalClients[sessionID]
	delete(tr.terminalClients, sessionID)
	tr.terminalClientsMu.Unlock()

	// Close client connections
	for client := range clients {
		close(client.Send)
		client.Conn.Close()
	}

	tr.logger.Debug("session unregistered", "session_id", sessionID)
}

// ConnectClient connects a frontend client to a session
func (tr *TerminalRouter) ConnectClient(sessionID string, conn *websocket.Conn) (*TerminalClient, error) {
	client := &TerminalClient{
		Conn:      conn,
		SessionID: sessionID,
		Send:      make(chan []byte, 256),
	}

	tr.terminalClientsMu.Lock()
	if tr.terminalClients[sessionID] == nil {
		tr.terminalClients[sessionID] = make(map[*TerminalClient]bool)
	}
	tr.terminalClients[sessionID][client] = true
	tr.terminalClientsMu.Unlock()

	tr.logger.Info("terminal client connected", "session_id", sessionID)

	// Send scrollback data to the newly connected client
	tr.scrollbackMu.RLock()
	buffer := tr.scrollbackBuffers[sessionID]
	tr.scrollbackMu.RUnlock()

	if buffer != nil {
		data := buffer.GetData()
		if len(data) > 0 {
			select {
			case client.Send <- data:
				tr.logger.Debug("sent scrollback to client",
					"session_id", sessionID,
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
	if clients, ok := tr.terminalClients[client.SessionID]; ok {
		delete(clients, client)
		if len(clients) == 0 {
			delete(tr.terminalClients, client.SessionID)
		}
	}
	tr.terminalClientsMu.Unlock()

	close(client.Send)
	tr.logger.Info("terminal client disconnected", "session_id", client.SessionID)
}

// handleTerminalOutput handles terminal output from a runner
func (tr *TerminalRouter) handleTerminalOutput(runnerID int64, data *TerminalOutputData) {
	sessionID := data.SessionID

	// Store in scrollback buffer (raw data for frontend)
	tr.scrollbackMu.RLock()
	buffer := tr.scrollbackBuffers[sessionID]
	tr.scrollbackMu.RUnlock()

	if buffer != nil {
		buffer.Write(data.Data)
	}

	// Feed to virtual terminal (processed data for agent observation)
	tr.virtualTermMu.RLock()
	vt := tr.virtualTerminals[sessionID]
	tr.virtualTermMu.RUnlock()

	if vt != nil {
		vt.Feed(data.Data)
	}

	// Route to all connected clients
	tr.terminalClientsMu.RLock()
	clients := tr.terminalClients[sessionID]
	tr.terminalClientsMu.RUnlock()

	if len(clients) == 0 {
		tr.logger.Debug("no clients for terminal output", "session_id", sessionID)
		return
	}

	// Broadcast to all clients
	var deadClients []*TerminalClient
	for client := range clients {
		select {
		case client.Send <- data.Data:
		default:
			// Client buffer full, mark for removal
			deadClients = append(deadClients, client)
		}
	}

	// Clean up dead clients
	if len(deadClients) > 0 {
		tr.terminalClientsMu.Lock()
		for _, client := range deadClients {
			delete(tr.terminalClients[sessionID], client)
		}
		tr.terminalClientsMu.Unlock()
	}
}

// handlePtyResized handles PTY resize notifications from runner
func (tr *TerminalRouter) handlePtyResized(runnerID int64, data *PtyResizedData) {
	sessionID := data.SessionID

	// Update virtual terminal size
	tr.virtualTermMu.Lock()
	if vt, exists := tr.virtualTerminals[sessionID]; exists {
		vt.Resize(data.Cols, data.Rows)
		tr.logger.Debug("virtual terminal resized",
			"session_id", sessionID,
			"cols", data.Cols,
			"rows", data.Rows)
	}
	tr.virtualTermMu.Unlock()
}

// RouteInput routes terminal input from frontend to runner
func (tr *TerminalRouter) RouteInput(sessionID string, data []byte) error {
	tr.sessionRunnerMu.RLock()
	runnerID, ok := tr.sessionRunnerMap[sessionID]
	tr.sessionRunnerMu.RUnlock()

	if !ok {
		tr.logger.Warn("no runner for session", "session_id", sessionID)
		return ErrRunnerNotConnected
	}

	return tr.connectionManager.SendTerminalInput(nil, runnerID, sessionID, data)
}

// RouteResize routes terminal resize from frontend to runner
func (tr *TerminalRouter) RouteResize(sessionID string, cols, rows int) error {
	tr.sessionRunnerMu.RLock()
	runnerID, ok := tr.sessionRunnerMap[sessionID]
	tr.sessionRunnerMu.RUnlock()

	if !ok {
		tr.logger.Warn("no runner for session", "session_id", sessionID)
		return ErrRunnerNotConnected
	}

	return tr.connectionManager.SendTerminalResize(nil, runnerID, sessionID, cols, rows)
}

// GetRecentOutput returns recent terminal output for observation
// If raw is true, returns raw scrollback data; otherwise returns processed output from virtual terminal
func (tr *TerminalRouter) GetRecentOutput(sessionID string, lines int, raw bool) []byte {
	if raw {
		// Return raw scrollback data
		tr.scrollbackMu.RLock()
		buffer := tr.scrollbackBuffers[sessionID]
		tr.scrollbackMu.RUnlock()

		if buffer == nil {
			return nil
		}
		return buffer.GetRecentLines(lines)
	}

	// Try to return processed output from virtual terminal
	tr.virtualTermMu.RLock()
	vt := tr.virtualTerminals[sessionID]
	tr.virtualTermMu.RUnlock()

	if vt != nil {
		output := vt.GetOutput(lines)
		if output != "" {
			return []byte(output)
		}
	}

	// Fallback: if virtual terminal has no data, strip ANSI from raw scrollback
	tr.scrollbackMu.RLock()
	buffer := tr.scrollbackBuffers[sessionID]
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
func (tr *TerminalRouter) GetScreenSnapshot(sessionID string) string {
	tr.virtualTermMu.RLock()
	vt := tr.virtualTerminals[sessionID]
	tr.virtualTermMu.RUnlock()

	if vt != nil {
		display := vt.GetDisplay()
		if display != "" {
			return display
		}
	}

	// Fallback: strip ANSI from raw scrollback and return last screen worth of lines
	tr.scrollbackMu.RLock()
	buffer := tr.scrollbackBuffers[sessionID]
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

// GetCursorPosition returns the current cursor position (row, col) for a session
func (tr *TerminalRouter) GetCursorPosition(sessionID string) (row, col int) {
	tr.virtualTermMu.RLock()
	vt := tr.virtualTerminals[sessionID]
	tr.virtualTermMu.RUnlock()

	if vt == nil {
		return 0, 0
	}
	return vt.CursorPosition()
}

// GetClientCount returns the number of clients connected to a session
func (tr *TerminalRouter) GetClientCount(sessionID string) int {
	tr.terminalClientsMu.RLock()
	defer tr.terminalClientsMu.RUnlock()
	return len(tr.terminalClients[sessionID])
}

// IsSessionRegistered checks if a session is registered
func (tr *TerminalRouter) IsSessionRegistered(sessionID string) bool {
	tr.sessionRunnerMu.RLock()
	defer tr.sessionRunnerMu.RUnlock()
	_, ok := tr.sessionRunnerMap[sessionID]
	return ok
}

// GetRunnerID returns the runner ID for a session
func (tr *TerminalRouter) GetRunnerID(sessionID string) (int64, bool) {
	tr.sessionRunnerMu.RLock()
	defer tr.sessionRunnerMu.RUnlock()
	id, ok := tr.sessionRunnerMap[sessionID]
	return id, ok
}

// GetAllScrollbackData returns all scrollback buffer data
func (tr *TerminalRouter) GetAllScrollbackData(sessionID string) []byte {
	tr.scrollbackMu.RLock()
	buffer := tr.scrollbackBuffers[sessionID]
	tr.scrollbackMu.RUnlock()

	if buffer == nil {
		return nil
	}

	return buffer.GetData()
}

// ClearScrollback clears the scrollback buffer for a session
func (tr *TerminalRouter) ClearScrollback(sessionID string) {
	tr.scrollbackMu.RLock()
	buffer := tr.scrollbackBuffers[sessionID]
	tr.scrollbackMu.RUnlock()

	if buffer != nil {
		buffer.Clear()
	}
}
