package runner

import (
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/poddaemon"
	"github.com/anthropics/agentsmesh/runner/internal/relay"
	"github.com/anthropics/agentsmesh/runner/internal/terminal"
)

type perpetualCoverageIO struct {
	mu             sync.Mutex
	startErr       error
	pid            int
	starts         int
	stops          int
	teardowns      int
	exitHandler    func(int)
	ioErrorHandler func(error)
	onStart        func()
	onTeardown     func()
}

func (i *perpetualCoverageIO) Mode() string                              { return InteractionModePTY }
func (i *perpetualCoverageIO) SendInput(string) error                    { return nil }
func (i *perpetualCoverageIO) GetSnapshot(int) (string, error)           { return "", nil }
func (i *perpetualCoverageIO) GetAgentStatus() string                    { return "idle" }
func (i *perpetualCoverageIO) SubscribeStateChange(string, func(string)) {}
func (i *perpetualCoverageIO) UnsubscribeStateChange(string)             {}
func (i *perpetualCoverageIO) GetPID() int                               { return i.pid }
func (i *perpetualCoverageIO) Detach()                                   {}

func (i *perpetualCoverageIO) Start() error {
	i.mu.Lock()
	i.starts++
	err := i.startErr
	hook := i.onStart
	i.mu.Unlock()
	if hook != nil {
		hook()
	}
	return err
}

func (i *perpetualCoverageIO) Stop() {
	i.mu.Lock()
	i.stops++
	i.mu.Unlock()
}

func (i *perpetualCoverageIO) Teardown() string {
	i.mu.Lock()
	i.teardowns++
	hook := i.onTeardown
	i.mu.Unlock()
	if hook != nil {
		hook()
	}
	return ""
}

func (i *perpetualCoverageIO) SetExitHandler(handler func(int)) {
	i.mu.Lock()
	i.exitHandler = handler
	i.mu.Unlock()
}

func (i *perpetualCoverageIO) SetIOErrorHandler(handler func(error)) {
	i.mu.Lock()
	i.ioErrorHandler = handler
	i.mu.Unlock()
}

type perpetualCoverageRelay struct{}

func (*perpetualCoverageRelay) SetupHandlers(relay.RelayClient, RelayInboundGuard) {}
func (*perpetualCoverageRelay) SendSnapshot(relay.RelayClient)                     {}
func (*perpetualCoverageRelay) RecoverSnapshot(relay.RelayClient)                  {}
func (*perpetualCoverageRelay) OnRelayConnected(relay.RelayClient)                 {}
func (*perpetualCoverageRelay) OnRelayDisconnected(relay.RelayClient)              {}
func (*perpetualCoverageRelay) BroadcastEvent(relay.RelayClient, byte, []byte)     {}

type perpetualCoveragePTY struct {
	mu      sync.Mutex
	killErr error
	kills   int
	closes  int
}

func (*perpetualCoveragePTY) Read([]byte) (int, error)        { return 0, io.EOF }
func (*perpetualCoveragePTY) Write(data []byte) (int, error)  { return len(data), nil }
func (p *perpetualCoveragePTY) Close() error                  { p.mu.Lock(); p.closes++; p.mu.Unlock(); return nil }
func (*perpetualCoveragePTY) Resize(int, int) error           { return nil }
func (*perpetualCoveragePTY) GetSize() (int, int, error)      { return 80, 24, nil }
func (*perpetualCoveragePTY) Pid() int                        { return 77 }
func (*perpetualCoveragePTY) SetReadDeadline(time.Time) error { return nil }
func (*perpetualCoveragePTY) Wait() (int, error)              { return 0, nil }
func (p *perpetualCoveragePTY) Kill() error {
	p.mu.Lock()
	p.kills++
	err := p.killErr
	p.mu.Unlock()
	return err
}
func (*perpetualCoveragePTY) GracefulStop() error { return nil }

func newPerpetualCoveragePod(key string, io PodIO) *Pod {
	pod := &Pod{
		PodKey:    key,
		Agent:     "coverage-agent",
		Perpetual: true,
		Status:    PodStatusRunning,
		StartedAt: time.Now(),
	}
	if io != nil {
		pod.installRuntime(PodRuntime{IO: io, Relay: &perpetualCoverageRelay{}})
	}
	return pod
}

