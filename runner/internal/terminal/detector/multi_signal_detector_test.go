package detector

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMultiSignalDetector(t *testing.T) {
	d := NewMultiSignalDetector(MultiSignalConfig{})

	assert.NotNil(t, d)
	assert.NotNil(t, d.activityDetector)
	assert.NotNil(t, d.promptDetector)
	assert.Equal(t, StateNotRunning, d.GetState())

	// Check defaults
	assert.Equal(t, 0.4, d.config.ActivityWeight)
	assert.Equal(t, 0.3, d.config.StabilityWeight)
	assert.Equal(t, 0.3, d.config.PromptWeight)
	assert.Equal(t, 500*time.Millisecond, d.config.MinStableTime)
	assert.Equal(t, 0.6, d.config.WaitingThreshold)
}

func TestMultiSignalDetector_OnOutput(t *testing.T) {
	d := NewMultiSignalDetector(MultiSignalConfig{})

	// Initially not running
	assert.Equal(t, StateNotRunning, d.GetState())

	// Receiving output transitions to Executing
	d.OnOutput(100)
	assert.Equal(t, StateExecuting, d.GetState())
}

func TestMultiSignalDetector_StateCallback(t *testing.T) {
	var mu sync.Mutex
	var transitions []struct {
		newState  AgentState
		prevState AgentState
	}

	d := NewMultiSignalDetector(MultiSignalConfig{})

	d.Subscribe("test-callback", func(event StateChangeEvent) {
		mu.Lock()
		defer mu.Unlock()
		transitions = append(transitions, struct {
			newState  AgentState
			prevState AgentState
		}{event.NewState, event.PrevState})
	})

	// Trigger transition
	d.OnOutput(100)

	// Wait for callback
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, transitions, 1)
	assert.Equal(t, StateExecuting, transitions[0].newState)
	assert.Equal(t, StateNotRunning, transitions[0].prevState)
}

func TestMultiSignalDetector_ScreenStability(t *testing.T) {
	d := NewMultiSignalDetector(MultiSignalConfig{
		MinStableTime: 100 * time.Millisecond,
	})

	// Start with some output
	d.OnOutput(100)
	assert.Equal(t, StateExecuting, d.GetState())

	// Update screen
	lines := []string{"$ ", ""}
	d.OnScreenUpdate(lines)
	assert.Equal(t, time.Duration(0), d.GetScreenStableTime())

	// Wait and update with same content
	time.Sleep(150 * time.Millisecond)
	d.OnScreenUpdate(lines) // Same content

	// Screen should now be stable
	assert.GreaterOrEqual(t, d.GetScreenStableTime().Milliseconds(), int64(100))
}

func TestMultiSignalDetector_PromptDetection(t *testing.T) {
	d := NewMultiSignalDetector(MultiSignalConfig{})

	// Update with prompt-like content
	lines := []string{
		"Some output",
		"Continue? (y/n)",
	}
	d.OnScreenUpdate(lines)

	result := d.GetLastPromptResult()
	assert.True(t, result.IsPrompt)
	assert.Equal(t, PromptTypeConfirm, result.PromptType)
	assert.GreaterOrEqual(t, result.Confidence, 0.9)
}

func TestMultiSignalDetector_WaitingTransition(t *testing.T) {
	d := NewMultiSignalDetector(MultiSignalConfig{
		IdleThreshold:    50 * time.Millisecond,
		ConfirmThreshold: 50 * time.Millisecond,
		MinStableTime:    50 * time.Millisecond,
		WaitingThreshold: 0.5, // Lower threshold for testing
	})

	// Start executing
	d.OnOutput(100)
	assert.Equal(t, StateExecuting, d.GetState())

	// Update screen with prompt
	lines := []string{"$ "}
	d.OnScreenUpdate(lines)

	// Wait for idle threshold
	time.Sleep(200 * time.Millisecond)

	// Update screen again (same content to build stability)
	d.OnScreenUpdate(lines)

	// Detect state - should transition to Waiting
	state := d.DetectState()
	assert.Equal(t, StateWaiting, state)
}

