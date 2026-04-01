package loop

import (
	"testing"
	"time"

	loopDomain "github.com/anthropics/agentsmesh/backend/internal/domain/loop"
	"github.com/stretchr/testify/assert"
)

// TestDeriveRunStatus is the core SSOT logic test.
// This function determines how Pod/Autopilot state maps to Loop Run status.
func TestDeriveRunStatus(t *testing.T) {
	// Direct mode (no autopilot)
	t.Run("direct mode: running pod → running", func(t *testing.T) {
		assert.Equal(t, loopDomain.RunStatusRunning, DeriveRunStatus("running", ""))
	})

	t.Run("direct mode: initializing pod → running", func(t *testing.T) {
		assert.Equal(t, loopDomain.RunStatusRunning, DeriveRunStatus("initializing", ""))
	})

	t.Run("direct mode: paused pod → running", func(t *testing.T) {
		assert.Equal(t, loopDomain.RunStatusRunning, DeriveRunStatus("paused", ""))
	})

	t.Run("direct mode: disconnected pod → running", func(t *testing.T) {
		assert.Equal(t, loopDomain.RunStatusRunning, DeriveRunStatus("disconnected", ""))
	})

	t.Run("direct mode: orphaned pod → running", func(t *testing.T) {
		assert.Equal(t, loopDomain.RunStatusRunning, DeriveRunStatus("orphaned", ""))
	})

	t.Run("direct mode: completed pod → completed", func(t *testing.T) {
		assert.Equal(t, loopDomain.RunStatusCompleted, DeriveRunStatus("completed", ""))
	})

	t.Run("direct mode: terminated pod → cancelled", func(t *testing.T) {
		assert.Equal(t, loopDomain.RunStatusCancelled, DeriveRunStatus("terminated", ""))
	})

	t.Run("direct mode: error pod → failed", func(t *testing.T) {
		assert.Equal(t, loopDomain.RunStatusFailed, DeriveRunStatus("error", ""))
	})

	// Autopilot mode — terminal phases are authoritative
	t.Run("autopilot: completed phase → completed", func(t *testing.T) {
		assert.Equal(t, loopDomain.RunStatusCompleted, DeriveRunStatus("running", "completed"))
	})

	t.Run("autopilot: failed phase → failed", func(t *testing.T) {
		assert.Equal(t, loopDomain.RunStatusFailed, DeriveRunStatus("running", "failed"))
	})

	t.Run("autopilot: stopped phase → cancelled", func(t *testing.T) {
		assert.Equal(t, loopDomain.RunStatusCancelled, DeriveRunStatus("running", "stopped"))
	})

	// Autopilot mode — active phase, pod still running
	t.Run("autopilot: running phase + running pod → running", func(t *testing.T) {
		assert.Equal(t, loopDomain.RunStatusRunning, DeriveRunStatus("running", "running"))
	})

	t.Run("autopilot: initializing phase + running pod → running", func(t *testing.T) {
		assert.Equal(t, loopDomain.RunStatusRunning, DeriveRunStatus("running", "initializing"))
	})

	t.Run("autopilot: waiting_approval phase + running pod → running", func(t *testing.T) {
		assert.Equal(t, loopDomain.RunStatusRunning, DeriveRunStatus("running", "waiting_approval"))
	})

	// CRITICAL: Autopilot non-terminal but Pod terminal → Pod wins (ground truth)
	// This handles manual Pod termination while autopilot is still active
	t.Run("autopilot: running phase + completed pod → completed (Pod wins)", func(t *testing.T) {
		assert.Equal(t, loopDomain.RunStatusCompleted, DeriveRunStatus("completed", "running"))
	})

	t.Run("autopilot: running phase + terminated pod → cancelled (Pod wins)", func(t *testing.T) {
		assert.Equal(t, loopDomain.RunStatusCancelled, DeriveRunStatus("terminated", "running"))
	})

	t.Run("autopilot: running phase + error pod → failed (Pod wins)", func(t *testing.T) {
		assert.Equal(t, loopDomain.RunStatusFailed, DeriveRunStatus("error", "running"))
	})

	t.Run("autopilot: initializing phase + completed pod → completed (Pod wins)", func(t *testing.T) {
		assert.Equal(t, loopDomain.RunStatusCompleted, DeriveRunStatus("completed", "initializing"))
	})

	t.Run("autopilot: waiting_approval phase + terminated pod → cancelled (Pod wins)", func(t *testing.T) {
		assert.Equal(t, loopDomain.RunStatusCancelled, DeriveRunStatus("terminated", "waiting_approval"))
	})
}

