package runner

import (
	"sync"
	"testing"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/autopilot"
	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/config"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/detector"
)

// --- Test mocks for autopilot interfaces ---

type stubPodController struct {
	workDir string
	podKey  string
}

func (s *stubPodController) SendInput(string) error                    { return nil }
func (s *stubPodController) GetWorkDir() string                        { return s.workDir }
func (s *stubPodController) GetPodKey() string                         { return s.podKey }
func (s *stubPodController) GetAgentStatus() string                    { return "idle" }
func (s *stubPodController) GetStateDetector() detector.StateDetector  { return nil }

type stubEventReporter struct{}

func (s *stubEventReporter) ReportAutopilotStatus(*runnerv1.AutopilotStatusEvent)         {}
func (s *stubEventReporter) ReportAutopilotIteration(*runnerv1.AutopilotIterationEvent)   {}
func (s *stubEventReporter) ReportAutopilotCreated(*runnerv1.AutopilotCreatedEvent)       {}
func (s *stubEventReporter) ReportAutopilotTerminated(*runnerv1.AutopilotTerminatedEvent) {}
func (s *stubEventReporter) ReportAutopilotThinking(*runnerv1.AutopilotThinkingEvent)     {}

// newTestAutopilotController creates an AutopilotController suitable for unit tests.
// Uses temp dir for workDir to satisfy MCP config file creation.
func newTestAutopilotController(t *testing.T, apKey, podKey string) *autopilot.AutopilotController {
	t.Helper()
	return autopilot.NewAutopilotController(autopilot.Config{
		AutopilotKey: apKey,
		PodKey:       podKey,
		ProtoConfig:  &runnerv1.AutopilotConfig{MaxIterations: 5},
		PodCtrl:      &stubPodController{workDir: t.TempDir(), podKey: podKey},
		Reporter:     &stubEventReporter{},
	})
}

// TestOnTerminatePodCleansUpAutopilot verifies that OnTerminatePod stops
// and removes an associated Autopilot when the pod is terminated.
func TestOnTerminatePodCleansUpAutopilot(t *testing.T) {
	tempDir := t.TempDir()
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{
		cfg:        &config.Config{WorkspaceRoot: tempDir},
		autopilots: make(map[string]*autopilot.AutopilotController),
	}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	ac := newTestAutopilotController(t, "ap-test-1", "pod-with-autopilot")
	runner.AddAutopilot(ac)

	// Verify autopilot is registered
	if runner.GetAutopilotByPodKey("pod-with-autopilot") == nil {
		t.Fatal("autopilot should be registered before terminate")
	}

	// Add pod to store
	store.Put("pod-with-autopilot", &Pod{
		ID:       "pod-with-autopilot",
		PodKey:   "pod-with-autopilot",
		Terminal: nil,
	})

	// Terminate the pod
	err := handler.OnTerminatePod(client.TerminatePodRequest{
		PodKey: "pod-with-autopilot",
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify autopilot was cleaned up
	if runner.GetAutopilotByPodKey("pod-with-autopilot") != nil {
		t.Error("autopilot should be removed after pod termination")
	}
	if runner.GetAutopilot("ap-test-1") != nil {
		t.Error("autopilot should be removed from map")
	}
}

// TestExitHandlerCleansUpAutopilot verifies that the exit handler stops
// and removes an associated Autopilot when the pod process exits.
func TestExitHandlerCleansUpAutopilot(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{
		cfg:        &config.Config{},
		autopilots: make(map[string]*autopilot.AutopilotController),
	}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	ac := newTestAutopilotController(t, "ap-exit-1", "pod-exit-autopilot")
	runner.AddAutopilot(ac)

	// Add pod to store
	store.Put("pod-exit-autopilot", &Pod{
		ID:       "pod-exit-autopilot",
		PodKey:   "pod-exit-autopilot",
		Terminal: nil,
	})

	// Create exit handler and invoke it
	exitHandler := handler.createExitHandler("pod-exit-autopilot")
	exitHandler(0) // simulate process exit with code 0

	// Verify autopilot was cleaned up
	if runner.GetAutopilotByPodKey("pod-exit-autopilot") != nil {
		t.Error("autopilot should be removed after pod exit")
	}
	if runner.GetAutopilot("ap-exit-1") != nil {
		t.Error("autopilot should be removed from map after exit")
	}
}

// TestOnTerminatePodWithoutAutopilot verifies termination works normally
// when no autopilot is associated (regression guard).
func TestOnTerminatePodWithoutAutopilot(t *testing.T) {
	tempDir := t.TempDir()
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{
		cfg:        &config.Config{WorkspaceRoot: tempDir},
		autopilots: make(map[string]*autopilot.AutopilotController),
	}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	store.Put("plain-pod", &Pod{
		ID:       "plain-pod",
		PodKey:   "plain-pod",
		Terminal: nil,
	})

	err := handler.OnTerminatePod(client.TerminatePodRequest{
		PodKey: "plain-pod",
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestConcurrentTerminateWithAutopilot verifies no race when pod termination
// and autopilot access happen concurrently.
func TestConcurrentTerminateWithAutopilot(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{
		cfg:        &config.Config{},
		autopilots: make(map[string]*autopilot.AutopilotController),
	}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	const podCount = 20
	var wg sync.WaitGroup

	// Create pods with autopilots
	for i := 0; i < podCount; i++ {
		podKey := "concurrent-pod-" + string(rune('A'+i))
		apKey := "ap-" + podKey

		store.Put(podKey, &Pod{
			ID:       podKey,
			PodKey:   podKey,
			Terminal: nil,
		})

		ac := newTestAutopilotController(t, apKey, podKey)
		runner.AddAutopilot(ac)
	}

	// Terminate all pods concurrently
	for i := 0; i < podCount; i++ {
		wg.Add(1)
		podKey := "concurrent-pod-" + string(rune('A'+i))
		go func(pk string) {
			defer wg.Done()
			_ = handler.OnTerminatePod(client.TerminatePodRequest{PodKey: pk})
		}(podKey)
	}

	wg.Wait()

	// All autopilots should be cleaned up
	runner.autopilotsMu.RLock()
	remaining := len(runner.autopilots)
	runner.autopilotsMu.RUnlock()
	if remaining != 0 {
		t.Errorf("expected 0 remaining autopilots, got %d", remaining)
	}
}
