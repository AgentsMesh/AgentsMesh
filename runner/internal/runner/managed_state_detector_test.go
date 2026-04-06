package runner

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/anthropics/agentsmesh/runner/internal/terminal/detector"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/vt"
)

func TestNewManagedStateDetector(t *testing.T) {
	vterminal := vt.NewVirtualTerminal(80, 24, 1000)

	d := NewManagedStateDetector(vterminal)
	require.NotNil(t, d)

	// Should start in NotRunning state
	assert.Equal(t, detector.StateNotRunning, d.GetState())

	// Cleanup
	d.Stop()
}

func TestManagedStateDetector_OnOutput(t *testing.T) {
	vterminal := vt.NewVirtualTerminal(80, 24, 1000)

	d := NewManagedStateDetector(vterminal)
	defer d.Stop()

	// Initially not running
	assert.Equal(t, detector.StateNotRunning, d.GetState())

	// Output should transition to Executing
	d.OnOutput(100)
	assert.Equal(t, detector.StateExecuting, d.GetState())
}

func TestManagedStateDetector_Subscribe(t *testing.T) {
	vterminal := vt.NewVirtualTerminal(80, 24, 1000)

	d := NewManagedStateDetector(vterminal)
	defer d.Stop()

	var mu sync.Mutex
	var events []detector.StateChangeEvent

	// Subscribe
	d.Subscribe("test-subscriber", func(event detector.StateChangeEvent) {
		mu.Lock()
		defer mu.Unlock()
		events = append(events, event)
	})

	// Trigger state change
	d.OnOutput(100)

	// Wait for async callback
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, events, 1)
	assert.Equal(t, detector.StateExecuting, events[0].NewState)
	assert.Equal(t, detector.StateNotRunning, events[0].PrevState)
}

func TestManagedStateDetector_Unsubscribe(t *testing.T) {
	vterminal := vt.NewVirtualTerminal(80, 24, 1000)

	d := NewManagedStateDetector(vterminal)
	defer d.Stop()

	var mu sync.Mutex
	var events []detector.StateChangeEvent

	// Subscribe
	d.Subscribe("test-subscriber", func(event detector.StateChangeEvent) {
		mu.Lock()
		defer mu.Unlock()
		events = append(events, event)
	})

	// Unsubscribe
	d.Unsubscribe("test-subscriber")

	// Trigger state change
	d.OnOutput(100)

	// Wait for potential async callback
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	assert.Len(t, events, 0, "Unsubscribed callback should not be called")
}

func TestManagedStateDetector_MultipleSubscribers(t *testing.T) {
	vterminal := vt.NewVirtualTerminal(80, 24, 1000)

	d := NewManagedStateDetector(vterminal)
	defer d.Stop()

	var mu sync.Mutex
	sub1Events := []detector.StateChangeEvent{}
	sub2Events := []detector.StateChangeEvent{}

	// Subscribe two subscribers
	d.Subscribe("sub1", func(event detector.StateChangeEvent) {
		mu.Lock()
		defer mu.Unlock()
		sub1Events = append(sub1Events, event)
	})
	d.Subscribe("sub2", func(event detector.StateChangeEvent) {
		mu.Lock()
		defer mu.Unlock()
		sub2Events = append(sub2Events, event)
	})

	// Trigger state change
	d.OnOutput(100)

	// Wait for async callbacks
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	assert.Len(t, sub1Events, 1, "sub1 should receive event")
	assert.Len(t, sub2Events, 1, "sub2 should receive event")
}

func TestManagedStateDetector_ImplementsStateDetector(t *testing.T) {
	vterminal := vt.NewVirtualTerminal(80, 24, 1000)

	// Compile-time check is in the main file, but we verify interface assignment here
	var sd detector.StateDetector = NewManagedStateDetector(vterminal)
	defer sd.(*ManagedStateDetector).Stop()

	// All interface methods should work
	sd.OnOutput(100)
	sd.OnScreenUpdate([]string{"test"})
	_ = sd.GetState()
	_ = sd.DetectState()
	sd.Subscribe("test", func(event detector.StateChangeEvent) {})
	sd.Unsubscribe("test")
	sd.Reset()
}

func TestManagedStateDetector_Reset(t *testing.T) {
	vterminal := vt.NewVirtualTerminal(80, 24, 1000)

	d := NewManagedStateDetector(vterminal)
	defer d.Stop()

	// Build up some state
	d.OnOutput(100)
	assert.Equal(t, detector.StateExecuting, d.GetState())

	// Reset
	d.Reset()
	assert.Equal(t, detector.StateNotRunning, d.GetState())
}

func TestManagedStateDetector_OnScreenUpdate(t *testing.T) {
	vterminal := vt.NewVirtualTerminal(80, 24, 1000)

	d := NewManagedStateDetector(vterminal)
	defer d.Stop()

	// Should not panic
	d.OnScreenUpdate([]string{"line1", "line2"})
	d.OnScreenUpdate(nil)
	d.OnScreenUpdate([]string{})
}
