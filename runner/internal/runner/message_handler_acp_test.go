package runner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/config"
)

// acpMockPodIO is a minimal PodIO for testing ACP-related code paths.
type acpMockPodIO struct {
	sendInputCalled     bool
	sendInputText       string
	respondPermCalled   bool
	respondPermReqID    string
	respondPermApproved bool
	cancelSessionCalled bool
}

func (m *acpMockPodIO) Mode() string                              { return "acp" }
func (m *acpMockPodIO) GetSnapshot(int) (string, error)           { return "", nil }
func (m *acpMockPodIO) GetAgentStatus() string                    { return "idle" }
func (m *acpMockPodIO) SubscribeStateChange(string, func(string)) {}
func (m *acpMockPodIO) UnsubscribeStateChange(string)             {}
func (m *acpMockPodIO) SendKeys([]string) error                   { return nil }
func (m *acpMockPodIO) Resize(int, int) (bool, error)             { return false, nil }
func (m *acpMockPodIO) GetPID() int                               { return 0 }
func (m *acpMockPodIO) CursorPosition() (int, int)                { return 0, 0 }
func (m *acpMockPodIO) GetScreenSnapshot() string                 { return "" }
func (m *acpMockPodIO) Stop()                                     {}
func (m *acpMockPodIO) Teardown() string                          { return "" }
func (m *acpMockPodIO) SetExitHandler(func(int))                  {}
func (m *acpMockPodIO) SetIOErrorHandler(func(error))             {}
func (m *acpMockPodIO) Detach()                                   {}
func (m *acpMockPodIO) WriteOutput([]byte)                        {}
func (m *acpMockPodIO) Start() error                              { return nil }

func (m *acpMockPodIO) SendInput(text string) error {
	m.sendInputCalled = true
	m.sendInputText = text
	return nil
}

func (m *acpMockPodIO) RespondToPermission(reqID string, approved bool) error {
	m.respondPermCalled = true
	m.respondPermReqID = reqID
	m.respondPermApproved = approved
	return nil
}

func (m *acpMockPodIO) CancelSession() error {
	m.cancelSessionCalled = true
	return nil
}

// --- handleACPExit ---

func TestHandleACPExit_CleanupPodExit(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()
	handler := NewRunnerMessageHandler(&Runner{cfg: &config.Config{}}, store, mockConn)

	pod := &Pod{PodKey: "acp-exit-1", Status: PodStatusRunning}
	store.Put("acp-exit-1", pod)

	handler.handleACPExit("acp-exit-1", 0)

	// Pod should be removed from store
	if _, ok := store.Get("acp-exit-1"); ok {
		t.Error("pod should be removed after exit")
	}
}

func TestMarshalAcpRelayEventDefensivePayloads(t *testing.T) {
	if payload := marshalAcpRelayEvent("invalid", "session", make(chan int)); payload != nil {
		t.Fatalf("unsupported payload encoded as %q", payload)
	}
	payload := marshalAcpRelayEvent("scalar", "session", "value")
	if got := string(payload); !strings.Contains(got, `"type":"scalar"`) || !strings.Contains(got, `"sessionId":"session"`) {
		t.Fatalf("scalar event envelope = %q", payload)
	}

	pod := &Pod{PodKey: "marshal-failure"}
	sendAcpViaRelay(pod, "invalid", "session", make(chan int))
}

func TestWireAndStartACPPodRejectsLostOwnership(t *testing.T) {
	r, _ := NewTestRunner(t)
	handler := r.messageHandler
	pod := &Pod{PodKey: "lost", LaunchCommand: "agent"}

	err := handler.wireAndStartACPPod(pod, &runnerv1.CreatePodCommand{PodKey: pod.PodKey}, 80, 24)
	if err == nil || !strings.Contains(err.Error(), "terminated before ACP start") {
		t.Fatalf("wireAndStartACPPod error = %v, want lost ownership", err)
	}
}

func TestWireAndStartACPPodCleansFailedRuntime(t *testing.T) {
	r, _ := NewTestRunner(t)
	handler := r.messageHandler
	sandbox := filepath.Join(t.TempDir(), "sandbox")
	if err := os.MkdirAll(sandbox, 0700); err != nil {
		t.Fatal(err)
	}
	pod := &Pod{
		PodKey: "start-failure", LaunchCommand: filepath.Join(t.TempDir(), "missing-agent"),
		WorkDir: t.TempDir(), SandboxPath: sandbox,
	}
	r.podStore.Put(pod.PodKey, pod)

	err := handler.wireAndStartACPPod(pod, &runnerv1.CreatePodCommand{PodKey: pod.PodKey}, 80, 24)
	if err == nil || !strings.Contains(err.Error(), "failed to start ACP agent") {
		t.Fatalf("wireAndStartACPPod error = %v, want process start failure", err)
	}
	if _, ok := r.podStore.Get(pod.PodKey); ok {
		t.Fatal("failed ACP runtime remained in pod store")
	}
	if _, err := os.Stat(sandbox); !os.IsNotExist(err) {
		t.Fatalf("failed ACP sandbox was not removed: %v", err)
	}
}

func TestWireAndStartACPPodPublishesRunnableRelayCommand(t *testing.T) {
	r, _ := NewTestRunner(t)
	pod := &Pod{
		PodKey:        "wire-success",
		LaunchCommand: os.Args[0],
		LaunchEnv:     append(os.Environ(), acpWireTestAgentEnv+"=1"),
		WorkDir:       t.TempDir(),
	}
	r.podStore.Put(pod.PodKey, pod)

	err := r.messageHandler.wireAndStartACPPod(
		pod,
		&runnerv1.CreatePodCommand{PodKey: pod.PodKey},
		80,
		24,
	)
	if err != nil {
		t.Fatalf("wireAndStartACPPod: %v", err)
	}
	runtime := pod.installedRuntime()
	defer func() {
		runtime.IO.Stop()
		runtime.IO.Teardown()
		r.podStore.DeleteIf(pod.PodKey, pod)
	}()
	if pod.GetStatus() != PodStatusRunning || !runtime.valid() {
		t.Fatalf("published ACP runtime = %+v, status=%s", runtime, pod.GetStatus())
	}
	modeRelay, ok := runtime.Relay.(*ACPPodRelay)
	if !ok {
		t.Fatalf("runtime relay type = %T, want *ACPPodRelay", runtime.Relay)
	}
	commandIO := &acpMockPodIO{}
	modeRelay.onCommand(
		RelayInboundContext{IO: commandIO, Relay: modeRelay},
		[]byte(`{"type":"prompt","prompt":"from-relay"}`),
	)
	if !commandIO.sendInputCalled || commandIO.sendInputText != "from-relay" {
		t.Fatalf("relay command was not bound to the published handler: %+v", commandIO)
	}
}
