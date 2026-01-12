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
	// Check shards are initialized
	if tr.shards[0] == nil {
		t.Error("shards should be initialized")
	}
	if tr.scrollbackSize != DefaultScrollbackSize {
		t.Errorf("scrollbackSize = %d, want %d", tr.scrollbackSize, DefaultScrollbackSize)
	}
}

func TestTerminalRouterRegisterPod(t *testing.T) {
	cm := NewConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())

	tr.RegisterPod("pod-1", 100)

	// Check pod is registered
	if !tr.IsPodRegistered("pod-1") {
		t.Error("pod should be registered")
	}

	// Check runner ID is stored
	runnerID, ok := tr.GetRunnerID("pod-1")
	if !ok {
		t.Error("should find runner ID")
	}
	if runnerID != 100 {
		t.Errorf("runnerID = %d, want 100", runnerID)
	}

	// Check scrollback buffer is created
	shard := tr.getShard("pod-1")
	shard.mu.RLock()
	buffer := shard.scrollbackBuffers["pod-1"]
	shard.mu.RUnlock()
	if buffer == nil {
		t.Error("scrollback buffer should be created")
	}
}

func TestTerminalRouterUnregisterPod(t *testing.T) {
	cm := NewConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())

	tr.RegisterPod("pod-1", 100)

	// Unregister
	tr.UnregisterPod("pod-1")

	// Check pod is unregistered
	if tr.IsPodRegistered("pod-1") {
		t.Error("pod should be unregistered")
	}

	// Check scrollback buffer is removed
	shard := tr.getShard("pod-1")
	shard.mu.RLock()
	buffer := shard.scrollbackBuffers["pod-1"]
	shard.mu.RUnlock()
	if buffer != nil {
		t.Error("scrollback buffer should be removed")
	}
}

func TestTerminalRouterIsPodRegistered(t *testing.T) {
	cm := NewConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())

	if tr.IsPodRegistered("nonexistent") {
		t.Error("nonexistent pod should not be registered")
	}

	tr.RegisterPod("pod-1", 100)
	if !tr.IsPodRegistered("pod-1") {
		t.Error("registered pod should be found")
	}
}

func TestTerminalRouterGetRunnerID(t *testing.T) {
	cm := NewConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())

	// Not found case
	_, ok := tr.GetRunnerID("nonexistent")
	if ok {
		t.Error("should not find nonexistent pod")
	}

	tr.RegisterPod("pod-1", 100)
	id, ok := tr.GetRunnerID("pod-1")
	if !ok {
		t.Error("should find registered pod")
	}
	if id != 100 {
		t.Errorf("runnerID = %d, want 100", id)
	}
}

func TestTerminalRouterGetClientCount(t *testing.T) {
	cm := NewConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())

	// No clients
	count := tr.GetClientCount("pod-1")
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}

	// Add some mock clients using shard
	shard := tr.getShard("pod-1")
	shard.mu.Lock()
	shard.terminalClients["pod-1"] = map[*TerminalClient]bool{
		{PodKey: "pod-1"}: true,
		{PodKey: "pod-1"}: true,
	}
	shard.mu.Unlock()

	count = tr.GetClientCount("pod-1")
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
		t.Error("should return nil for nonexistent pod")
	}

	// Register pod and add some output
	tr.RegisterPod("pod-1", 100)
	shard := tr.getShard("pod-1")
	shard.mu.RLock()
	buffer := shard.scrollbackBuffers["pod-1"]
	shard.mu.RUnlock()
	buffer.Write([]byte("line1\nline2\nline3\n"))

	// Test raw output
	output = tr.GetRecentOutput("pod-1", 2, true)
	if output == nil {
		t.Error("should return raw output")
	}

	// Test processed output (feed to virtual terminal first)
	shard.mu.RLock()
	vt := shard.virtualTerminals["pod-1"]
	shard.mu.RUnlock()
	vt.Feed([]byte("Hello, World!"))

	processedOutput := tr.GetRecentOutput("pod-1", 10, false)
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
		t.Error("should return empty string for nonexistent pod")
	}

	// Register pod
	tr.RegisterPod("pod-1", 100)

	// Feed some data
	shard := tr.getShard("pod-1")
	shard.mu.RLock()
	vt := shard.virtualTerminals["pod-1"]
	shard.mu.RUnlock()
	vt.Feed([]byte("Hello, World!"))

	snapshot = tr.GetScreenSnapshot("pod-1")
	if snapshot != "Hello, World!" {
		t.Errorf("snapshot = %q, want %q", snapshot, "Hello, World!")
	}
}

