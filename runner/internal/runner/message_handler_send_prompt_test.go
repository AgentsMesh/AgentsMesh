package runner

import (
	"sync"
	"testing"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// sendPromptMockIO records SendInput and SendKeys calls (with timestamps so
// tests can assert ordering and the post-text submission gap).
type sendPromptMockIO struct {
	stubPodIOZero
	mu        sync.Mutex
	mode      string
	inputs    []timedCall
	keys      []timedCall
	inputErr  error
	keysErr   error
	hasTermAx bool
}

type timedCall struct {
	payload string
	at      time.Time
}

func (m *sendPromptMockIO) Mode() string { return m.mode }

func (m *sendPromptMockIO) SendInput(text string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.inputs = append(m.inputs, timedCall{payload: text, at: time.Now()})
	return m.inputErr
}

// ptyTerminalMock satisfies TerminalAccess on top of sendPromptMockIO so the
// PTY branch of OnSendPrompt resolves through SendKeys (the "press Enter"
// path) instead of SendInput (the "raw bytes" path).
type ptyTerminalMock struct {
	*sendPromptMockIO
}

func (p *ptyTerminalMock) SendKeys(keys []string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, k := range keys {
		p.keys = append(p.keys, timedCall{payload: k, at: time.Now()})
	}
	return p.keysErr
}

func (p *ptyTerminalMock) Resize(int, int) (bool, error)             { return true, nil }
func (p *ptyTerminalMock) CursorPosition() (int, int)                { return 0, 0 }
func (p *ptyTerminalMock) GetScreenSnapshot() string                 { return "" }
func (p *ptyTerminalMock) Redraw() error                             { return nil }
func (p *ptyTerminalMock) WriteOutput([]byte)                        {}

// stubPodIOZero satisfies PodIO with no-ops so each test mock only overrides
// the methods it cares about.
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

func TestOnSendPrompt_PTY_PressesEnterViaSendKeys(t *testing.T) {
	base := &sendPromptMockIO{mode: InteractionModePTY}
	io := &ptyTerminalMock{sendPromptMockIO: base}
	pod := &Pod{PodKey: "pty-pod", InteractionMode: InteractionModePTY, IO: io}

	store := NewInMemoryPodStore()
	store.Put(pod.PodKey, pod)
	h := &RunnerMessageHandler{podStore: store}

	if err := h.OnSendPrompt(&runnerv1.SendPromptCommand{PodKey: pod.PodKey, Prompt: "hello"}); err != nil {
		t.Fatalf("OnSendPrompt error: %v", err)
	}

	base.mu.Lock()
	defer base.mu.Unlock()
	if len(base.inputs) != 1 {
		t.Fatalf("expected 1 SendInput for the body; got %d: %v", len(base.inputs), base.inputs)
	}
	if base.inputs[0].payload != "hello" {
		t.Errorf("body payload = %q, want %q", base.inputs[0].payload, "hello")
	}
	if len(base.keys) != 1 || base.keys[0].payload != "enter" {
		t.Fatalf("expected one SendKeys([\"enter\"]); got %v", base.keys)
	}
}

func TestOnSendPrompt_PTY_GapBetweenBodyAndEnter(t *testing.T) {
	base := &sendPromptMockIO{mode: InteractionModePTY}
	io := &ptyTerminalMock{sendPromptMockIO: base}
	pod := &Pod{PodKey: "pty-pod", InteractionMode: InteractionModePTY, IO: io}

	store := NewInMemoryPodStore()
	store.Put(pod.PodKey, pod)
	h := &RunnerMessageHandler{podStore: store}

	if err := h.OnSendPrompt(&runnerv1.SendPromptCommand{PodKey: pod.PodKey, Prompt: "hello"}); err != nil {
		t.Fatalf("OnSendPrompt error: %v", err)
	}

	base.mu.Lock()
	defer base.mu.Unlock()
	gap := base.keys[0].at.Sub(base.inputs[0].at)
	// Allow a small floor under ptySubmitGap to absorb scheduler jitter, but
	// require enough separation that the TUI's read loop can tick. Without
	// this gap the trailing Enter is folded into the body paste.
	const minGap = 50 * time.Millisecond
	if gap < minGap {
		t.Fatalf("gap between body and Enter = %v, want >= %v (TUI read-loop separation)", gap, minGap)
	}
}

func TestOnSendPrompt_ACP_NoEnterKey(t *testing.T) {
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
		t.Fatalf("ACP must submit via the ACP RPC only (1 SendInput); got %d: %v", len(io.inputs), io.inputs)
	}
	if io.inputs[0].payload != "hello" {
		t.Errorf("body payload = %q, want %q", io.inputs[0].payload, "hello")
	}
	if len(io.keys) != 0 {
		t.Fatalf("ACP must not press Enter via SendKeys; got %v", io.keys)
	}
}
