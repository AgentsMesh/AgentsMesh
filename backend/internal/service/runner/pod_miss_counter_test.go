package runner

import (
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPodMissCount_IncrementAndClear(t *testing.T) {
	_, _, tr, hb, podStore, runnerRepo := setupPodCoordinatorDeps(t)
	logger := newTestLogger()
	cm := NewRunnerConnectionManager(logger)
	defer cm.Close()

	pc := NewPodCoordinator(podStore, runnerRepo, cm, tr, hb, logger)

	// Increment miss count
	assert.Equal(t, 1, pc.incrementMissCount("pod-1", 1))
	assert.Equal(t, 2, pc.incrementMissCount("pod-1", 1))
	assert.Equal(t, 3, pc.incrementMissCount("pod-1", 1))

	// Different pod has independent counter
	assert.Equal(t, 1, pc.incrementMissCount("pod-2", 1))

	// Clear single pod
	pc.clearMissCount("pod-1")
	assert.Equal(t, 1, pc.incrementMissCount("pod-1", 1), "counter should restart after clear")

	// pod-2 should be unaffected
	assert.Equal(t, 2, pc.incrementMissCount("pod-2", 1))
}

func TestPodMissCount_ClearForRunner(t *testing.T) {
	_, _, tr, hb, podStore, runnerRepo := setupPodCoordinatorDeps(t)
	logger := newTestLogger()
	cm := NewRunnerConnectionManager(logger)
	defer cm.Close()

	pc := NewPodCoordinator(podStore, runnerRepo, cm, tr, hb, logger)

	// Accumulate miss counts with runner ownership
	pc.incrementMissCount("pod-a", 1) // runner 1
	pc.incrementMissCount("pod-a", 1)
	pc.incrementMissCount("pod-b", 1) // runner 1
	pc.incrementMissCount("pod-c", 2) // runner 2

	// Clear for runner 1 only (no DB query needed — uses reverse index)
	pc.clearMissCountsForRunner(1)

	// Runner 1's pods should be reset
	assert.Equal(t, 1, pc.incrementMissCount("pod-a", 1), "pod-a should restart after runner clear")
	assert.Equal(t, 1, pc.incrementMissCount("pod-b", 1), "pod-b should restart after runner clear")

	// Runner 2's pod should be unaffected
	assert.Equal(t, 2, pc.incrementMissCount("pod-c", 2), "pod-c should continue counting")
}

func TestOrphanMissThreshold_NotReachedYet(t *testing.T) {
	db, _, tr, hb, podStore, runnerRepo := setupPodCoordinatorDeps(t)
	logger := newTestLogger()
	cm := NewRunnerConnectionManager(logger)
	defer cm.Close()

	pc := NewPodCoordinator(podStore, runnerRepo, cm, tr, hb, logger)

	// Create a running pod
	require.NoError(t, db.Exec(`INSERT INTO pods (pod_key, runner_id, status) VALUES (?, ?, ?)`,
		"pod-x", 1, agentpod.StatusRunning).Error)

	// Simulate heartbeats that don't report this pod, but fewer than threshold
	for i := 0; i < orphanMissThreshold-1; i++ {
		pc.reconcilePods(t.Context(), 1, map[string]bool{}) // empty heartbeat
	}

	// Pod should still be running (not yet orphaned)
	pod, err := podStore.GetByKey(t.Context(), "pod-x")
	require.NoError(t, err)
	assert.Equal(t, agentpod.StatusRunning, pod.Status, "pod should NOT be orphaned before threshold")
}

func TestOrphanMissThreshold_ReachedOrphans(t *testing.T) {
	db, _, tr, hb, podStore, runnerRepo := setupPodCoordinatorDeps(t)
	logger := newTestLogger()
	cm := NewRunnerConnectionManager(logger)
	defer cm.Close()

	pc := NewPodCoordinator(podStore, runnerRepo, cm, tr, hb, logger)

	// Create a running pod
	require.NoError(t, db.Exec(`INSERT INTO pods (pod_key, runner_id, status) VALUES (?, ?, ?)`,
		"pod-y", 1, agentpod.StatusRunning).Error)

	// Simulate heartbeats that don't report this pod, reaching threshold
	for i := 0; i < orphanMissThreshold; i++ {
		pc.reconcilePods(t.Context(), 1, map[string]bool{})
	}

	// Pod should now be orphaned
	pod, err := podStore.GetByKey(t.Context(), "pod-y")
	require.NoError(t, err)
	assert.Equal(t, agentpod.StatusOrphaned, pod.Status, "pod should be orphaned after threshold")
}

func TestOrphanMissThreshold_ResetByPresence(t *testing.T) {
	db, _, tr, hb, podStore, runnerRepo := setupPodCoordinatorDeps(t)
	logger := newTestLogger()
	cm := NewRunnerConnectionManager(logger)
	defer cm.Close()

	pc := NewPodCoordinator(podStore, runnerRepo, cm, tr, hb, logger)

	// Create a running pod
	require.NoError(t, db.Exec(`INSERT INTO pods (pod_key, runner_id, status) VALUES (?, ?, ?)`,
		"pod-z", 1, agentpod.StatusRunning).Error)

	// Miss twice (below threshold)
	pc.reconcilePods(t.Context(), 1, map[string]bool{})
	pc.reconcilePods(t.Context(), 1, map[string]bool{})

	// Pod reports in heartbeat — counter should reset
	pc.reconcilePods(t.Context(), 1, map[string]bool{"pod-z": true})

	// Miss twice again — still below threshold from fresh start
	pc.reconcilePods(t.Context(), 1, map[string]bool{})
	pc.reconcilePods(t.Context(), 1, map[string]bool{})

	// Pod should still be running
	pod, err := podStore.GetByKey(t.Context(), "pod-z")
	require.NoError(t, err)
	assert.Equal(t, agentpod.StatusRunning, pod.Status, "miss counter should have reset on pod presence")
}