func TestTerminalRouterPtyResized(t *testing.T) {
	cm := NewConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())

	// Register pod with default size (80x24)
	tr.RegisterPod("pod-1", 100)

	shard := tr.getShard("pod-1")
	shard.mu.RLock()
	vt := shard.virtualTerminals["pod-1"]
	shard.mu.RUnlock()

	// Verify initial size
	if vt.Cols() != DefaultTerminalCols || vt.Rows() != DefaultTerminalRows {
		t.Errorf("initial size = %dx%d, want %dx%d", vt.Cols(), vt.Rows(), DefaultTerminalCols, DefaultTerminalRows)
	}

	// Simulate pty_resized callback
	tr.handlePtyResized(100, &PtyResizedData{
		PodKey: "pod-1",
		Cols:   120,
		Rows:   40,
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
		t.Error("should return nil for nonexistent pod")
	}

	// Register pod and add some data
	tr.RegisterPod("pod-1", 100)
	shard := tr.getShard("pod-1")
	shard.mu.RLock()
	buffer := shard.scrollbackBuffers["pod-1"]
	shard.mu.RUnlock()
	buffer.Write([]byte("test data"))

	data = tr.GetAllScrollbackData("pod-1")
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
	tr.RegisterPod("pod-1", 100)
	shard := tr.getShard("pod-1")
	shard.mu.RLock()
	buffer := shard.scrollbackBuffers["pod-1"]
	shard.mu.RUnlock()
	buffer.Write([]byte("test data"))

	tr.ClearScrollback("pod-1")

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

	// Register pod
	tr.RegisterPod("pod-1", 100)

	// Handle output with no clients
	tr.handleTerminalOutput(100, &TerminalOutputData{
		PodKey: "pod-1",
		Data:   []byte("test output"),
	})

	// Check scrollback buffer has the data
	data := tr.GetAllScrollbackData("pod-1")
	if string(data) != "test output" {
		t.Errorf("scrollback = %q, want %q", data, "test output")
	}
}

func TestTerminalClientStruct(t *testing.T) {
	client := &TerminalClient{
		PodKey: "pod-1",
		Send:   make(chan TerminalMessage, 256),
	}

	if client.PodKey != "pod-1" {
		t.Errorf("PodKey = %s, want pod-1", client.PodKey)
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

func TestTerminalRouterSharding(t *testing.T) {
	cm := NewConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())

	// Register pods with different keys
	tr.RegisterPod("pod-1", 100)
	tr.RegisterPod("pod-2", 200)
	tr.RegisterPod("pod-3", 300)

	// All should be registered
	if !tr.IsPodRegistered("pod-1") || !tr.IsPodRegistered("pod-2") || !tr.IsPodRegistered("pod-3") {
		t.Error("all pods should be registered")
	}

	// Different pods might be in different shards (depends on hash)
	shard1 := tr.getShard("pod-1")
	shard2 := tr.getShard("pod-2")

	// At least verify that getShard returns valid shards
	if shard1 == nil || shard2 == nil {
		t.Error("getShard should return valid shards")
	}
}

func TestTerminalRouterGetRegisteredPodCount(t *testing.T) {
	cm := NewConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())

	// Initially zero
	count := tr.GetRegisteredPodCount()
	if count != 0 {
		t.Errorf("initial count = %d, want 0", count)
	}

	// Register some pods
	tr.RegisterPod("pod-1", 100)
	tr.RegisterPod("pod-2", 200)
	tr.RegisterPod("pod-3", 300)

	count = tr.GetRegisteredPodCount()
	if count != 3 {
		t.Errorf("count after registration = %d, want 3", count)
	}

	// Unregister one
	tr.UnregisterPod("pod-2")
	count = tr.GetRegisteredPodCount()
	if count != 2 {
		t.Errorf("count after unregister = %d, want 2", count)
	}
}
