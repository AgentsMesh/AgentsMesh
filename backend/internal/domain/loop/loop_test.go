package loop

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoop_TableName(t *testing.T) {
	l := Loop{}
	assert.Equal(t, "loops", l.TableName())
}

func TestLoop_IsEnabled(t *testing.T) {
	t.Run("should return true when enabled", func(t *testing.T) {
		l := &Loop{Status: StatusEnabled}
		assert.True(t, l.IsEnabled())
	})

	t.Run("should return false when disabled", func(t *testing.T) {
		l := &Loop{Status: StatusDisabled}
		assert.False(t, l.IsEnabled())
	})

	t.Run("should return false when archived", func(t *testing.T) {
		l := &Loop{Status: StatusArchived}
		assert.False(t, l.IsEnabled())
	})
}

func TestLoop_HasCron(t *testing.T) {
	cron := "0 9 * * *"
	empty := ""

	t.Run("should return true when cron expression is set", func(t *testing.T) {
		l := &Loop{CronExpression: &cron}
		assert.True(t, l.HasCron())
	})

	t.Run("should return false when cron is nil", func(t *testing.T) {
		l := &Loop{CronExpression: nil}
		assert.False(t, l.HasCron())
	})

	t.Run("should return false when cron is empty string", func(t *testing.T) {
		l := &Loop{CronExpression: &empty}
		assert.False(t, l.HasCron())
	})
}

func TestLoop_IsAutopilot(t *testing.T) {
	t.Run("should return true for autopilot mode", func(t *testing.T) {
		l := &Loop{ExecutionMode: ExecutionModeAutopilot}
		assert.True(t, l.IsAutopilot())
	})

	t.Run("should return false for direct mode", func(t *testing.T) {
		l := &Loop{ExecutionMode: ExecutionModeDirect}
		assert.False(t, l.IsAutopilot())
	})
}

func TestLoop_IsPersistent(t *testing.T) {
	t.Run("should return true for persistent strategy", func(t *testing.T) {
		l := &Loop{SandboxStrategy: SandboxStrategyPersistent}
		assert.True(t, l.IsPersistent())
	})

	t.Run("should return false for fresh strategy", func(t *testing.T) {
		l := &Loop{SandboxStrategy: SandboxStrategyFresh}
		assert.False(t, l.IsPersistent())
	})
}

func TestLoop_SuccessRate(t *testing.T) {
	t.Run("should return 0 when no runs", func(t *testing.T) {
		l := &Loop{TotalRuns: 0}
		assert.Equal(t, float64(0), l.SuccessRate())
	})

	t.Run("should return 100 when all successful", func(t *testing.T) {
		l := &Loop{TotalRuns: 10, SuccessfulRuns: 10}
		assert.Equal(t, float64(100), l.SuccessRate())
	})

	t.Run("should return correct percentage", func(t *testing.T) {
		l := &Loop{TotalRuns: 10, SuccessfulRuns: 7}
		assert.Equal(t, float64(70), l.SuccessRate())
	})
}
