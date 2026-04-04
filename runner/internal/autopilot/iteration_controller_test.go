package autopilot

import (
	"testing"
	"time"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
)

func TestIterationController_NewIterationController(t *testing.T) {
	reporter := &MockEventReporter{}

	ic := NewIterationController(IterationControllerConfig{
		MaxIterations: 20,
		MinTriggerGap: 10 * time.Second,
		Reporter:      reporter,
		AutopilotKey:  "autopilot-123",
		PodKey:        "worker-123",
		Logger:        nil,
	})

	assert.NotNil(t, ic)
	assert.Equal(t, 20, ic.GetMaxIterations())
	assert.Equal(t, 0, ic.GetCurrentIteration())
}

func TestIterationController_DefaultValues(t *testing.T) {
	// Test with zero values - should use defaults
	ic := NewIterationController(IterationControllerConfig{
		MaxIterations: 0, // Should default to 10
		MinTriggerGap: 0, // Should default to 5 seconds
	})

	assert.Equal(t, 10, ic.GetMaxIterations())
}

func TestIterationController_GetStartedAt(t *testing.T) {
	before := time.Now()
	ic := NewIterationController(IterationControllerConfig{
		MaxIterations: 10,
	})
	after := time.Now()

	startedAt := ic.GetStartedAt()
	assert.True(t, startedAt.After(before) || startedAt.Equal(before))
	assert.True(t, startedAt.Before(after) || startedAt.Equal(after))
}

func TestIterationController_GetLastIterationAt(t *testing.T) {
	ic := NewIterationController(IterationControllerConfig{
		MaxIterations: 10,
	})

	// Initially zero
	assert.True(t, ic.GetLastIterationAt().IsZero())

	// After setting initial iteration
	ic.SetInitialIteration()
	assert.False(t, ic.GetLastIterationAt().IsZero())
}

func TestIterationController_AddMaxIterations(t *testing.T) {
	ic := NewIterationController(IterationControllerConfig{
		MaxIterations: 10,
	})

	newMax := ic.AddMaxIterations(5)
	assert.Equal(t, 15, newMax)
	assert.Equal(t, 15, ic.GetMaxIterations())
}

func TestIterationController_CheckTriggerDedup(t *testing.T) {
	ic := NewIterationController(IterationControllerConfig{
		MaxIterations: 10,
		MinTriggerGap: 100 * time.Millisecond,
	})

	// First call should pass
	assert.True(t, ic.CheckTriggerDedup())

	// Immediate second call should fail (too soon)
	assert.False(t, ic.CheckTriggerDedup())

	// Wait for gap and try again
	time.Sleep(110 * time.Millisecond)
	assert.True(t, ic.CheckTriggerDedup())
}

func TestIterationController_CheckTriggerDedup_WithLogger(t *testing.T) {
	ic := NewIterationController(IterationControllerConfig{
		MaxIterations: 10,
		MinTriggerGap: 100 * time.Millisecond,
		AutopilotKey:  "autopilot-123",
		Logger:        nil, // Will use default logger
	})

	// First call should pass
	assert.True(t, ic.CheckTriggerDedup())

	// Second call should fail and log
	assert.False(t, ic.CheckTriggerDedup())
}

func TestIterationController_UpdateTriggerTime(t *testing.T) {
	ic := NewIterationController(IterationControllerConfig{
		MaxIterations: 10,
		MinTriggerGap: 100 * time.Millisecond,
	})

	// First trigger
	assert.True(t, ic.CheckTriggerDedup())

	// Wait a bit but not enough
	time.Sleep(50 * time.Millisecond)

	// Update trigger time manually
	ic.UpdateTriggerTime()

	// Now immediate check should fail again
	assert.False(t, ic.CheckTriggerDedup())
}

func TestIterationController_HasReachedMaxIterations(t *testing.T) {
	ic := NewIterationController(IterationControllerConfig{
		MaxIterations: 2,
	})

	assert.False(t, ic.HasReachedMaxIterations())

	ic.SetInitialIteration() // 1
	assert.False(t, ic.HasReachedMaxIterations())

	ic.IncrementIteration() // 2
	assert.True(t, ic.HasReachedMaxIterations())
}

