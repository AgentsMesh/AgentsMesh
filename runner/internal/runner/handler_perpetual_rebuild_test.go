package runner

import (
	"os"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/config"
)

func TestBuildPerpetualPTYRuntime(t *testing.T) {
	t.Run("rejects empty command", func(t *testing.T) {
		runner, _ := NewTestRunner(t)
		handler := runner.messageHandler
		pod := &Pod{PodKey: "invalid"}
		if _, err := handler.buildPerpetualPTYRuntime(pod, newOrderedStreamPTY(), 80, 24); err == nil {
			t.Fatal("empty launch command built a perpetual runtime")
		}
	})

	t.Run("assembles complete runtime", func(t *testing.T) {
		runner, _ := NewTestRunner(t, WithTestConfig(&config.Config{
			WorkspaceRoot: t.TempDir(), LogPTY: true,
		}))
		handler := runner.messageHandler
		pod := &Pod{PodKey: "perpetual", LaunchCommand: "fake", WorkDir: t.TempDir()}
		process := newOrderedStreamPTY()

		runtime, err := handler.buildPerpetualPTYRuntime(pod, process, 100, 30)
		if err != nil {
			t.Fatalf("buildPerpetualPTYRuntime: %v", err)
		}
		if !runtime.valid() || runtime.vtProvider == nil {
			t.Fatalf("perpetual runtime incomplete: %+v", runtime)
		}
		components := runtime.IO.(*PTYPodIO).components
		if components.Terminal == nil || components.VirtualTerminal == nil || components.Aggregator == nil || components.PTYLogger == nil {
			t.Fatalf("perpetual PTY components incomplete: %+v", components)
		}
		if err := runtime.IO.Start(); err != nil {
			t.Fatalf("start perpetual runtime: %v", err)
		}
		select {
		case <-process.readStarted:
		case <-time.After(time.Second):
			t.Fatal("perpetual PTY factory was not used")
		}
		close(process.releaseRead)
		runtime.IO.Stop()
		runtime.IO.Teardown()
		if err := os.RemoveAll(components.PTYLogger.LogDir()); err != nil {
			t.Fatalf("remove PTY log fixture: %v", err)
		}
	})
}

func TestOwnsPodRuntimeTransition(t *testing.T) {
	runner, _ := NewTestRunner(t)
	handler := runner.messageHandler
	pod := &Pod{PodKey: "pod", Status: PodStatusRunning}
	transition := pod.BeginRelayRuntimeTransition()

	if handler.ownsPodRuntimeTransition(pod, transition) {
		t.Fatal("pod absent from store owned transition")
	}
	runner.podStore.Put(pod.PodKey, &Pod{PodKey: pod.PodKey})
	if handler.ownsPodRuntimeTransition(pod, transition) {
		t.Fatal("replaced pod owned transition")
	}
	runner.podStore.Put(pod.PodKey, pod)
	if !handler.ownsPodRuntimeTransition(pod, transition) {
		t.Fatal("current pod did not own current transition")
	}
	pod.BeginRelayRuntimeTransition()
	if handler.ownsPodRuntimeTransition(pod, transition) {
		t.Fatal("stale transition retained ownership")
	}
}
