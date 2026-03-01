package loop

import (
	"testing"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/agentpod"
	"github.com/stretchr/testify/assert"
)

func TestLoopRun_TableName(t *testing.T) {
	r := LoopRun{}
	assert.Equal(t, "loop_runs", r.TableName())
}

func TestLoopRun_IsTerminal(t *testing.T) {
	terminalStatuses := []string{
		RunStatusCompleted,
		RunStatusFailed,
		RunStatusTimeout,
		RunStatusCancelled,
		RunStatusSkipped,
	}
	for _, status := range terminalStatuses {
		t.Run("should return true for "+status, func(t *testing.T) {
			r := &LoopRun{Status: status}
			assert.True(t, r.IsTerminal())
		})
	}

	activeStatuses := []string{RunStatusPending, RunStatusRunning}
	for _, status := range activeStatuses {
		t.Run("should return false for "+status, func(t *testing.T) {
			r := &LoopRun{Status: status}
			assert.False(t, r.IsTerminal())
		})
	}
}

func TestLoopRun_IsActive(t *testing.T) {
	t.Run("should return true for pending", func(t *testing.T) {
		r := &LoopRun{Status: RunStatusPending}
		assert.True(t, r.IsActive())
	})

	t.Run("should return true for running", func(t *testing.T) {
		r := &LoopRun{Status: RunStatusRunning}
		assert.True(t, r.IsActive())
	})

	t.Run("should return false for completed", func(t *testing.T) {
		r := &LoopRun{Status: RunStatusCompleted}
		assert.False(t, r.IsActive())
	})

	t.Run("should return false for failed", func(t *testing.T) {
		r := &LoopRun{Status: RunStatusFailed}
		assert.False(t, r.IsActive())
	})
}

