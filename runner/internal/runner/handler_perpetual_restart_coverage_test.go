package runner

import (
	"errors"
	"os"
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/poddaemon"
	"github.com/anthropics/agentsmesh/runner/internal/terminal"
	"github.com/anthropics/agentsmesh/runner/internal/testutil"
)

func TestRestartPerpetualPodRealDaemonCommit(t *testing.T) {
	runner, conn := NewTestRunner(t)
	manager, err := poddaemon.NewPodDaemonManager(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	runner.podDaemonManager = manager
	handler := runner.messageHandler
	command, args := testutil.SleepCommand(30)
	pod := newPerpetualCoveragePod("real-daemon-restart", &perpetualCoverageIO{})
	pod.Agent = "coverage-agent"
	pod.LaunchCommand = command
	pod.LaunchArgs = args
	pod.LaunchEnv = os.Environ()
	pod.WorkDir = t.TempDir()
	pod.SandboxPath = t.TempDir()
	pod.RepositoryURL = "https://example.invalid/restart.git"
	pod.Branch = "restart-branch"
	pod.TicketSlug = "RESTART-1"
	runner.podStore.Put(pod.PodKey, pod)
	transition, handled := handler.preparePerpetualRestart(pod, 0, false)
	if transition == 0 || !handled {
		t.Fatalf("prepare = (%d, %v)", transition, handled)
	}
	// Directly constructed handlers retain a nil-safe path to the same real
	// session/runtime implementations wired by NewRunnerMessageHandler.
	handler.perpetualSessionFactory = nil
	handler.perpetualRuntimeFactory = nil

	handler.restartPerpetualPod(pod, 0, transition)
	t.Cleanup(func() {
		if current, ok := runner.podStore.Get(pod.PodKey); ok && current == pod {
			pod.Perpetual = false
			handler.cleanupPodExit(pod.PodKey, -1, true)
		}
		_ = poddaemon.DeleteState(pod.SandboxPath)
	})
	runtime := pod.installedRuntime()
	if !runtime.valid() || runtime.IO.GetPID() <= 0 {
		t.Fatalf("replacement runtime not committed: %+v", runtime)
	}
	if pod.GetStatus() != PodStatusRunning || pod.RestartCount != 1 {
		t.Fatalf("restart status/count = %s/%d", pod.GetStatus(), pod.RestartCount)
	}
	if pod.RelayRuntimeTransitionCurrent(transition) {
		t.Fatal("successful replacement left relay runtime blocked")
	}
	if !hasPerpetualCoverageEvent(conn.GetEvents(), client.MessageType("pod_restarting")) {
		t.Fatal("successful replacement did not emit pod_restarting")
	}

	pod.Perpetual = false
	handler.cleanupPodExit(pod.PodKey, -1, true)
	if _, ok := runner.podStore.Get(pod.PodKey); ok {
		t.Fatal("replacement pod survived explicit cleanup")
	}
}

func TestRestartPerpetualPodRejectsStaleOwnerAfterCreate(t *testing.T) {
	runner, _ := NewTestRunner(t)
	manager, err := poddaemon.NewPodDaemonManager(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	runner.podDaemonManager = manager
	handler := runner.messageHandler
	command, args := testutil.SleepCommand(30)
	pod := newPerpetualCoveragePod("stale-after-create", nil)
	pod.LaunchCommand = command
	pod.LaunchArgs = args
	pod.LaunchEnv = os.Environ()
	pod.WorkDir = t.TempDir()
	pod.SandboxPath = t.TempDir()
	runner.podStore.Put(pod.PodKey, pod)
	stale := pod.BeginRelayRuntimeTransition()
	pod.BeginRelayRuntimeTransition()

	handler.restartPerpetualPod(pod, 0, stale)
	t.Cleanup(func() { _ = poddaemon.DeleteState(pod.SandboxPath) })
	if runtime := pod.installedRuntime(); runtime.IO != nil || runtime.Relay != nil {
		t.Fatalf("stale owner installed replacement runtime: %+v", runtime)
	}
	if err := poddaemon.DeleteState(pod.SandboxPath); err != nil && !os.IsNotExist(err) {
		t.Fatalf("delete stale daemon state: %v", err)
	}
}

func TestPerpetualFactoriesDefaultToProductionImplementations(t *testing.T) {
	runner, _ := NewTestRunner(t)
	handler := runner.messageHandler
	if handler.perpetualSessionFactory == nil || handler.perpetualRuntimeFactory == nil {
		t.Fatal("NewRunnerMessageHandler did not wire perpetual production factories")
	}
	manager, err := poddaemon.NewPodDaemonManager(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := handler.perpetualSessionFactory(manager, poddaemon.CreateOpts{}); err == nil {
		t.Fatal("default session factory did not invoke PodDaemonManager.CreateSession")
	}
	if _, err := handler.perpetualRuntimeFactory(
		&Pod{PodKey: "default-runtime"}, &perpetualCoveragePTY{}, 80, 24,
	); err == nil {
		t.Fatal("default runtime factory did not invoke buildPerpetualPTYRuntime")
	}
}

func TestRestartPerpetualPodInjectedFailureCoverage(t *testing.T) {
	t.Run("runtime build error finalizes current owner", func(t *testing.T) {
		runner, handler, pod, transition, process := newPerpetualFactoryHarness(t, "build-error")
		handler.perpetualRuntimeFactory = func(*Pod, terminal.PtyProcess, int, int) (PodRuntime, error) {
			return PodRuntime{}, errors.New("injected build failure")
		}

		handler.restartPerpetualPod(pod, 0, transition)
		if _, ok := runner.podStore.Get(pod.PodKey); ok {
			t.Fatal("runtime build failure retained pod ownership")
		}
		if process.closes != 1 {
			t.Fatalf("runtime build failure closes = %d, want 1", process.closes)
		}
	})

	t.Run("runtime build error preserves a superseding owner", func(t *testing.T) {
		runner, handler, pod, transition, process := newPerpetualFactoryHarness(t, "build-error-superseded")
		handler.perpetualRuntimeFactory = func(*Pod, terminal.PtyProcess, int, int) (PodRuntime, error) {
			pod.BeginRelayRuntimeTransition()
			return PodRuntime{}, errors.New("injected build failure after supersede")
		}

		handler.restartPerpetualPod(pod, 0, transition)
		if current, ok := runner.podStore.Get(pod.PodKey); !ok || current != pod {
			t.Fatal("superseded build failure removed the current pod")
		}
		if process.closes != 1 {
			t.Fatalf("superseded build failure closes = %d, want 1", process.closes)
		}
		runner.podStore.DeleteIf(pod.PodKey, pod)
	})

	t.Run("ownership loss after build discards candidate", func(t *testing.T) {
		runner, handler, pod, transition, process := newPerpetualFactoryHarness(t, "ownership-after-build")
		candidateIO := &perpetualCoverageIO{}
		replacement := &Pod{PodKey: pod.PodKey, Status: PodStatusRunning}
		handler.perpetualRuntimeFactory = func(*Pod, terminal.PtyProcess, int, int) (PodRuntime, error) {
			runner.podStore.Put(pod.PodKey, replacement)
			return PodRuntime{IO: candidateIO, Relay: &perpetualCoverageRelay{}}, nil
		}

		handler.restartPerpetualPod(pod, 0, transition)
		if current, _ := runner.podStore.Get(pod.PodKey); current != replacement {
			t.Fatal("stale restart replaced the superseding store owner")
		}
		if candidateIO.teardowns != 1 || process.kills != 1 || process.closes != 1 {
			t.Fatalf("discard calls teardown/kill/close = %d/%d/%d",
				candidateIO.teardowns, process.kills, process.closes)
		}
		runner.podStore.DeleteIf(pod.PodKey, replacement)
	})

	t.Run("runtime start error clears and finalizes", func(t *testing.T) {
		runner, handler, pod, transition, process := newPerpetualFactoryHarness(t, "start-error")
		candidateIO := &perpetualCoverageIO{startErr: errors.New("injected start failure")}
		handler.perpetualRuntimeFactory = func(*Pod, terminal.PtyProcess, int, int) (PodRuntime, error) {
			return PodRuntime{IO: candidateIO, Relay: &perpetualCoverageRelay{}}, nil
		}

		handler.restartPerpetualPod(pod, 0, transition)
		if _, ok := runner.podStore.Get(pod.PodKey); ok {
			t.Fatal("start failure retained pod ownership")
		}
		if candidateIO.starts != 1 || candidateIO.teardowns != 1 || process.kills != 1 || process.closes != 1 {
			t.Fatalf("start/teardown/kill/close = %d/%d/%d/%d",
				candidateIO.starts, candidateIO.teardowns, process.kills, process.closes)
		}
	})

	t.Run("transition loss after start retires candidate", func(t *testing.T) {
		runner, handler, pod, transition, _ := newPerpetualFactoryHarness(t, "end-transition-loss")
		candidateIO := &perpetualCoverageIO{}
		candidateIO.onStart = func() { pod.BeginRelayRuntimeTransition() }
		handler.perpetualRuntimeFactory = func(*Pod, terminal.PtyProcess, int, int) (PodRuntime, error) {
			return PodRuntime{IO: candidateIO, Relay: &perpetualCoverageRelay{}}, nil
		}

		handler.restartPerpetualPod(pod, 0, transition)
		if _, ok := runner.podStore.Get(pod.PodKey); ok {
			t.Fatal("transition-lost candidate retained pod ownership")
		}
		if candidateIO.starts != 1 || candidateIO.stops == 0 || candidateIO.teardowns == 0 {
			t.Fatalf("transition-lost start/stop/teardown = %d/%d/%d",
				candidateIO.starts, candidateIO.stops, candidateIO.teardowns)
		}
	})

	t.Run("backend notification error does not roll back commit", func(t *testing.T) {
		runner, handler, pod, transition, _ := newPerpetualFactoryHarness(t, "send-error")
		conn := handler.conn.(*client.MockConnection)
		conn.SendErr = errors.New("injected backend send failure")
		candidateIO := &perpetualCoverageIO{pid: 9090}
		handler.perpetualRuntimeFactory = func(*Pod, terminal.PtyProcess, int, int) (PodRuntime, error) {
			return PodRuntime{IO: candidateIO, Relay: &perpetualCoverageRelay{}}, nil
		}

		handler.restartPerpetualPod(pod, 0, transition)
		if pod.GetStatus() != PodStatusRunning || candidateIO.starts != 1 {
			t.Fatalf("notification error rolled back runtime: status=%s starts=%d",
				pod.GetStatus(), candidateIO.starts)
		}
		conn.SendErr = nil
		pod.Perpetual = false
		handler.cleanupPodExit(pod.PodKey, -1, true)
		if _, ok := runner.podStore.Get(pod.PodKey); ok {
			t.Fatal("committed send-error pod survived cleanup")
		}
	})
}
