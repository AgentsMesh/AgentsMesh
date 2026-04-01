package loop

import (
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
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