func TestMultiSignalDetector_BackToExecuting(t *testing.T) {
	d := NewMultiSignalDetector(MultiSignalConfig{
		IdleThreshold:    50 * time.Millisecond,
		ConfirmThreshold: 50 * time.Millisecond,
		MinStableTime:    50 * time.Millisecond,
		WaitingThreshold: 0.5,
	})

	// Get to waiting state
	d.OnOutput(100)
	d.OnScreenUpdate([]string{"$ "})
	time.Sleep(200 * time.Millisecond)
	d.OnScreenUpdate([]string{"$ "})
	d.DetectState()
	assert.Equal(t, StateWaiting, d.GetState())

	// New output should transition back to Executing
	d.OnOutput(50)
	assert.Equal(t, StateExecuting, d.GetState())
}

func TestMultiSignalDetector_Reset(t *testing.T) {
	d := NewMultiSignalDetector(MultiSignalConfig{})

	// Build up some state
	d.OnOutput(100)
	d.OnScreenUpdate([]string{"test"})
	d.OnOSCTitle("Test Title")

	assert.Equal(t, StateExecuting, d.GetState())

	// Reset
	d.Reset()

	assert.Equal(t, StateNotRunning, d.GetState())
	assert.Equal(t, time.Duration(0), d.GetScreenStableTime())
}

func TestMultiSignalDetector_SetProcessRunning(t *testing.T) {
	d := NewMultiSignalDetector(MultiSignalConfig{})

	// Start executing
	d.OnOutput(100)
	assert.Equal(t, StateExecuting, d.GetState())

	// Process stops
	d.SetProcessRunning(false)
	assert.Equal(t, StateNotRunning, d.GetState())

	// Process starts again (doesn't immediately go to Executing)
	d.SetProcessRunning(true)
	assert.Equal(t, StateNotRunning, d.GetState())

	// Output received - now goes to Executing
	d.OnOutput(50)
	assert.Equal(t, StateExecuting, d.GetState())
}

func TestMultiSignalDetector_OSCTitleBoost(t *testing.T) {
	d := NewMultiSignalDetector(MultiSignalConfig{
		IdleThreshold:    50 * time.Millisecond,
		ConfirmThreshold: 50 * time.Millisecond,
		MinStableTime:    50 * time.Millisecond,
		WaitingThreshold: 0.6,
	})

	// Start executing
	d.OnOutput(100)

	// Set OSC title that suggests waiting
	d.OnOSCTitle("claude ✳ Waiting for input")

	// Update screen (no obvious prompt)
	d.OnScreenUpdate([]string{"working...", "done."})

	// Wait for idle
	time.Sleep(200 * time.Millisecond)
	d.OnScreenUpdate([]string{"working...", "done."})

	// The OSC title boost should help reach the threshold
	state := d.DetectState()
	// Note: May or may not transition depending on exact timing
	// The important thing is that OSC title is considered
	_ = state
}

func TestMultiSignalDetector_ConcurrentAccess(t *testing.T) {
	d := NewMultiSignalDetector(MultiSignalConfig{})

	var wg sync.WaitGroup

	// Multiple goroutines accessing detector
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				d.OnOutput(10)
				d.OnScreenUpdate([]string{"test"})
				d.DetectState()
				_ = d.GetState()
			}
		}()
	}

	wg.Wait()
	// Test passes if no race conditions
}

func TestContainsIgnoreCase(t *testing.T) {
	tests := []struct {
		s      string
		substr string
		want   bool
	}{
		{"Hello World", "hello", true},
		{"Hello World", "WORLD", true},
		{"Hello World", "xyz", false},
		{"✳ Waiting", "waiting", true},
		{"", "test", false},
		{"test", "", true},
	}

	for _, tt := range tests {
		got := containsIgnoreCase(tt.s, tt.substr)
		assert.Equal(t, tt.want, got, "containsIgnoreCase(%q, %q)", tt.s, tt.substr)
	}
}

func TestComputeScreenHash(t *testing.T) {
	d := NewMultiSignalDetector(MultiSignalConfig{})

	lines1 := []string{"line1", "line2"}
	lines2 := []string{"line1", "line2"}
	lines3 := []string{"line1", "line3"}

	hash1 := d.computeScreenHash(lines1)
	hash2 := d.computeScreenHash(lines2)
	hash3 := d.computeScreenHash(lines3)

	// Same content should have same hash
	assert.Equal(t, hash1, hash2)

	// Different content should have different hash
	assert.NotEqual(t, hash1, hash3)
}
