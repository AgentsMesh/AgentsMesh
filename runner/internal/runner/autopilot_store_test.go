package runner

import (
	"testing"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/autopilot"
)

// --- Stub types for autopilot tests ---

type apStubPodCtrl struct{ podKey string }

func (s *apStubPodCtrl) SendInput(string) error                    { return nil }
func (s *apStubPodCtrl) GetWorkDir() string                        { return "/tmp" }
func (s *apStubPodCtrl) GetPodKey() string                         { return s.podKey }
func (s *apStubPodCtrl) GetAgentStatus() string                    { return "idle" }
func (s *apStubPodCtrl) SubscribeStateChange(string, func(string)) {}
func (s *apStubPodCtrl) UnsubscribeStateChange(string)             {}

type apStubReporter struct{}

func (s *apStubReporter) ReportAutopilotStatus(*runnerv1.AutopilotStatusEvent)         {}
func (s *apStubReporter) ReportAutopilotIteration(*runnerv1.AutopilotIterationEvent)   {}
func (s *apStubReporter) ReportAutopilotCreated(*runnerv1.AutopilotCreatedEvent)       {}
func (s *apStubReporter) ReportAutopilotTerminated(*runnerv1.AutopilotTerminatedEvent) {}
func (s *apStubReporter) ReportAutopilotThinking(*runnerv1.AutopilotThinkingEvent)     {}

func makeTestAC(t *testing.T, key, podKey string) *autopilot.AutopilotController {
	t.Helper()
	return autopilot.NewAutopilotController(autopilot.Config{
		AutopilotKey: key,
		PodKey:       podKey,
		ProtoConfig:  &runnerv1.AutopilotConfig{MaxIterations: 1},
		PodCtrl:      &apStubPodCtrl{podKey: podKey},
		Reporter:     &apStubReporter{},
	})
}

// --- AutopilotStore tests ---

func TestAutopilotStoreAddAndGet(t *testing.T) {
	store := NewAutopilotStore()
	ac := makeTestAC(t, "ap-1", "pod-1")

	store.AddAutopilot(ac)

	got := store.GetAutopilot("ap-1")
	if got == nil {
		t.Fatal("expected autopilot to be found")
	}
	if got.Key() != "ap-1" {
		t.Errorf("key = %q, want ap-1", got.Key())
	}
}

func TestAutopilotStoreGetNonExistent(t *testing.T) {
	store := NewAutopilotStore()
	if store.GetAutopilot("missing") != nil {
		t.Error("expected nil for missing key")
	}
}

func TestAutopilotStoreRemove(t *testing.T) {
	store := NewAutopilotStore()
	ac := makeTestAC(t, "ap-rm", "pod-rm")
	store.AddAutopilot(ac)

	store.RemoveAutopilot("ap-rm")

	if store.GetAutopilot("ap-rm") != nil {
		t.Error("expected nil after remove")
	}
}

func TestAutopilotStoreGetByPodKey(t *testing.T) {
	store := NewAutopilotStore()
	ac1 := makeTestAC(t, "ap-a", "pod-target")
	ac2 := makeTestAC(t, "ap-b", "pod-other")
	store.AddAutopilot(ac1)
	store.AddAutopilot(ac2)

	got := store.GetAutopilotByPodKey("pod-target")
	if got == nil || got.Key() != "ap-a" {
		t.Errorf("expected ap-a for pod-target, got %v", got)
	}

	if store.GetAutopilotByPodKey("no-such-pod") != nil {
		t.Error("expected nil for non-existent pod key")
	}
}

func TestAutopilotStoreDrainAll(t *testing.T) {
	store := NewAutopilotStore()
	store.AddAutopilot(makeTestAC(t, "ap-1", "pod-1"))
	store.AddAutopilot(makeTestAC(t, "ap-2", "pod-2"))
	store.AddAutopilot(makeTestAC(t, "ap-3", "pod-3"))

	drained := store.DrainAll()
	if len(drained) != 3 {
		t.Errorf("DrainAll returned %d items, want 3", len(drained))
	}

	// Store should be empty after drain
	if store.GetAutopilot("ap-1") != nil {
		t.Error("store should be empty after DrainAll")
	}
}

func TestAutopilotStoreDrainAllEmpty(t *testing.T) {
	store := NewAutopilotStore()
	drained := store.DrainAll()
	if len(drained) != 0 {
		t.Errorf("DrainAll on empty store returned %d items, want 0", len(drained))
	}
}

func TestAutopilotStoreInterfaceCompliance(t *testing.T) {
	var _ AutopilotRegistry = (*AutopilotStore)(nil)
}