// TestDeriveRunStatus is the core SSOT logic test.
// This function determines how Pod/Autopilot state maps to Loop Run status.
func TestDeriveRunStatus(t *testing.T) {
	// Direct mode (no autopilot)
	t.Run("direct mode: running pod → running", func(t *testing.T) {
		assert.Equal(t, RunStatusRunning, DeriveRunStatus("running", ""))
	})

	t.Run("direct mode: initializing pod → running", func(t *testing.T) {
		assert.Equal(t, RunStatusRunning, DeriveRunStatus("initializing", ""))
	})

	t.Run("direct mode: paused pod → running", func(t *testing.T) {
		assert.Equal(t, RunStatusRunning, DeriveRunStatus("paused", ""))
	})

	t.Run("direct mode: disconnected pod → running", func(t *testing.T) {
		assert.Equal(t, RunStatusRunning, DeriveRunStatus("disconnected", ""))
	})

	t.Run("direct mode: orphaned pod → running", func(t *testing.T) {
		assert.Equal(t, RunStatusRunning, DeriveRunStatus("orphaned", ""))
	})

	t.Run("direct mode: completed pod → completed", func(t *testing.T) {
		assert.Equal(t, RunStatusCompleted, DeriveRunStatus("completed", ""))
	})

	t.Run("direct mode: terminated pod → cancelled", func(t *testing.T) {
		assert.Equal(t, RunStatusCancelled, DeriveRunStatus("terminated", ""))
	})

	t.Run("direct mode: error pod → failed", func(t *testing.T) {
		assert.Equal(t, RunStatusFailed, DeriveRunStatus("error", ""))
	})

	// Autopilot mode — terminal phases are authoritative
	t.Run("autopilot: completed phase → completed", func(t *testing.T) {
		assert.Equal(t, RunStatusCompleted, DeriveRunStatus("running", "completed"))
	})

	t.Run("autopilot: failed phase → failed", func(t *testing.T) {
		assert.Equal(t, RunStatusFailed, DeriveRunStatus("running", "failed"))
	})

	t.Run("autopilot: stopped phase → cancelled", func(t *testing.T) {
		assert.Equal(t, RunStatusCancelled, DeriveRunStatus("running", "stopped"))
	})

	// Autopilot mode — active phase, pod still running
	t.Run("autopilot: running phase + running pod → running", func(t *testing.T) {
		assert.Equal(t, RunStatusRunning, DeriveRunStatus("running", "running"))
	})

	t.Run("autopilot: initializing phase + running pod → running", func(t *testing.T) {
		assert.Equal(t, RunStatusRunning, DeriveRunStatus("running", "initializing"))
	})

	t.Run("autopilot: waiting_approval phase + running pod → running", func(t *testing.T) {
		assert.Equal(t, RunStatusRunning, DeriveRunStatus("running", "waiting_approval"))
	})

	// CRITICAL: Autopilot non-terminal but Pod terminal → Pod wins (ground truth)
	// This handles manual Pod termination while autopilot is still active
	t.Run("autopilot: running phase + completed pod → completed (Pod wins)", func(t *testing.T) {
		assert.Equal(t, RunStatusCompleted, DeriveRunStatus("completed", "running"))
	})

	t.Run("autopilot: running phase + terminated pod → cancelled (Pod wins)", func(t *testing.T) {
		assert.Equal(t, RunStatusCancelled, DeriveRunStatus("terminated", "running"))
	})

	t.Run("autopilot: running phase + error pod → failed (Pod wins)", func(t *testing.T) {
		assert.Equal(t, RunStatusFailed, DeriveRunStatus("error", "running"))
	})

	t.Run("autopilot: initializing phase + completed pod → completed (Pod wins)", func(t *testing.T) {
		assert.Equal(t, RunStatusCompleted, DeriveRunStatus("completed", "initializing"))
	})

	t.Run("autopilot: waiting_approval phase + terminated pod → cancelled (Pod wins)", func(t *testing.T) {
		assert.Equal(t, RunStatusCancelled, DeriveRunStatus("terminated", "waiting_approval"))
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

// TestPodDomainHelpers validates the agentpod package-level status helpers
// used across domains for consistent status classification.
func TestPodDomainHelpers(t *testing.T) {
	t.Run("IsPodStatusTerminal excludes completed", func(t *testing.T) {
		assert.False(t, agentpod.IsPodStatusTerminal("completed"))
	})

	t.Run("IsPodStatusTerminal includes orphaned", func(t *testing.T) {
		assert.True(t, agentpod.IsPodStatusTerminal("orphaned"))
	})

	t.Run("IsPodStatusFinished includes completed and terminal", func(t *testing.T) {
		assert.True(t, agentpod.IsPodStatusFinished("completed"))
		assert.True(t, agentpod.IsPodStatusFinished("terminated"))
		assert.True(t, agentpod.IsPodStatusFinished("error"))
		assert.False(t, agentpod.IsPodStatusFinished("running"))
	})

	t.Run("IsPodStatusActive covers active states", func(t *testing.T) {
		assert.True(t, agentpod.IsPodStatusActive("running"))
		assert.True(t, agentpod.IsPodStatusActive("initializing"))
		assert.True(t, agentpod.IsPodStatusActive("paused"))
		assert.True(t, agentpod.IsPodStatusActive("disconnected"))
		assert.False(t, agentpod.IsPodStatusActive("completed"))
	})
}

func TestLoopRun_ResolveStatus(t *testing.T) {
	podKey := "test-pod-key"
	startedAt := time.Now().Add(-5 * time.Minute)
	finishedAt := time.Now()

	t.Run("should skip resolution when no pod_key", func(t *testing.T) {
		r := &LoopRun{Status: RunStatusPending, PodKey: nil}
		r.ResolveStatus("completed", "", &finishedAt)
		assert.Equal(t, RunStatusPending, r.Status, "status should not change without pod_key")
		assert.Nil(t, r.FinishedAt)
	})

	t.Run("should resolve status from pod when pod_key is set", func(t *testing.T) {
		r := &LoopRun{
			Status:    RunStatusPending,
			PodKey:    &podKey,
			StartedAt: &startedAt,
		}
		r.ResolveStatus("completed", "", &finishedAt)

		assert.Equal(t, RunStatusCompleted, r.Status)
		assert.NotNil(t, r.FinishedAt)
		assert.NotNil(t, r.DurationSec)
		assert.True(t, *r.DurationSec > 0)
	})

	t.Run("should compute duration from started_at and pod finished_at", func(t *testing.T) {
		start := time.Now().Add(-300 * time.Second)
		finish := time.Now()
		r := &LoopRun{
			Status:    RunStatusRunning,
			PodKey:    &podKey,
			StartedAt: &start,
		}
		r.ResolveStatus("completed", "", &finish)

		assert.NotNil(t, r.DurationSec)
		assert.InDelta(t, 300, *r.DurationSec, 2) // ~300 seconds with tolerance
	})

	t.Run("should not set duration if started_at is nil", func(t *testing.T) {
		r := &LoopRun{
			Status:    RunStatusRunning,
			PodKey:    &podKey,
			StartedAt: nil,
		}
		r.ResolveStatus("completed", "", &finishedAt)

		assert.Equal(t, RunStatusCompleted, r.Status)
		assert.NotNil(t, r.FinishedAt)
		assert.Nil(t, r.DurationSec)
	})

	t.Run("should not set finished_at if pod has no finished_at", func(t *testing.T) {
		r := &LoopRun{
			Status:    RunStatusRunning,
			PodKey:    &podKey,
			StartedAt: &startedAt,
		}
		r.ResolveStatus("running", "running", nil)

		assert.Equal(t, RunStatusRunning, r.Status)
		assert.Nil(t, r.FinishedAt)
		assert.Nil(t, r.DurationSec)
	})

	t.Run("autopilot terminal phase overrides pod status", func(t *testing.T) {
		r := &LoopRun{
			Status:    RunStatusRunning,
			PodKey:    &podKey,
			StartedAt: &startedAt,
		}
		// Pod still running, but autopilot says completed
		r.ResolveStatus("running", "completed", nil)

		assert.Equal(t, RunStatusCompleted, r.Status)
	})

	t.Run("pod terminal overrides autopilot non-terminal", func(t *testing.T) {
		r := &LoopRun{
			Status:    RunStatusRunning,
			PodKey:    &podKey,
			StartedAt: &startedAt,
		}
		// Autopilot still running, but Pod has terminated (killed)
		r.ResolveStatus("terminated", "running", &finishedAt)

		assert.Equal(t, RunStatusCancelled, r.Status)
		assert.NotNil(t, r.FinishedAt)
	})
}
