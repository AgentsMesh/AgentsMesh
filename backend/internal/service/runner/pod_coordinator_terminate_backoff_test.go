package runner

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== Terminate Backoff Tests ====================

func TestIsTerminateCooldown_NoPriorSend(t *testing.T) {
	_, cm, tr, hb, podStore, runnerRepo := setupPodCoordinatorDeps(t)
	pc := NewPodCoordinator(podStore, runnerRepo, cm, tr, hb, newTestLogger())

	// No prior terminate sent -- should not be in cooldown
	assert.False(t, pc.isTerminateCooldown("pod-1"))
}

func TestIsTerminateCooldown_WithinCooldown(t *testing.T) {
	_, cm, tr, hb, podStore, runnerRepo := setupPodCoordinatorDeps(t)
	pc := NewPodCoordinator(podStore, runnerRepo, cm, tr, hb, newTestLogger())

	// Record a terminate
	pc.recordTerminateSent("pod-1")

	// Should be within cooldown
	assert.True(t, pc.isTerminateCooldown("pod-1"))
}

func TestIsTerminateCooldown_AfterCooldownExpires(t *testing.T) {
	_, cm, tr, hb, podStore, runnerRepo := setupPodCoordinatorDeps(t)
	pc := NewPodCoordinator(podStore, runnerRepo, cm, tr, hb, newTestLogger())

	// Manually set a past timestamp (beyond cooldown)
	pc.terminateCacheMu.Lock()
	pc.terminateSentCache["pod-1"] = time.Now().Add(-terminateCooldown - time.Second)
	pc.terminateCacheMu.Unlock()

	// Should no longer be in cooldown
	assert.False(t, pc.isTerminateCooldown("pod-1"))
}

func TestRecordTerminateSent_UpdatesTimestamp(t *testing.T) {
	_, cm, tr, hb, podStore, runnerRepo := setupPodCoordinatorDeps(t)
	pc := NewPodCoordinator(podStore, runnerRepo, cm, tr, hb, newTestLogger())

	before := time.Now()
	pc.recordTerminateSent("pod-1")
	after := time.Now()

	pc.terminateCacheMu.Lock()
	recorded, ok := pc.terminateSentCache["pod-1"]
	pc.terminateCacheMu.Unlock()

	require.True(t, ok)
	assert.False(t, recorded.Before(before))
	assert.False(t, recorded.After(after))
}

func TestRecordTerminateSent_CleansExpiredEntries(t *testing.T) {
	_, cm, tr, hb, podStore, runnerRepo := setupPodCoordinatorDeps(t)
	pc := NewPodCoordinator(podStore, runnerRepo, cm, tr, hb, newTestLogger())

	// Pre-populate with an expired entry
	pc.terminateCacheMu.Lock()
	pc.terminateSentCache["old-pod"] = time.Now().Add(-terminateCacheCleanup - time.Minute)
	pc.terminateSentCache["recent-pod"] = time.Now().Add(-time.Minute) // still valid
	pc.terminateCacheMu.Unlock()

	// Recording a new entry triggers cleanup
	pc.recordTerminateSent("new-pod")

	pc.terminateCacheMu.Lock()
	_, hasOld := pc.terminateSentCache["old-pod"]
	_, hasRecent := pc.terminateSentCache["recent-pod"]
	_, hasNew := pc.terminateSentCache["new-pod"]
	pc.terminateCacheMu.Unlock()

	assert.False(t, hasOld, "expired entry should be cleaned up")
	assert.True(t, hasRecent, "recent entry should be preserved")
	assert.True(t, hasNew, "new entry should be present")
}

func TestTerminateCooldown_DifferentPods(t *testing.T) {
	_, cm, tr, hb, podStore, runnerRepo := setupPodCoordinatorDeps(t)
	pc := NewPodCoordinator(podStore, runnerRepo, cm, tr, hb, newTestLogger())

	// Record terminate for pod-1 only
	pc.recordTerminateSent("pod-1")

	// pod-1 should be in cooldown, pod-2 should not
	assert.True(t, pc.isTerminateCooldown("pod-1"))
	assert.False(t, pc.isTerminateCooldown("pod-2"))
}