// TestIsPodDoneForLoop validates the Loop domain's definition of "pod done".
// This is deliberately different from Pod.IsTerminal() — it excludes orphaned
// but includes completed.
func TestIsPodDoneForLoop(t *testing.T) {
	t.Run("completed is done", func(t *testing.T) {
		assert.True(t, isPodDoneForLoop("completed"))
	})

	t.Run("terminated is done", func(t *testing.T) {
		assert.True(t, isPodDoneForLoop("terminated"))
	})

	t.Run("error is done", func(t *testing.T) {
		assert.True(t, isPodDoneForLoop("error"))
	})

	t.Run("running is not done", func(t *testing.T) {
		assert.False(t, isPodDoneForLoop("running"))
	})

	t.Run("initializing is not done", func(t *testing.T) {
		assert.False(t, isPodDoneForLoop("initializing"))
	})

	t.Run("orphaned is not done (may reconnect)", func(t *testing.T) {
		assert.False(t, isPodDoneForLoop("orphaned"))
	})
}

func TestResolveRunStatus(t *testing.T) {
	podKey := "test-pod-key"
	startedAt := time.Now().Add(-5 * time.Minute)
	finishedAt := time.Now()

	t.Run("should skip resolution when no pod_key", func(t *testing.T) {
		r := &loopDomain.LoopRun{Status: loopDomain.RunStatusPending, PodKey: nil}
		ResolveRunStatus(r, "completed", "", &finishedAt)
		assert.Equal(t, loopDomain.RunStatusPending, r.Status, "status should not change without pod_key")
		assert.Nil(t, r.FinishedAt)
	})

	t.Run("should resolve status from pod when pod_key is set", func(t *testing.T) {
		r := &loopDomain.LoopRun{
			Status:    loopDomain.RunStatusPending,
			PodKey:    &podKey,
			StartedAt: &startedAt,
		}
		ResolveRunStatus(r, "completed", "", &finishedAt)

		assert.Equal(t, loopDomain.RunStatusCompleted, r.Status)
		assert.NotNil(t, r.FinishedAt)
		assert.NotNil(t, r.DurationSec)
		assert.True(t, *r.DurationSec > 0)
	})

	t.Run("should compute duration from started_at and pod finished_at", func(t *testing.T) {
		start := time.Now().Add(-300 * time.Second)
		finish := time.Now()
		r := &loopDomain.LoopRun{
			Status:    loopDomain.RunStatusRunning,
			PodKey:    &podKey,
			StartedAt: &start,
		}
		ResolveRunStatus(r, "completed", "", &finish)

		assert.NotNil(t, r.DurationSec)
		assert.InDelta(t, 300, *r.DurationSec, 2) // ~300 seconds with tolerance
	})

	t.Run("should not set duration if started_at is nil", func(t *testing.T) {
		r := &loopDomain.LoopRun{
			Status:    loopDomain.RunStatusRunning,
			PodKey:    &podKey,
			StartedAt: nil,
		}
		ResolveRunStatus(r, "completed", "", &finishedAt)

		assert.Equal(t, loopDomain.RunStatusCompleted, r.Status)
		assert.NotNil(t, r.FinishedAt)
		assert.Nil(t, r.DurationSec)
	})

	t.Run("should not set finished_at if pod has no finished_at", func(t *testing.T) {
		r := &loopDomain.LoopRun{
			Status:    loopDomain.RunStatusRunning,
			PodKey:    &podKey,
			StartedAt: &startedAt,
		}
		ResolveRunStatus(r, "running", "running", nil)

		assert.Equal(t, loopDomain.RunStatusRunning, r.Status)
		assert.Nil(t, r.FinishedAt)
		assert.Nil(t, r.DurationSec)
	})

	t.Run("autopilot terminal phase overrides pod status", func(t *testing.T) {
		r := &loopDomain.LoopRun{
			Status:    loopDomain.RunStatusRunning,
			PodKey:    &podKey,
			StartedAt: &startedAt,
		}
		// Pod still running, but autopilot says completed
		ResolveRunStatus(r, "running", "completed", nil)

		assert.Equal(t, loopDomain.RunStatusCompleted, r.Status)
	})

	t.Run("pod terminal overrides autopilot non-terminal", func(t *testing.T) {
		r := &loopDomain.LoopRun{
			Status:    loopDomain.RunStatusRunning,
			PodKey:    &podKey,
			StartedAt: &startedAt,
		}
		// Autopilot still running, but Pod has terminated (killed)
		ResolveRunStatus(r, "terminated", "running", &finishedAt)

		assert.Equal(t, loopDomain.RunStatusCancelled, r.Status)
		assert.NotNil(t, r.FinishedAt)
	})
}
