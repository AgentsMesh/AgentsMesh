package runner

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type mcpCoveragePodIO struct {
	stubPodIO
	snapshot       string
	snapshotErr    error
	agentStatus    string
	pid            int
	mode           string
	input          []string
	inputErr       error
	subscribedID   string
	unsubscribedID string
	callback       func(string)
}

func (s *mcpCoveragePodIO) Mode() string {
	if s.mode != "" {
		return s.mode
	}
	return "acp"
}

func (s *mcpCoveragePodIO) GetSnapshot(int) (string, error) {
	return s.snapshot, s.snapshotErr
}

func (s *mcpCoveragePodIO) GetAgentStatus() string { return s.agentStatus }
func (s *mcpCoveragePodIO) GetPID() int            { return s.pid }

func (s *mcpCoveragePodIO) SendInput(text string) error {
	s.input = append(s.input, text)
	return s.inputErr
}

func (s *mcpCoveragePodIO) SubscribeStateChange(id string, callback func(string)) {
	s.subscribedID, s.callback = id, callback
}

func (s *mcpCoveragePodIO) UnsubscribeStateChange(id string) {
	s.unsubscribedID = id
}

type mcpCoverageTerminalIO struct {
	*mcpCoveragePodIO
	keys    []string
	keysErr error
}

func (s *mcpCoverageTerminalIO) SendKeys(keys []string) error {
	s.keys = append([]string(nil), keys...)
	return s.keysErr
}

func (s *mcpCoverageTerminalIO) CursorPosition() (int, int) { return 3, 4 }
func (s *mcpCoverageTerminalIO) GetScreenSnapshot() string  { return "screen" }
func (s *mcpCoverageTerminalIO) WriteOutput([]byte)         {}
func (s *mcpCoverageTerminalIO) Resize(int, int) (bool, error) {
	return true, nil
}

func newMCPTestRunner(store PodStore) *Runner {
	return &Runner{podStore: store}
}

func TestRunnerMCPLocalPodAccess(t *testing.T) {
	store := NewInMemoryPodStore()
	runner := newMCPTestRunner(store)

	agentStatus, podStatus, pid, found := runner.GetPodStatus("missing")
	require.False(t, found)
	require.Equal(t, "idle", agentStatus)
	require.Equal(t, "not_found", podStatus)
	require.Zero(t, pid)

	store.Put("nil", nil)
	_, _, _, found = runner.GetPodStatus("nil")
	require.False(t, found)

	io := &mcpCoveragePodIO{
		snapshot:    "latest screen",
		agentStatus: "working",
		pid:         4242,
	}
	pod := &Pod{PodKey: "active", Status: PodStatusRunning, IO: io}
	store.Put(pod.PodKey, pod)

	agentStatus, podStatus, pid, found = runner.GetPodStatus(pod.PodKey)
	require.True(t, found)
	require.Equal(t, "working", agentStatus)
	require.Equal(t, PodStatusRunning, podStatus)
	require.Equal(t, 4242, pid)

	snapshot, err := runner.GetPodSnapshot(pod.PodKey, 12)
	require.NoError(t, err)
	require.Equal(t, "latest screen", snapshot)

	io.snapshotErr = errors.New("snapshot failed")
	_, err = runner.GetPodSnapshot(pod.PodKey, 12)
	require.ErrorContains(t, err, "snapshot failed")

	_, err = runner.GetPodSnapshot("missing", 12)
	require.ErrorContains(t, err, "pod not found")
	_, err = runner.GetPodSnapshot("nil", 12)
	require.ErrorContains(t, err, "pod not found")

	pod.SetStatus(PodStatusStopped)
	_, err = runner.GetPodSnapshot(pod.PodKey, 12)
	require.ErrorContains(t, err, "pod IO not available")
}

func TestRunnerMCPSendInputBranches(t *testing.T) {
	store := NewInMemoryPodStore()
	runner := newMCPTestRunner(store)

	require.ErrorContains(t, runner.SendPodInput("missing", "text", nil), "pod not found")
	store.Put("nil", nil)
	require.ErrorContains(t, runner.SendPodInput("nil", "text", nil), "pod not found")

	base := &mcpCoveragePodIO{mode: "acp"}
	pod := &Pod{PodKey: "active", Status: PodStatusRunning, IO: base}
	store.Put(pod.PodKey, pod)

	require.NoError(t, runner.SendPodInput(pod.PodKey, "hello", nil))
	require.Equal(t, []string{"hello"}, base.input)
	require.ErrorContains(t, runner.SendPodInput(pod.PodKey, "", []string{"enter"}), "special keys not supported")

	base.inputErr = errors.New("write failed")
	require.ErrorContains(t, runner.SendPodInput(pod.PodKey, "bad", []string{"enter"}), "failed to send text")
	require.Empty(t, base.input[2:], "keys must not run after a text write failure")

	terminalBase := &mcpCoveragePodIO{mode: "pty"}
	terminalIO := &mcpCoverageTerminalIO{mcpCoveragePodIO: terminalBase}
	pod.IO = terminalIO
	require.NoError(t, runner.SendPodInput(pod.PodKey, "", []string{"ctrl-c", "enter"}))
	require.Equal(t, []string{"ctrl-c", "enter"}, terminalIO.keys)

	terminalIO.keysErr = errors.New("key failed")
	require.ErrorContains(t, runner.SendPodInput(pod.PodKey, "", []string{"escape"}), "key failed")

	pod.SetStatus(PodStatusStopped)
	require.ErrorContains(t, runner.SendPodInput(pod.PodKey, "text", nil), "pod IO not available")
}

func TestPodControllerDelegatesToCurrentIO(t *testing.T) {
	store := NewInMemoryPodStore()
	runner := newMCPTestRunner(store)
	io := &mcpCoveragePodIO{agentStatus: "waiting"}
	pod := &Pod{
		PodKey:      "controller-pod",
		SandboxPath: "/tmp/controller-workdir",
		Status:      PodStatusRunning,
		IO:          io,
	}
	store.Put(pod.PodKey, pod)
	controller := NewPodController(pod, runner)

	require.Equal(t, pod.PodKey, controller.GetPodKey())
	require.Equal(t, pod.SandboxPath, controller.GetWorkDir())
	require.Equal(t, "waiting", controller.GetAgentStatus())
	require.NoError(t, controller.SendInput("prompt"))
	require.Equal(t, []string{"prompt\n"}, io.input)

	callback := func(string) {}
	controller.SubscribeStateChange("watch", callback)
	controller.UnsubscribeStateChange("watch")
	require.Equal(t, "watch", io.subscribedID)
	require.NotNil(t, io.callback)
	require.Equal(t, "watch", io.unsubscribedID)

	io.inputErr = errors.New("send failed")
	require.ErrorContains(t, controller.SendInput("bad"), "send failed")
	pod.SetStatus(PodStatusStopped)
	require.ErrorContains(t, controller.SendInput("stopped"), "pod IO not available")
	controller.SubscribeStateChange("ignored", callback)
	controller.UnsubscribeStateChange("ignored")
	require.Equal(t, "watch", io.subscribedID)
	require.Equal(t, "watch", io.unsubscribedID)
}
