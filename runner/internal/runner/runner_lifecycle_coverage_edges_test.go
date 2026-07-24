package runner

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/anthropics/agentsmesh/runner/internal/client"
)

type rejectDeletePodStore struct {
	*InMemoryPodStore
}

func (s *rejectDeletePodStore) DeleteIf(string, *Pod) bool { return false }

func TestStopAllPodsDrainsAutopilotsBeforeEmptyPodStore(t *testing.T) {
	runner, _ := NewTestRunner(t)
	controller := makeTestAC(t, "shutdown-autopilot", "shutdown-pod")
	runner.autopilotStore.AddAutopilot(controller)

	runner.stopAllPods()
	require.Nil(t, runner.autopilotStore.GetAutopilot("shutdown-autopilot"))
}

func TestStopAllPodsSkipsPodWhoseStoreOwnershipChanged(t *testing.T) {
	store := &rejectDeletePodStore{InMemoryPodStore: NewInMemoryPodStore()}
	pod := &Pod{PodKey: "replaced-before-shutdown", Status: PodStatusRunning}
	store.Put(pod.PodKey, pod)
	runner, _ := NewTestRunner(t, WithTestPodStore(store))

	runner.stopAllPods()
	current, found := store.Get(pod.PodKey)
	require.True(t, found)
	require.Same(t, pod, current)
	store.InMemoryPodStore.Delete(pod.PodKey)
}

func TestFinalizePodExitRejectsLostStoreOwnership(t *testing.T) {
	store := &rejectDeletePodStore{InMemoryPodStore: NewInMemoryPodStore()}
	pod := &Pod{PodKey: "replaced-before-finalize", Status: PodStatusRunning}
	store.Put(pod.PodKey, pod)
	runner := &Runner{podStore: store}
	handler := NewRunnerMessageHandler(runner, store, client.NewMockConnection())

	require.False(t, handler.finalizePodExit(pod.PodKey, pod, 0, false))
	store.InMemoryPodStore.Delete(pod.PodKey)
}

func TestStopPodDependentsRemovesAutopilot(t *testing.T) {
	runner, _ := NewTestRunner(t)
	controller := makeTestAC(t, "dependent-autopilot", "dependent-pod")
	runner.autopilotStore.AddAutopilot(controller)
	pod := &Pod{PodKey: "dependent-pod", Status: PodStatusRunning}

	runner.messageHandler.stopPodDependents(pod.PodKey, pod)
	require.Nil(t, runner.autopilotStore.GetAutopilot("dependent-autopilot"))
}

func TestSendPodTerminationSynthesizesExitError(t *testing.T) {
	runner, conn := NewTestRunner(t)
	pod := &Pod{PodKey: "failed-exit", Status: PodStatusStopped}

	runner.messageHandler.sendPodTermination(pod.PodKey, pod, 2, "")
	events := conn.GetEvents()
	require.NotEmpty(t, events)
	var payload map[string]interface{}
	for _, event := range events {
		if event.Type == client.MsgTypePodTerminated {
			payload, _ = event.Data.(map[string]interface{})
			break
		}
	}
	require.NotNil(t, payload)
	_, ok := payload["status"]
	require.True(t, ok)
	require.Equal(t, "error", payload["status"])
	require.Equal(t, "process exited with code 2", payload["error"])
}

func TestCleanupFailedPodRuntimeAcceptsNil(t *testing.T) {
	runner := &Runner{}
	handler := &RunnerMessageHandler{runner: runner}
	handler.cleanupFailedPodRuntime(nil)
}