func TestPreparePerpetualRestartOwnershipCoverage(t *testing.T) {
	runner, _ := NewTestRunner(t)
	handler := runner.messageHandler

	if transition, handled := handler.preparePerpetualRestart(
		newPerpetualCoveragePod("forced-stop", nil), 0, true,
	); transition != 0 || handled {
		t.Fatalf("forced stop = (%d, %v), want (0, false)", transition, handled)
	}
	if transition, handled := handler.preparePerpetualRestart(
		newPerpetualCoveragePod("dirty-exit", nil), 7, false,
	); transition != 0 || handled {
		t.Fatalf("dirty exit = (%d, %v), want (0, false)", transition, handled)
	}

	absent := newPerpetualCoveragePod("absent", nil)
	if transition, handled := handler.preparePerpetualRestart(absent, 0, false); transition != 0 || handled {
		t.Fatalf("absent pod = (%d, %v), want (0, false)", transition, handled)
	}

	replaced := newPerpetualCoveragePod("replaced", nil)
	runner.podStore.Put(replaced.PodKey, &Pod{PodKey: replaced.PodKey})
	if transition, handled := handler.preparePerpetualRestart(replaced, 0, false); transition != 0 || handled {
		t.Fatalf("replaced pod = (%d, %v), want (0, false)", transition, handled)
	}

	nonPerpetual := newPerpetualCoveragePod("non-perpetual", nil)
	nonPerpetual.Perpetual = false
	runner.podStore.Put(nonPerpetual.PodKey, nonPerpetual)
	if transition, handled := handler.preparePerpetualRestart(nonPerpetual, 0, false); transition != 0 || handled {
		t.Fatalf("non-perpetual pod = (%d, %v), want (0, false)", transition, handled)
	}

	stopped := newPerpetualCoveragePod("stopped", nil)
	stopped.SetStatus(PodStatusStopped)
	runner.podStore.Put(stopped.PodKey, stopped)
	if transition, handled := handler.preparePerpetualRestart(stopped, 0, false); transition != 0 || !handled {
		t.Fatalf("stopped pod = (%d, %v), want (0, true)", transition, handled)
	}

	oldIO := &perpetualCoverageIO{}
	owned := newPerpetualCoveragePod("owned", oldIO)
	owned.SetPTYError("stale read failure")
	runner.podStore.Put(owned.PodKey, owned)
	transition, handled := handler.preparePerpetualRestart(owned, 0, false)
	if transition == 0 || !handled {
		t.Fatalf("owned pod = (%d, %v), want non-zero transition", transition, handled)
	}
	if owned.RestartCount != 1 || oldIO.teardowns != 1 {
		t.Fatalf("restart count/teardowns = %d/%d, want 1/1", owned.RestartCount, oldIO.teardowns)
	}
	if owned.GetPTYError() != "" {
		t.Fatalf("PTY error survived retirement: %q", owned.GetPTYError())
	}
	if runtime := owned.installedRuntime(); runtime.IO != nil || runtime.Relay != nil {
		t.Fatalf("retired runtime still installed: %+v", runtime)
	}
	if !owned.RelayRuntimeTransitionCurrent(transition) {
		t.Fatal("replacement transition was not kept blocked and current")
	}
}

func TestPreparePerpetualRestartRejectsSupersededClear(t *testing.T) {
	runner, _ := NewTestRunner(t)
	handler := runner.messageHandler
	pod := newPerpetualCoveragePod("superseded-clear", nil)
	io := &perpetualCoverageIO{}
	pod.installRuntime(PodRuntime{IO: io, Relay: &perpetualCoverageRelay{}})
	io.onTeardown = func() { pod.BeginRelayRuntimeTransition() }
	runner.podStore.Put(pod.PodKey, pod)

	transition, handled := handler.preparePerpetualRestart(pod, 0, false)
	if transition != 0 || !handled {
		t.Fatalf("superseded clear = (%d, %v), want (0, true)", transition, handled)
	}
}

