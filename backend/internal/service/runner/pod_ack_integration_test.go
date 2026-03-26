package runner

import (
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandlePodInitProgress_Received_ResolvesAck(t *testing.T) {
	pc, _, _, _ := setupPodEventHandlerDeps(t)

	pc.ackTracker.Register("progress-ack-pod")

	pc.handlePodInitProgress(1, &runnerv1.PodInitProgressEvent{
		PodKey:   "progress-ack-pod",
		Phase:    "received",
		Progress: 1,
		Message:  "Pod command received by runner",
	})

	// Resolve is fire-and-forget; just verify no panic
}

func TestHeartbeat_RecoverInitializingPod_AfterThreshold(t *testing.T) {
	pc, _, _, db := setupPodEventHandlerDeps(t)

	r := &runner.Runner{
		OrganizationID: 1,
		NodeID:         "recover-init-node",
		Status:         "online",
	}
	require.NoError(t, db.Create(r).Error)

	db.Exec(`INSERT INTO pods (pod_key, runner_id, status) VALUES (?, ?, ?)`,
		"recover-init-pod", r.ID, agentpod.StatusInitializing)

	var statusChanged bool
	pc.SetStatusChangeCallback(func(podKey, status, agentStatus string) {
		if podKey == "recover-init-pod" && status == agentpod.StatusRunning {
			statusChanged = true
		}
	})

	hbData := &runnerv1.HeartbeatData{
		Pods: []*runnerv1.PodInfo{{PodKey: "recover-init-pod", Status: "running"}},
	}

	// First heartbeat — below threshold, should NOT recover
	pc.handleHeartbeat(r.ID, hbData)
	var pod agentpod.Pod
	require.NoError(t, db.Where("pod_key = ?", "recover-init-pod").First(&pod).Error)
	assert.Equal(t, agentpod.StatusInitializing, pod.Status)
	assert.False(t, statusChanged)

	// Second heartbeat — reaches threshold, should recover
	pc.handleHeartbeat(r.ID, hbData)
	require.NoError(t, db.Where("pod_key = ?", "recover-init-pod").First(&pod).Error)
	assert.Equal(t, agentpod.StatusRunning, pod.Status)
	assert.True(t, statusChanged)
}

func TestHeartbeat_NoRecoverInitializingPod_BelowThreshold(t *testing.T) {
	pc, _, _, db := setupPodEventHandlerDeps(t)

	r := &runner.Runner{
		OrganizationID: 1,
		NodeID:         "slow-init-node",
		Status:         "online",
	}
	require.NoError(t, db.Create(r).Error)

	db.Exec(`INSERT INTO pods (pod_key, runner_id, status) VALUES (?, ?, ?)`,
		"slow-init-pod", r.ID, agentpod.StatusInitializing)

	// Single heartbeat — below threshold
	pc.handleHeartbeat(r.ID, &runnerv1.HeartbeatData{
		Pods: []*runnerv1.PodInfo{{PodKey: "slow-init-pod", Status: "initializing"}},
	})

	var pod agentpod.Pod
	require.NoError(t, db.Where("pod_key = ?", "slow-init-pod").First(&pod).Error)
	assert.Equal(t, agentpod.StatusInitializing, pod.Status)
}

func TestHeartbeat_InitReportCounter_ClearedByPodCreated(t *testing.T) {
	pc, _, _, db := setupPodEventHandlerDeps(t)

	r := &runner.Runner{
		OrganizationID: 1,
		NodeID:         "counter-clear-node",
		Status:         "online",
	}
	require.NoError(t, db.Create(r).Error)

	db.Exec(`INSERT INTO pods (pod_key, runner_id, status) VALUES (?, ?, ?)`,
		"counter-clear-pod", r.ID, agentpod.StatusInitializing)

	// First heartbeat — counter goes to 1
	pc.handleHeartbeat(r.ID, &runnerv1.HeartbeatData{
		Pods: []*runnerv1.PodInfo{{PodKey: "counter-clear-pod", Status: "initializing"}},
	})

	// PodCreated arrives — clears the counter
	pc.handlePodCreated(r.ID, &runnerv1.PodCreatedEvent{
		PodKey: "counter-clear-pod",
		Pid:    1234,
	})

	// Pod should be running now (via handlePodCreated, not heartbeat recovery)
	var pod agentpod.Pod
	require.NoError(t, db.Where("pod_key = ?", "counter-clear-pod").First(&pod).Error)
	assert.Equal(t, agentpod.StatusRunning, pod.Status)
}

func TestDisconnect_FailsInitializingPods(t *testing.T) {
	pc, _, _, db := setupPodEventHandlerDeps(t)

	r := &runner.Runner{
		OrganizationID: 1,
		NodeID:         "disconnect-init-node",
		Status:         "online",
		CurrentPods:    2,
	}
	require.NoError(t, db.Create(r).Error)

	db.Exec(`INSERT INTO pods (pod_key, runner_id, status) VALUES (?, ?, ?)`,
		"disconnect-init-pod", r.ID, agentpod.StatusInitializing)
	db.Exec(`INSERT INTO pods (pod_key, runner_id, status) VALUES (?, ?, ?)`,
		"disconnect-running-pod", r.ID, agentpod.StatusRunning)

	pc.ackTracker.Register("disconnect-init-pod")

	pc.handleRunnerDisconnect(r.ID)

	var initPod agentpod.Pod
	require.NoError(t, db.Where("pod_key = ?", "disconnect-init-pod").First(&initPod).Error)
	assert.Equal(t, agentpod.StatusError, initPod.Status)
	require.NotNil(t, initPod.ErrorCode)
	assert.Equal(t, ErrCodeRunnerDisconnected, *initPod.ErrorCode)

	var runPod agentpod.Pod
	require.NoError(t, db.Where("pod_key = ?", "disconnect-running-pod").First(&runPod).Error)
	assert.Equal(t, agentpod.StatusRunning, runPod.Status)
}
