package runner

import (
	"log/slog"
	"os"
	"testing"
)

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestNewTerminalRouter(t *testing.T) {
	cm := NewConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())

	if tr == nil {
		t.Fatal("NewTerminalRouter returned nil")
	}
	if tr.connectionManager != cm {
		t.Error("connectionManager not set correctly")
	}
	if tr.sessionRunnerMap == nil {
		t.Error("sessionRunnerMap should be initialized")
	}
	if tr.terminalClients == nil {
		t.Error("terminalClients should be initialized")
	}
	if tr.scrollbackBuffers == nil {
		t.Error("scrollbackBuffers should be initialized")
	}
	if tr.scrollbackSize != DefaultScrollbackSize {
		t.Errorf("scrollbackSize = %d, want %d", tr.scrollbackSize, DefaultScrollbackSize)
	}
}

func TestTerminalRouterRegisterSession(t *testing.T) {
	cm := NewConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())

	tr.RegisterSession("session-1", 100)

	// Check session is registered
	if !tr.IsSessionRegistered("session-1") {
		t.Error("session should be registered")
	}

	// Check runner ID is stored
	runnerID, ok := tr.GetRunnerID("session-1")
	if !ok {
		t.Error("should find runner ID")
	}
	if runnerID != 100 {
		t.Errorf("runnerID = %d, want 100", runnerID)
	}

	// Check scrollback buffer is created
	tr.scrollbackMu.RLock()
	buffer := tr.scrollbackBuffers["session-1"]
	tr.scrollbackMu.RUnlock()
	if buffer == nil {
		t.Error("scrollback buffer should be created")
	}
}

func TestTerminalRouterUnregisterSession(t *testing.T) {
	cm := NewConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())

	tr.RegisterSession("session-1", 100)

	// Unregister
	tr.UnregisterSession("session-1")

	// Check session is unregistered
	if tr.IsSessionRegistered("session-1") {
		t.Error("session should be unregistered")
	}

	// Check scrollback buffer is removed
	tr.scrollbackMu.RLock()
	buffer := tr.scrollbackBuffers["session-1"]
	tr.scrollbackMu.RUnlock()
	if buffer != nil {
		t.Error("scrollback buffer should be removed")
	}
}

func TestTerminalRouterIsSessionRegistered(t *testing.T) {
	cm := NewConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())

	if tr.IsSessionRegistered("nonexistent") {
		t.Error("nonexistent session should not be registered")
	}

	tr.RegisterSession("session-1", 100)
	if !tr.IsSessionRegistered("session-1") {
		t.Error("registered session should be found")
	}
}

func TestTerminalRouterGetRunnerID(t *testing.T) {
	cm := NewConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())

	// Not found case
	_, ok := tr.GetRunnerID("nonexistent")
	if ok {
		t.Error("should not find nonexistent session")
	}

	tr.RegisterSession("session-1", 100)
	id, ok := tr.GetRunnerID("session-1")
	if !ok {
		t.Error("should find registered session")
	}
	if id != 100 {
		t.Errorf("runnerID = %d, want 100", id)
	}
}

func TestTerminalRouterGetClientCount(t *testing.T) {
	cm := NewConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())

	// No clients
	count := tr.GetClientCount("session-1")
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}

	// Add some mock clients
	tr.terminalClientsMu.Lock()
	tr.terminalClients["session-1"] = map[*TerminalClient]bool{
		{SessionID: "session-1"}: true,
		{SessionID: "session-1"}: true,
	}
	tr.terminalClientsMu.Unlock()

	count = tr.GetClientCount("session-1")
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}

func TestTerminalRouterGetRecentOutput(t *testing.T) {
	cm := NewConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())

	// No buffer
	output := tr.GetRecentOutput("nonexistent", 10, true)
	if output != nil {
		t.Error("should return nil for nonexistent session")
	}

	// Register session and add some output
	tr.RegisterSession("session-1", 100)
	tr.scrollbackMu.RLock()
	buffer := tr.scrollbackBuffers["session-1"]
	tr.scrollbackMu.RUnlock()
	buffer.Write([]byte("line1\nline2\nline3\n"))

	// Test raw output
	output = tr.GetRecentOutput("session-1", 2, true)
	if output == nil {
		t.Error("should return raw output")
	}

	// Test processed output (feed to virtual terminal first)
	tr.virtualTermMu.RLock()
	vt := tr.virtualTerminals["session-1"]
	tr.virtualTermMu.RUnlock()
	vt.Feed([]byte("Hello, World!"))

	processedOutput := tr.GetRecentOutput("session-1", 10, false)
	if processedOutput == nil {
		t.Error("should return processed output")
	}
}

