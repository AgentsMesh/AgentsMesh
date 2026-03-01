package runner

import (
	"testing"

	runnerv1 "github.com/AgentsMesh/AgentsMesh/proto/gen/go/runner/v1"
)

func TestNewTerminalRouter(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
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
}

func TestTerminalRouterRegisterPod(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
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
}

func TestTerminalRouterUnregisterPod(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())

	tr.RegisterPod("pod-1", 100)

	// Unregister
	tr.UnregisterPod("pod-1")

	// Check pod is unregistered
	if tr.IsPodRegistered("pod-1") {
		t.Error("pod should be unregistered")
	}
}

func TestTerminalRouterIsPodRegistered(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
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
	cm := NewRunnerConnectionManager(newTestLogger())
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

func TestTerminalRouterPtyResized(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())

	// Register pod with default size (80x24)
	tr.RegisterPod("pod-1", 100)

	// Simulate pty_resized callback (using Proto type)
	tr.handlePtyResized(100, &runnerv1.PtyResizedEvent{
		PodKey: "pod-1",
		Cols:   120,
		Rows:   40,
	})

	// Verify size is stored
	cols, rows, ok := tr.GetPtySize("pod-1")
	if !ok {
		t.Error("should find PTY size")
	}
	if cols != 120 || rows != 40 {
		t.Errorf("PTY size = %dx%d, want 120x40", cols, rows)
	}
}

func TestTerminalRouterRouteInputNoRunner(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())
	tr.SetCommandSender(&MockCommandSender{})

	err := tr.RouteInput("nonexistent", []byte("test"))
	if err != ErrRunnerNotConnected {
		t.Errorf("err = %v, want ErrRunnerNotConnected", err)
	}
}

func TestTerminalRouterRouteResizeNoRunner(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())
	tr.SetCommandSender(&MockCommandSender{})

	err := tr.RouteResize("nonexistent", 80, 24)
	if err != ErrRunnerNotConnected {
		t.Errorf("err = %v, want ErrRunnerNotConnected", err)
	}
}

func TestTerminalRouterRouteInputWithNoOpCommandSender(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())
	// Default NoOpCommandSender is used

	// Register a pod so we get past the runner lookup
	tr.RegisterPod("pod-1", 100)

	// NoOpCommandSender should return ErrCommandSenderNotSet
	err := tr.RouteInput("pod-1", []byte("test"))
	if err != ErrCommandSenderNotSet {
		t.Errorf("err = %v, want ErrCommandSenderNotSet", err)
	}
}

func TestTerminalRouterRouteResizeWithNoOpCommandSender(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())
	// Default NoOpCommandSender is used

	// Register a pod so we get past the runner lookup
	tr.RegisterPod("pod-1", 100)

	// NoOpCommandSender should return ErrCommandSenderNotSet
	err := tr.RouteResize("pod-1", 80, 24)
	if err != ErrCommandSenderNotSet {
		t.Errorf("err = %v, want ErrCommandSenderNotSet", err)
	}
}

// TestTerminalRouterAutoRegisterOnResize tests that pod is auto-registered on PTY resize
func TestTerminalRouterAutoRegisterOnResize(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())

	// Verify pod is not registered
	if tr.IsPodRegistered("resize-pod") {
		t.Error("pod should not be registered initially")
	}

	// Send PTY resize without prior registration (simulates server restart scenario)
	tr.handlePtyResized(300, &runnerv1.PtyResizedEvent{
		PodKey: "resize-pod",
		Cols:   150,
		Rows:   50,
	})

	// Pod should be auto-registered
	if !tr.IsPodRegistered("resize-pod") {
		t.Error("pod should be auto-registered after resize")
	}

	// Runner ID should be recorded
	runnerID, ok := tr.GetRunnerID("resize-pod")
	if !ok || runnerID != 300 {
		t.Errorf("runner ID = %d, want 300", runnerID)
	}

	// PTY size should be stored
	cols, rows, ok := tr.GetPtySize("resize-pod")
	if !ok {
		t.Error("PTY size should be stored")
	}
	if cols != 150 || rows != 50 {
		t.Errorf("PTY size = %dx%d, want 150x50", cols, rows)
	}
}

func TestTerminalRouterSharding(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
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
	cm := NewRunnerConnectionManager(newTestLogger())
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

func TestTerminalRouterSetEventBus(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())

	// Initially nil
	if tr.oscDetector != nil {
		t.Error("oscDetector should be nil initially")
	}

	// Set event bus - this should create oscDetector
	tr.SetEventBus(nil) // nil eventbus is allowed for testing

	if tr.oscDetector == nil {
		t.Error("oscDetector should be created after SetEventBus")
	}
}

func TestTerminalRouterSetPodInfoGetter(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())

	// Initially nil
	if tr.oscDetector != nil {
		t.Error("oscDetector should be nil initially")
	}

	// Set pod info getter - this should create oscDetector
	tr.SetPodInfoGetter(nil) // nil getter is allowed for testing

	if tr.oscDetector == nil {
		t.Error("oscDetector should be created after SetPodInfoGetter")
	}
}

func TestTerminalRouterRouteInputWithMockSender(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())

	mockSender := &MockCommandSender{}
	tr.SetCommandSender(mockSender)

	// Register pod
	tr.RegisterPod("pod-1", 100)

	// Route input
	err := tr.RouteInput("pod-1", []byte("test input"))
	if err != nil {
		t.Errorf("RouteInput error: %v", err)
	}

	// Verify mock sender was called
	if mockSender.TerminalInputCalls != 1 {
		t.Errorf("TerminalInputCalls = %d, want 1", mockSender.TerminalInputCalls)
	}
}

func TestTerminalRouterRouteResizeWithMockSender(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())

	mockSender := &MockCommandSender{}
	tr.SetCommandSender(mockSender)

	// Register pod
	tr.RegisterPod("pod-1", 100)

	// Route resize
	err := tr.RouteResize("pod-1", 120, 40)
	if err != nil {
		t.Errorf("RouteResize error: %v", err)
	}

	// Verify mock sender was called
	if mockSender.TerminalResizeCalls != 1 {
		t.Errorf("TerminalResizeCalls = %d, want 1", mockSender.TerminalResizeCalls)
	}
}

func TestTerminalRouterEnsurePodRegistered(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())

	// Ensure pod is registered
	tr.EnsurePodRegistered("pod-1", 100)

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
}

func TestTerminalRouterGetPtySize(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())

	// Not found - returns defaults
	cols, rows, ok := tr.GetPtySize("nonexistent")
	if ok {
		t.Error("should not find nonexistent pod")
	}
	if cols != DefaultTerminalCols || rows != DefaultTerminalRows {
		t.Errorf("default size = %dx%d, want %dx%d", cols, rows, DefaultTerminalCols, DefaultTerminalRows)
	}

	// Register with size
	tr.RegisterPodWithSize("pod-1", 100, 120, 40)

	cols, rows, ok = tr.GetPtySize("pod-1")
	if !ok {
		t.Error("should find pod PTY size")
	}
	if cols != 120 || rows != 40 {
		t.Errorf("PTY size = %dx%d, want 120x40", cols, rows)
	}
}
