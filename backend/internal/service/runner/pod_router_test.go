package runner

import (
	"testing"
)

func TestNewPodRouter(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	tr := NewPodRouter(cm, newTestLogger())

	if tr == nil {
		t.Fatal("NewPodRouter returned nil")
	}
	if tr.connectionManager != cm {
		t.Error("connectionManager not set correctly")
	}
	// Check shards are initialized
	if tr.shards[0] == nil {
		t.Error("shards should be initialized")
	}
}

func TestPodRouterRegisterPod(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	tr := NewPodRouter(cm, newTestLogger())

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

func TestPodRouterUnregisterPod(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	tr := NewPodRouter(cm, newTestLogger())

	tr.RegisterPod("pod-1", 100)

	// Unregister
	tr.UnregisterPod("pod-1")

	// Check pod is unregistered
	if tr.IsPodRegistered("pod-1") {
		t.Error("pod should be unregistered")
	}
}

func TestPodRouterIsPodRegistered(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	tr := NewPodRouter(cm, newTestLogger())

	if tr.IsPodRegistered("nonexistent") {
		t.Error("nonexistent pod should not be registered")
	}

	tr.RegisterPod("pod-1", 100)
	if !tr.IsPodRegistered("pod-1") {
		t.Error("registered pod should be found")
	}
}

func TestPodRouterGetRunnerID(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	tr := NewPodRouter(cm, newTestLogger())

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

func TestPodRouterRoutePodInputNoRunner(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	tr := NewPodRouter(cm, newTestLogger())
	tr.SetCommandSender(&MockCommandSender{})

	err := tr.RoutePodInput("nonexistent", []byte("test"))
	if err != ErrRunnerNotConnected {
		t.Errorf("err = %v, want ErrRunnerNotConnected", err)
	}
}

func TestPodRouterRoutePodInputWithNoOpCommandSender(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	tr := NewPodRouter(cm, newTestLogger())
	// Default NoOpCommandSender is used

	// Register a pod so we get past the runner lookup
	tr.RegisterPod("pod-1", 100)

	// NoOpCommandSender should return ErrCommandSenderNotSet
	err := tr.RoutePodInput("pod-1", []byte("test"))
	if err != ErrCommandSenderNotSet {
		t.Errorf("err = %v, want ErrCommandSenderNotSet", err)
	}
}

func TestPodRouterSharding(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	tr := NewPodRouter(cm, newTestLogger())

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

func TestPodRouterGetRegisteredPodCount(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	tr := NewPodRouter(cm, newTestLogger())

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

func TestPodRouterSetEventBus(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	tr := NewPodRouter(cm, newTestLogger())

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

func TestPodRouterSetPodInfoGetter(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	tr := NewPodRouter(cm, newTestLogger())

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

func TestPodRouterRoutePodInputWithMockSender(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	tr := NewPodRouter(cm, newTestLogger())

	mockSender := &MockCommandSender{}
	tr.SetCommandSender(mockSender)

	// Register pod
	tr.RegisterPod("pod-1", 100)

	// Route input
	err := tr.RoutePodInput("pod-1", []byte("test input"))
	if err != nil {
		t.Errorf("RoutePodInput error: %v", err)
	}

	// Verify mock sender was called
	if mockSender.PodInputCalls != 1 {
		t.Errorf("PodInputCalls = %d, want 1", mockSender.PodInputCalls)
	}
}

func TestPodRouterEnsurePodRegistered(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	tr := NewPodRouter(cm, newTestLogger())

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

func TestPodRouterRoutePromptNoRunner(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	tr := NewPodRouter(cm, newTestLogger())
	tr.SetCommandSender(&MockCommandSender{})

	err := tr.RoutePrompt("nonexistent", "hello")
	if err != ErrRunnerNotConnected {
		t.Errorf("err = %v, want ErrRunnerNotConnected", err)
	}
}

func TestPodRouterRoutePromptWithNoOpCommandSender(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	tr := NewPodRouter(cm, newTestLogger())
	// Default NoOpCommandSender is used

	// Register a pod so we get past the runner lookup
	tr.RegisterPod("pod-1", 100)

	// NoOpCommandSender should return ErrCommandSenderNotSet
	err := tr.RoutePrompt("pod-1", "hello")
	if err != ErrCommandSenderNotSet {
		t.Errorf("err = %v, want ErrCommandSenderNotSet", err)
	}
}

func TestPodRouterRoutePromptWithMockSender(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	tr := NewPodRouter(cm, newTestLogger())

	mockSender := &MockCommandSender{}
	tr.SetCommandSender(mockSender)

	// Register pod
	tr.RegisterPod("pod-1", 100)

	// Route prompt
	err := tr.RoutePrompt("pod-1", "Fix the bug in main.go")
	if err != nil {
		t.Errorf("RoutePrompt error: %v", err)
	}

	// Verify mock sender was called
	if mockSender.SendPromptCalls != 1 {
		t.Errorf("SendPromptCalls = %d, want 1", mockSender.SendPromptCalls)
	}
}