func TestIterationController_IncrementIteration(t *testing.T) {
	ic := NewIterationController(IterationControllerConfig{
		MaxIterations: 3,
	})

	// First increment
	iter, ok := ic.IncrementIteration()
	assert.True(t, ok)
	assert.Equal(t, 1, iter)

	// Second increment
	iter, ok = ic.IncrementIteration()
	assert.True(t, ok)
	assert.Equal(t, 2, iter)

	// Third increment
	iter, ok = ic.IncrementIteration()
	assert.True(t, ok)
	assert.Equal(t, 3, iter)

	// Fourth increment - should fail (max reached)
	iter, ok = ic.IncrementIteration()
	assert.False(t, ok)
	assert.Equal(t, 3, iter) // Returns current, not incremented
}

func TestIterationController_SetInitialIteration(t *testing.T) {
	ic := NewIterationController(IterationControllerConfig{
		MaxIterations: 10,
	})

	iter := ic.SetInitialIteration()
	assert.Equal(t, 1, iter)
	assert.Equal(t, 1, ic.GetCurrentIteration())
}

func TestIterationController_ReportIterationEvent(t *testing.T) {
	reporter := &MockEventReporter{}
	ic := NewIterationController(IterationControllerConfig{
		MaxIterations: 10,
		Reporter:      reporter,
		AutopilotKey:  "autopilot-123",
	})

	ic.ReportIterationEvent(1, "started", "Starting iteration", []string{"file1.go", "file2.go"})

	iterationEvents := reporter.GetIterationEvents()
	assert.Len(t, iterationEvents, 1)
	assert.Equal(t, int32(1), iterationEvents[0].Iteration)
	assert.Equal(t, "started", iterationEvents[0].Phase)
	assert.Equal(t, "Starting iteration", iterationEvents[0].Summary)
	assert.Equal(t, []string{"file1.go", "file2.go"}, iterationEvents[0].FilesChanged)
}

func TestIterationController_ReportIterationEvent_NilReporter(t *testing.T) {
	ic := NewIterationController(IterationControllerConfig{
		MaxIterations: 10,
		Reporter:      nil,
	})

	// Should not panic
	ic.ReportIterationEvent(1, "started", "test", nil)
}

func TestIterationController_GetStatus(t *testing.T) {
	ic := NewIterationController(IterationControllerConfig{
		MaxIterations: 10,
	})

	ic.SetInitialIteration()

	status := ic.GetStatus()
	assert.Equal(t, int32(1), status.CurrentIteration)
	assert.Equal(t, int32(10), status.MaxIterations)
	assert.NotZero(t, status.StartedAt)
	assert.NotZero(t, status.LastIterationAt)
}

func TestSanitizeUTF8String(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "valid ASCII within limit",
			input:    "hello world",
			maxLen:   100,
			expected: "hello world",
		},
		{
			name:     "valid UTF-8 Chinese within limit",
			input:    "你好世界",
			maxLen:   100,
			expected: "你好世界",
		},
		{
			name:     "truncate ASCII at boundary",
			input:    "hello world",
			maxLen:   5,
			expected: "hello",
		},
		{
			name:     "truncate Chinese at rune boundary",
			input:    "你好世界",
			maxLen:   6, // "你好" is 6 bytes
			expected: "你好",
		},
		{
			name:     "mixed content truncation",
			input:    "Hello 世界",
			maxLen:   10,
			expected: "Hello 世",
		},
		{
			name:     "empty string",
			input:    "",
			maxLen:   100,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeUTF8String(tt.input, tt.maxLen)
			assert.Equal(t, tt.expected, result)
			// Verify result is valid UTF-8
			assert.True(t, isValidUTF8(result), "result should be valid UTF-8")
		})
	}
}

func isValidUTF8(s string) bool {
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			return false
		}
		i += size
	}
	return true
}