func TestTerminalRouterGetScreenSnapshot(t *testing.T) {
	cm := NewConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())

	// No virtual terminal
	snapshot := tr.GetScreenSnapshot("nonexistent")
	if snapshot != "" {
		t.Error("should return empty string for nonexistent session")
	}

	// Register session
	tr.RegisterSession("session-1", 100)

	// Feed some data
	tr.virtualTermMu.RLock()
	vt := tr.virtualTerminals["session-1"]
	tr.virtualTermMu.RUnlock()
	vt.Feed([]byte("Hello, World!"))

	snapshot = tr.GetScreenSnapshot("session-1")
	if snapshot != "Hello, World!" {
		t.Errorf("snapshot = %q, want %q", snapshot, "Hello, World!")
	}
}

func TestTerminalRouterPtyResized(t *testing.T) {
	cm := NewConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())

	// Register session with default size (80x24)
	tr.RegisterSession("session-1", 100)

	tr.virtualTermMu.RLock()
	vt := tr.virtualTerminals["session-1"]
	tr.virtualTermMu.RUnlock()

	// Verify initial size
	if vt.Cols() != DefaultTerminalCols || vt.Rows() != DefaultTerminalRows {
		t.Errorf("initial size = %dx%d, want %dx%d", vt.Cols(), vt.Rows(), DefaultTerminalCols, DefaultTerminalRows)
	}

	// Simulate pty_resized callback
	tr.handlePtyResized(100, &PtyResizedData{
		SessionID: "session-1",
		Cols:      120,
		Rows:      40,
	})

	// Verify resized
	if vt.Cols() != 120 || vt.Rows() != 40 {
		t.Errorf("resized size = %dx%d, want 120x40", vt.Cols(), vt.Rows())
	}
}

func TestTerminalRouterGetAllScrollbackData(t *testing.T) {
	cm := NewConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())

	// No buffer
	data := tr.GetAllScrollbackData("nonexistent")
	if data != nil {
		t.Error("should return nil for nonexistent session")
	}

	// Register session and add some data
	tr.RegisterSession("session-1", 100)
	tr.scrollbackMu.RLock()
	buffer := tr.scrollbackBuffers["session-1"]
	tr.scrollbackMu.RUnlock()
	buffer.Write([]byte("test data"))

	data = tr.GetAllScrollbackData("session-1")
	if string(data) != "test data" {
		t.Errorf("data = %q, want %q", data, "test data")
	}
}

func TestTerminalRouterClearScrollback(t *testing.T) {
	cm := NewConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())

	// Clear nonexistent - should not panic
	tr.ClearScrollback("nonexistent")

	// Register and clear
	tr.RegisterSession("session-1", 100)
	tr.scrollbackMu.RLock()
	buffer := tr.scrollbackBuffers["session-1"]
	tr.scrollbackMu.RUnlock()
	buffer.Write([]byte("test data"))

	tr.ClearScrollback("session-1")

	data := buffer.GetData()
	if len(data) != 0 {
		t.Errorf("data should be cleared, got %q", data)
	}
}

func TestTerminalRouterRouteInputNoRunner(t *testing.T) {
	cm := NewConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())

	err := tr.RouteInput("nonexistent", []byte("test"))
	if err != ErrRunnerNotConnected {
		t.Errorf("err = %v, want ErrRunnerNotConnected", err)
	}
}

func TestTerminalRouterRouteResizeNoRunner(t *testing.T) {
	cm := NewConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())

	err := tr.RouteResize("nonexistent", 80, 24)
	if err != ErrRunnerNotConnected {
		t.Errorf("err = %v, want ErrRunnerNotConnected", err)
	}
}

func TestTerminalRouterHandleTerminalOutput(t *testing.T) {
	cm := NewConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())

	// Register session
	tr.RegisterSession("session-1", 100)

	// Handle output with no clients
	tr.handleTerminalOutput(100, &TerminalOutputData{
		SessionID: "session-1",
		Data:      []byte("test output"),
	})

	// Check scrollback buffer has the data
	data := tr.GetAllScrollbackData("session-1")
	if string(data) != "test output" {
		t.Errorf("scrollback = %q, want %q", data, "test output")
	}
}

func TestTerminalClientStruct(t *testing.T) {
	client := &TerminalClient{
		SessionID: "session-1",
		Send:      make(chan []byte, 256),
	}

	if client.SessionID != "session-1" {
		t.Errorf("SessionID = %s, want session-1", client.SessionID)
	}
	if client.Send == nil {
		t.Error("Send channel should be initialized")
	}
}

func TestDefaultScrollbackSize(t *testing.T) {
	if DefaultScrollbackSize != 100*1024 {
		t.Errorf("DefaultScrollbackSize = %d, want %d", DefaultScrollbackSize, 100*1024)
	}
}