func TestDiscardPerpetualRuntimeCoverage(t *testing.T) {
	t.Run("not started kills and closes", func(t *testing.T) {
		io := &perpetualCoverageIO{}
		pty := &perpetualCoveragePTY{killErr: errors.New("already exited")}
		discardPerpetualRuntime(PodRuntime{IO: io}, pty, false)
		if io.stops != 0 || io.teardowns != 1 || pty.kills != 1 || pty.closes != 1 {
			t.Fatalf("stop/teardown/kill/close = %d/%d/%d/%d", io.stops, io.teardowns, pty.kills, pty.closes)
		}
	})

	t.Run("started stops and tears down without killing", func(t *testing.T) {
		io := &perpetualCoverageIO{}
		pty := &perpetualCoveragePTY{}
		discardPerpetualRuntime(PodRuntime{IO: io}, pty, true)
		if io.stops != 1 || io.teardowns != 1 || pty.kills != 0 || pty.closes != 0 {
			t.Fatalf("stop/teardown/kill/close = %d/%d/%d/%d", io.stops, io.teardowns, pty.kills, pty.closes)
		}
	})
}

func TestRestartPerpetualPodDependencyFailures(t *testing.T) {
	t.Run("missing manager finalizes pod", func(t *testing.T) {
		runner, conn := NewTestRunner(t)
		handler := runner.messageHandler
		pod := newPerpetualCoveragePod("missing-manager", &perpetualCoverageIO{})
		runner.podStore.Put(pod.PodKey, pod)
		transition, handled := handler.preparePerpetualRestart(pod, 0, false)
		if transition == 0 || !handled {
			t.Fatalf("prepare = (%d, %v)", transition, handled)
		}

		handler.restartPerpetualPod(pod, 0, transition)
		if _, ok := runner.podStore.Get(pod.PodKey); ok {
			t.Fatal("pod survived a restart without PodDaemonManager")
		}
		if !hasPerpetualCoverageEvent(conn.GetEvents(), client.MsgTypePodTerminated) {
			t.Fatal("missing manager did not emit pod termination")
		}
	})

	t.Run("manager create error finalizes pod", func(t *testing.T) {
		runner, conn := NewTestRunner(t)
		manager, err := poddaemon.NewPodDaemonManager(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		runner.podDaemonManager = manager
		handler := runner.messageHandler
		pod := newPerpetualCoveragePod("manager-error", &perpetualCoverageIO{})
		pod.LaunchCommand = "ignored"
		// Empty SandboxPath makes CreateSession fail before spawning anything.
		runner.podStore.Put(pod.PodKey, pod)
		transition, handled := handler.preparePerpetualRestart(pod, 0, false)
		if transition == 0 || !handled {
			t.Fatalf("prepare = (%d, %v)", transition, handled)
		}

		handler.restartPerpetualPod(pod, 0, transition)
		if _, ok := runner.podStore.Get(pod.PodKey); ok {
			t.Fatal("pod survived CreateSession failure")
		}
		if !hasPerpetualCoverageEvent(conn.GetEvents(), client.MsgTypePodTerminated) {
			t.Fatal("CreateSession failure did not emit pod termination")
		}
	})
}

func newPerpetualFactoryHarness(
	t *testing.T,
	podKey string,
) (*Runner, *RunnerMessageHandler, *Pod, RelayRuntimeTransition, *perpetualCoveragePTY) {
	t.Helper()
	runner, _ := NewTestRunner(t)
	manager, err := poddaemon.NewPodDaemonManager(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	runner.podDaemonManager = manager
	handler := runner.messageHandler
	process := &perpetualCoveragePTY{}
	handler.perpetualSessionFactory = func(
		*poddaemon.PodDaemonManager,
		poddaemon.CreateOpts,
	) (terminal.PtyProcess, error) {
		return process, nil
	}
	pod := newPerpetualCoveragePod(podKey, &perpetualCoverageIO{})
	pod.LaunchCommand = "coverage-command"
	pod.SandboxPath = t.TempDir()
	runner.podStore.Put(pod.PodKey, pod)
	transition, handled := handler.preparePerpetualRestart(pod, 0, false)
	if transition == 0 || !handled {
		t.Fatalf("prepare = (%d, %v)", transition, handled)
	}
	return runner, handler, pod, transition, process
}

func hasPerpetualCoverageEvent(events []client.EventCall, eventType client.MessageType) bool {
	for _, event := range events {
		if event.Type == eventType {
			return true
		}
	}
	return false
}
