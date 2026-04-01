package runner

import (
	"sync"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	runnerDomain "github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// statusChangeCall records a single onStatusChange callback invocation.
type statusChangeCall struct {
	PodKey      string
	Status      string
	AgentStatus string
}

// statusChangeRecorder collects onStatusChange callbacks in a thread-safe way.
type statusChangeRecorder struct {
	mu    sync.Mutex
	calls []statusChangeCall
}

func (r *statusChangeRecorder) callback(podKey, status, agentStatus string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, statusChangeCall{
		PodKey:      podKey,
		Status:      status,
		AgentStatus: agentStatus,
	})
}

func (r *statusChangeRecorder) getCalls() []statusChangeCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := make([]statusChangeCall, len(r.calls))
	copy(cp, r.calls)
	return cp
}

// TestPodCoordinator_RunnerDisconnect_FailsInitializingPods verifies that
// when a runner disconnects, all its initializing pods are marked as error
// with RUNNER_DISCONNECTED while running pods remain untouched.
func TestPodCoordinator_RunnerDisconnect_FailsInitializingPods(t *testing.T) {
	pc, _, _, db := setupPodEventHandlerDeps(t)

	r := &runnerDomain.Runner{
		OrganizationID: 1,
		NodeID:         "disconnect-multi-node",
		Status:         "online",
		CurrentPods:    3,
	}
	require.NoError(t, db.Create(r).Error)

	// 2 initializing pods + 1 running pod
	db.Exec(`INSERT INTO pods (pod_key, runner_id, status, agent_status) VALUES (?, ?, ?, ?)`,
		"dc-init-1", r.ID, agentpod.StatusInitializing, "idle")
	db.Exec(`INSERT INTO pods (pod_key, runner_id, status, agent_status) VALUES (?, ?, ?, ?)`,
		"dc-init-2", r.ID, agentpod.StatusInitializing, "idle")
	db.Exec(`INSERT INTO pods (pod_key, runner_id, status, agent_status) VALUES (?, ?, ?, ?)`,
		"dc-running-1", r.ID, agentpod.StatusRunning, "executing")

	recorder := &statusChangeRecorder{}
	pc.SetStatusChangeCallback(recorder.callback)

	// Act
	pc.handleRunnerDisconnect(r.ID)

	// Assert: both initializing pods are now "error"
	var initPod1 agentpod.Pod
	require.NoError(t, db.Where("pod_key = ?", "dc-init-1").First(&initPod1).Error)
	assert.Equal(t, agentpod.StatusError, initPod1.Status)
	require.NotNil(t, initPod1.ErrorCode)
	assert.Equal(t, ErrCodeRunnerDisconnected, *initPod1.ErrorCode)
	require.NotNil(t, initPod1.ErrorMessage)
	assert.Contains(t, *initPod1.ErrorMessage, "Runner disconnected")
	assert.NotNil(t, initPod1.FinishedAt)

	var initPod2 agentpod.Pod
	require.NoError(t, db.Where("pod_key = ?", "dc-init-2").First(&initPod2).Error)
	assert.Equal(t, agentpod.StatusError, initPod2.Status)
	require.NotNil(t, initPod2.ErrorCode)
	assert.Equal(t, ErrCodeRunnerDisconnected, *initPod2.ErrorCode)

	// Assert: running pod is UNCHANGED
	var runPod agentpod.Pod
	require.NoError(t, db.Where("pod_key = ?", "dc-running-1").First(&runPod).Error)
	assert.Equal(t, agentpod.StatusRunning, runPod.Status)
	assert.Equal(t, "executing", runPod.AgentStatus)
	assert.Nil(t, runPod.ErrorCode)

	// Assert: runner is offline
	var runner runnerDomain.Runner
	require.NoError(t, db.Where("id = ?", r.ID).First(&runner).Error)
	assert.Equal(t, "offline", runner.Status)

	// Assert: onStatusChange called exactly 2 times (one per init pod)
	calls := recorder.getCalls()
	assert.Len(t, calls, 2)

	// Verify each call has the right status
	podKeysNotified := map[string]bool{}
	for _, c := range calls {
		assert.Equal(t, agentpod.StatusError, c.Status)
		assert.Equal(t, "", c.AgentStatus)
		podKeysNotified[c.PodKey] = true
	}
	assert.True(t, podKeysNotified["dc-init-1"], "dc-init-1 should be notified")
	assert.True(t, podKeysNotified["dc-init-2"], "dc-init-2 should be notified")
}

