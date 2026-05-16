package runner

import (
	"sync"
	"testing"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// sendPromptMockIO is a minimal PodIO that records SendInput calls and lets
// the test pick the interaction mode. It satisfies PodIO via embedded stubs.
type sendPromptMockIO struct {
	stubPodIOZero
	mu     sync.Mutex
	mode   string
	inputs []string
	err    error
}

func (m *sendPromptMockIO) Mode() string { return m.mode }

func (m *sendPromptMockIO) SendInput(text string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.inputs = append(m.inputs, text)
	return m.err
}

// stubPodIOZero satisfies the rest of PodIO with no-ops so each test mock
// only overrides the methods it cares about.
type stubPodIOZero struct{}

func (stubPodIOZero) GetSnapshot(int) (string, error)           { return "", nil }
func (stubPodIOZero) GetAgentStatus() string                    { return "idle" }
func (stubPodIOZero) SubscribeStateChange(string, func(string)) {}
func (stubPodIOZero) UnsubscribeStateChange(string)             {}
func (stubPodIOZero) GetPID() int                               { return 0 }
func (stubPodIOZero) Start() error                              { return nil }
func (stubPodIOZero) Stop()                                     {}
func (stubPodIOZero) Teardown() string                          { return "" }
func (stubPodIOZero) SetExitHandler(func(int))                  {}
func (stubPodIOZero) SetIOErrorHandler(func(error))             {}
func (stubPodIOZero) Detach()                                   {}

func TestOnSendPrompt_PTY_AutoSubmitsEnter(t *testing.T) {
	io := &sendPromptMockIO{mode: InteractionModePTY}
	pod := &Pod{PodKey: "pty-pod", InteractionMode: InteractionModePTY, IO: io}

	store := NewInMemoryPodStore()
	store.Put(pod.PodKey, pod)
	h := &RunnerMessageHandler{podStore: store}

	if err := h.OnSendPrompt(&runnerv1.SendPromptCommand{PodKey: pod.PodKey, Prompt: "hello"}); err != nil {
		t.Fatalf("OnSendPrompt error: %v", err)
	}

	io.mu.Lock()
	defer io.mu.Unlock()
	if len(io.inputs) != 2 {
		t.Fatalf("PTY mode must split into 2 writes (text + Enter); got %d: %v", len(io.inputs), io.inputs)
	}
	if io.inputs[0] != "hello" {
		t.Errorf("inputs[0] = %q, want %q", io.inputs[0], "hello")
	}
	if io.inputs[1] != "\r" {
		t.Errorf("inputs[1] = %q, want \"\\r\"", io.inputs[1])
	}
}

func TestOnSendPrompt_ACP_NoEnter(t *testing.T) {
	io := &sendPromptMockIO{mode: InteractionModeACP}
	pod := &Pod{PodKey: "acp-pod", InteractionMode: InteractionModeACP, IO: io}

	store := NewInMemoryPodStore()
	store.Put(pod.PodKey, pod)
	h := &RunnerMessageHandler{podStore: store}

	if err := h.OnSendPrompt(&runnerv1.SendPromptCommand{PodKey: pod.PodKey, Prompt: "hello"}); err != nil {
		t.Fatalf("OnSendPrompt error: %v", err)
	}

	io.mu.Lock()
	defer io.mu.Unlock()
	if len(io.inputs) != 1 {
		t.Fatalf("ACP mode must submit via SendPrompt only (1 write); got %d: %v", len(io.inputs), io.inputs)
	}
	if io.inputs[0] != "hello" {
		t.Errorf("inputs[0] = %q, want %q", io.inputs[0], "hello")
	}
}
