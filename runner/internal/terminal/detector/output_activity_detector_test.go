package detector

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOutputActivityDetector(t *testing.T) {
	d := NewOutputActivityDetector(OutputActivityConfig{})

	assert.NotNil(t, d)
	assert.Equal(t, ActivityStateIdle, d.GetState())
	assert.Equal(t, 1*time.Second, d.windowDuration)
	assert.Equal(t, 500*time.Millisecond, d.idleThreshold)
	assert.Equal(t, 1*time.Second, d.confirmThreshold)
}

func TestOutputActivityDetector_CustomConfig(t *testing.T) {
	d := NewOutputActivityDetector(OutputActivityConfig{
		WindowDuration:   2 * time.Second,
		IdleThreshold:    100 * time.Millisecond,
		ConfirmThreshold: 200 * time.Millisecond,
	})

	assert.Equal(t, 2*time.Second, d.windowDuration)
	assert.Equal(t, 100*time.Millisecond, d.idleThreshold)
	assert.Equal(t, 200*time.Millisecond, d.confirmThreshold)
}

func TestOutputActivityDetector_OnOutput(t *testing.T) {
	d := NewOutputActivityDetector(OutputActivityConfig{})

	// Initially idle
	assert.True(t, d.IsIdle())
	assert.False(t, d.IsActive())

	// Receiving output transitions to active
	d.OnOutput(100)

	assert.True(t, d.IsActive())
	assert.False(t, d.IsIdle())
	assert.Equal(t, ActivityStateActive, d.GetState())
}

func TestOutputActivityDetector_StateTransitions(t *testing.T) {
	d := NewOutputActivityDetector(OutputActivityConfig{
		IdleThreshold:    50 * time.Millisecond,
		ConfirmThreshold: 50 * time.Millisecond,
	})

	// Start with output
	d.OnOutput(100)
	assert.Equal(t, ActivityStateActive, d.GetState())

	// Wait for idle threshold
	time.Sleep(60 * time.Millisecond)
	state := d.CheckState()
	assert.Equal(t, ActivityStatePotentialIdle, state)

	// Wait for confirm threshold
	time.Sleep(60 * time.Millisecond)
	state = d.CheckState()
	assert.Equal(t, ActivityStateIdle, state)

	// New output brings back to active
	d.OnOutput(50)
	assert.Equal(t, ActivityStateActive, d.GetState())
}

func TestOutputActivityDetector_Callback(t *testing.T) {
	var mu sync.Mutex
	var transitions []struct {
		newState  ActivityState
		prevState ActivityState
	}

	d := NewOutputActivityDetector(OutputActivityConfig{
		IdleThreshold:    50 * time.Millisecond,
		ConfirmThreshold: 50 * time.Millisecond,
		OnStateChange: func(newState, prevState ActivityState) {
			mu.Lock()
			defer mu.Unlock()
			transitions = append(transitions, struct {
				newState  ActivityState
				prevState ActivityState
			}{newState, prevState})
		},
	})

	// Trigger transitions
	d.OnOutput(100)                        // Idle -> Active
	time.Sleep(60 * time.Millisecond)      //
	d.CheckState()                         // Active -> PotentialIdle
	time.Sleep(60 * time.Millisecond)      //
	d.CheckState()                         // PotentialIdle -> Idle

	// Wait for callbacks to execute
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	require.Len(t, transitions, 3)
	assert.Equal(t, ActivityStateActive, transitions[0].newState)
	assert.Equal(t, ActivityStateIdle, transitions[0].prevState)
	assert.Equal(t, ActivityStatePotentialIdle, transitions[1].newState)
	assert.Equal(t, ActivityStateActive, transitions[1].prevState)
	assert.Equal(t, ActivityStateIdle, transitions[2].newState)
	assert.Equal(t, ActivityStatePotentialIdle, transitions[2].prevState)
}