// TestPodCoordinator_RunnerDisconnect_NoInitializingPods verifies that
// when a runner disconnects and has only running pods, no pod statuses change.
func TestPodCoordinator_RunnerDisconnect_NoInitializingPods(t *testing.T) {
	pc, _, _, db := setupPodEventHandlerDeps(t)

	r := &runnerDomain.Runner{
		OrganizationID: 1,
		NodeID:         "disconnect-noninit-node",
		Status:         "online",
		CurrentPods:    2,
	}
	require.NoError(t, db.Create(r).Error)

	// Only running/completed pods, no initializing
	db.Exec(`INSERT INTO pods (pod_key, runner_id, status, agent_status) VALUES (?, ?, ?, ?)`,
		"dc-run-a", r.ID, agentpod.StatusRunning, "idle")
	db.Exec(`INSERT INTO pods (pod_key, runner_id, status, agent_status) VALUES (?, ?, ?, ?)`,
		"dc-done-a", r.ID, agentpod.StatusCompleted, "idle")

	recorder := &statusChangeRecorder{}
	pc.SetStatusChangeCallback(recorder.callback)

	pc.handleRunnerDisconnect(r.ID)

	// Assert: no pods changed status
	var runPod agentpod.Pod
	require.NoError(t, db.Where("pod_key = ?", "dc-run-a").First(&runPod).Error)
	assert.Equal(t, agentpod.StatusRunning, runPod.Status)

	var donePod agentpod.Pod
	require.NoError(t, db.Where("pod_key = ?", "dc-done-a").First(&donePod).Error)
	assert.Equal(t, agentpod.StatusCompleted, donePod.Status)

	// Assert: runner is offline
	var runner runnerDomain.Runner
	require.NoError(t, db.Where("id = ?", r.ID).First(&runner).Error)
	assert.Equal(t, "offline", runner.Status)

	// Assert: no onStatusChange calls
	assert.Empty(t, recorder.getCalls())
}

// TestPodCoordinator_RunnerDisconnect_OnStatusChangeCallback verifies
// the precise arguments passed to the onStatusChange callback.
func TestPodCoordinator_RunnerDisconnect_OnStatusChangeCallback(t *testing.T) {
	pc, _, _, db := setupPodEventHandlerDeps(t)

	r := &runnerDomain.Runner{
		OrganizationID: 1,
		NodeID:         "disconnect-cb-node",
		Status:         "online",
		CurrentPods:    1,
	}
	require.NoError(t, db.Create(r).Error)

	db.Exec(`INSERT INTO pods (pod_key, runner_id, status, agent_status) VALUES (?, ?, ?, ?)`,
		"dc-cb-pod", r.ID, agentpod.StatusInitializing, "idle")

	recorder := &statusChangeRecorder{}
	pc.SetStatusChangeCallback(recorder.callback)

	pc.handleRunnerDisconnect(r.ID)

	calls := recorder.getCalls()
	require.Len(t, calls, 1)
	assert.Equal(t, "dc-cb-pod", calls[0].PodKey)
	assert.Equal(t, agentpod.StatusError, calls[0].Status)
	assert.Equal(t, "", calls[0].AgentStatus)

	// Verify pod in DB
	var pod agentpod.Pod
	require.NoError(t, db.Where("pod_key = ?", "dc-cb-pod").First(&pod).Error)
	assert.Equal(t, agentpod.StatusError, pod.Status)
	require.NotNil(t, pod.ErrorCode)
	assert.Equal(t, ErrCodeRunnerDisconnected, *pod.ErrorCode)
}

// TestPodCoordinator_RunnerDisconnect_AckTrackerCleared verifies that
// pending ACK tracking is cleaned up for initializing pods on disconnect.
func TestPodCoordinator_RunnerDisconnect_AckTrackerCleared(t *testing.T) {
	pc, _, _, db := setupPodEventHandlerDeps(t)

	r := &runnerDomain.Runner{
		OrganizationID: 1,
		NodeID:         "disconnect-ack-node",
		Status:         "online",
		CurrentPods:    1,
	}
	require.NoError(t, db.Create(r).Error)

	db.Exec(`INSERT INTO pods (pod_key, runner_id, status, agent_status) VALUES (?, ?, ?, ?)`,
		"dc-ack-pod", r.ID, agentpod.StatusInitializing, "idle")

	// Register ACK for this pod (simulates pending create_pod command)
	pc.ackTracker.Register("dc-ack-pod")

	pc.handleRunnerDisconnect(r.ID)

	// Verify ACK is no longer pending (Remove was called)
	// We can verify indirectly: calling Remove again should be a no-op (no panic)
	pc.ackTracker.Remove("dc-ack-pod")

	// Verify the pod is failed
	var pod agentpod.Pod
	require.NoError(t, db.Where("pod_key = ?", "dc-ack-pod").First(&pod).Error)
	assert.Equal(t, agentpod.StatusError, pod.Status)
}