func TestOutputActivityDetector_IdleDuration(t *testing.T) {
	d := NewOutputActivityDetector(OutputActivityConfig{})

	// No output yet
	assert.Equal(t, time.Duration(0), d.IdleDuration())

	// After output
	d.OnOutput(100)
	time.Sleep(50 * time.Millisecond)

	duration := d.IdleDuration()
	assert.GreaterOrEqual(t, duration.Milliseconds(), int64(50))
}

func TestOutputActivityDetector_OutputRate(t *testing.T) {
	d := NewOutputActivityDetector(OutputActivityConfig{
		WindowDuration: 100 * time.Millisecond,
	})

	// No output
	assert.Equal(t, float64(0), d.GetOutputRate())

	// Some output
	d.OnOutput(100)
	time.Sleep(100 * time.Millisecond)

	rate := d.GetOutputRate()
	// Should be roughly 100 bytes / 0.05 seconds = 2000 bytes/sec
	// But actual timing varies, so just check it's positive
	assert.Greater(t, rate, float64(0))
}

func TestOutputActivityDetector_Reset(t *testing.T) {
	d := NewOutputActivityDetector(OutputActivityConfig{})

	// Build up some state
	d.OnOutput(100)
	assert.Equal(t, ActivityStateActive, d.GetState())

	// Reset
	d.Reset()

	assert.Equal(t, ActivityStateIdle, d.GetState())
	assert.Equal(t, time.Duration(0), d.IdleDuration())
	assert.Equal(t, float64(0), d.GetOutputRate())
}

func TestOutputActivityDetector_SetCallback(t *testing.T) {
	d := NewOutputActivityDetector(OutputActivityConfig{})

	var called atomic.Bool
	d.SetCallback(func(newState, prevState ActivityState) {
		called.Store(true)
	})

	d.OnOutput(100)
	time.Sleep(50 * time.Millisecond)

	assert.True(t, called.Load())
}

func TestOutputActivityDetector_WindowReset(t *testing.T) {
	d := NewOutputActivityDetector(OutputActivityConfig{
		WindowDuration: 100 * time.Millisecond,
	})

	// First output
	d.OnOutput(100)
	time.Sleep(150 * time.Millisecond) // Window expires

	// Second output in new window
	d.OnOutput(50)

	// Small delay to ensure non-zero elapsed time for rate calculation
	time.Sleep(10 * time.Millisecond)

	// The output count should have been reset
	// We can't directly check outputCount, but the rate calculation reflects it
	rate := d.GetOutputRate()
	assert.Greater(t, rate, float64(0))
}

func TestOutputActivityDetector_ConcurrentAccess(t *testing.T) {
	d := NewOutputActivityDetector(OutputActivityConfig{})

	var wg sync.WaitGroup

	// Multiple goroutines sending output
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				d.OnOutput(10)
				d.CheckState()
				_ = d.GetState()
				_ = d.IdleDuration()
				_ = d.GetOutputRate()
			}
		}()
	}

	wg.Wait()
	// Test passes if no race conditions
}

func TestOutputActivityDetector_NoOutputNeverTransitions(t *testing.T) {
	d := NewOutputActivityDetector(OutputActivityConfig{
		IdleThreshold:    10 * time.Millisecond,
		ConfirmThreshold: 10 * time.Millisecond,
	})

	// Never receive output, check state multiple times
	for i := 0; i < 5; i++ {
		time.Sleep(20 * time.Millisecond)
		state := d.CheckState()
		assert.Equal(t, ActivityStateIdle, state)
	}
}

func TestOutputActivityDetector_IsPotentiallyIdle(t *testing.T) {
	d := NewOutputActivityDetector(OutputActivityConfig{
		IdleThreshold:    50 * time.Millisecond,
		ConfirmThreshold: 100 * time.Millisecond,
	})

	d.OnOutput(100)
	assert.False(t, d.IsPotentiallyIdle())

	time.Sleep(60 * time.Millisecond)
	d.CheckState()
	assert.True(t, d.IsPotentiallyIdle())

	// More output resets
	d.OnOutput(50)
	assert.False(t, d.IsPotentiallyIdle())
}
