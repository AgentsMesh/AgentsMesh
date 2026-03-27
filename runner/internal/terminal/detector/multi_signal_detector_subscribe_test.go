package detector

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMultiSignalDetector_Subscribe(t *testing.T) {
	d := NewMultiSignalDetector(MultiSignalConfig{})

	var mu sync.Mutex
	var events []StateChangeEvent

	// Subscribe
	d.Subscribe("test-subscriber", func(event StateChangeEvent) {
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
	assert.Equal(t, StateExecuting, events[0].NewState)
	assert.Equal(t, StateNotRunning, events[0].PrevState)
	assert.False(t, events[0].Timestamp.IsZero())
}

func TestMultiSignalDetector_Unsubscribe(t *testing.T) {
	d := NewMultiSignalDetector(MultiSignalConfig{})

	var mu sync.Mutex
	var events []StateChangeEvent

	// Subscribe
	d.Subscribe("test-subscriber", func(event StateChangeEvent) {
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

func TestMultiSignalDetector_MultipleSubscribers(t *testing.T) {
	d := NewMultiSignalDetector(MultiSignalConfig{})

	var mu sync.Mutex
	sub1Events := []StateChangeEvent{}
	sub2Events := []StateChangeEvent{}

	// Subscribe two subscribers
	d.Subscribe("sub1", func(event StateChangeEvent) {
		mu.Lock()
		defer mu.Unlock()
		sub1Events = append(sub1Events, event)
	})
	d.Subscribe("sub2", func(event StateChangeEvent) {
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
	assert.Equal(t, sub1Events[0].NewState, sub2Events[0].NewState)
}

func TestMultiSignalDetector_SubscribeReplace(t *testing.T) {
	d := NewMultiSignalDetector(MultiSignalConfig{})

	var mu sync.Mutex
	oldEvents := []StateChangeEvent{}
	newEvents := []StateChangeEvent{}

	// Subscribe with ID
	d.Subscribe("same-id", func(event StateChangeEvent) {
		mu.Lock()
		defer mu.Unlock()
		oldEvents = append(oldEvents, event)
	})

	// Replace with same ID
	d.Subscribe("same-id", func(event StateChangeEvent) {
		mu.Lock()
		defer mu.Unlock()
		newEvents = append(newEvents, event)
	})

	// Trigger state change
	d.OnOutput(100)

	// Wait for async callback
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	assert.Len(t, oldEvents, 0, "Old callback should be replaced")
	assert.Len(t, newEvents, 1, "New callback should receive event")
}

func TestMultiSignalDetector_StateChangeEventConfidence(t *testing.T) {
	d := NewMultiSignalDetector(MultiSignalConfig{
		IdleThreshold:    50 * time.Millisecond,
		ConfirmThreshold: 50 * time.Millisecond,
		MinStableTime:    50 * time.Millisecond,
		WaitingThreshold: 0.5,
	})

	var mu sync.Mutex
	var events []StateChangeEvent

	d.Subscribe("test", func(event StateChangeEvent) {
		mu.Lock()
		defer mu.Unlock()
		events = append(events, event)
	})

	// Start executing
	d.OnOutput(100)

	// Setup for waiting transition
	d.OnScreenUpdate([]string{"$ "})
	time.Sleep(200 * time.Millisecond)
	d.OnScreenUpdate([]string{"$ "})

	// Trigger detection which should transition to Waiting
	d.DetectState()

	// Wait for async callback
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	// Find the Waiting event
	var waitingEvent *StateChangeEvent
	for i := range events {
		if events[i].NewState == StateWaiting {
			waitingEvent = &events[i]
			break
		}
	}

	require.NotNil(t, waitingEvent, "Should have a Waiting transition event")
	assert.Equal(t, StateExecuting, waitingEvent.PrevState)
	assert.Greater(t, waitingEvent.Confidence, 0.0, "Confidence should be set for Waiting transition")
}

func TestMultiSignalDetector_LegacyAndSubscriberCallback(t *testing.T) {
	var mu sync.Mutex
	legacyEvents := []struct {
		newState  AgentState
		prevState AgentState
	}{}
	subscriberEvents := []StateChangeEvent{}

	d := NewMultiSignalDetector(MultiSignalConfig{
		OnStateChange: func(newState, prevState AgentState) {
			mu.Lock()
			defer mu.Unlock()
			legacyEvents = append(legacyEvents, struct {
				newState  AgentState
				prevState AgentState
			}{newState, prevState})
		},
	})

	d.Subscribe("test", func(event StateChangeEvent) {
		mu.Lock()
		defer mu.Unlock()
		subscriberEvents = append(subscriberEvents, event)
	})

	// Trigger state change
	d.OnOutput(100)

	// Wait for async callbacks
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	assert.Len(t, legacyEvents, 1, "Legacy callback should be called")
	assert.Len(t, subscriberEvents, 1, "Subscriber callback should be called")
}

func TestMultiSignalDetector_ConcurrentSubscribeUnsubscribe(t *testing.T) {
	d := NewMultiSignalDetector(MultiSignalConfig{})

	var wg sync.WaitGroup

	// Concurrent subscribe/unsubscribe operations
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			subID := "sub-" + string(rune('A'+id))
			for j := 0; j < 50; j++ {
				d.Subscribe(subID, func(event StateChangeEvent) {})
				d.OnOutput(10)
				d.Unsubscribe(subID)
			}
		}(i)
	}

	wg.Wait()
	// Test passes if no race conditions
}

func TestMultiSignalDetector_ImplementsStateDetector(t *testing.T) {
	// Compile-time check is in the main file, but we verify behavior here
	var sd StateDetector = NewMultiSignalDetector(MultiSignalConfig{})

	// All interface methods should work
	sd.OnOutput(100)
	sd.OnScreenUpdate([]string{"test"})
	_ = sd.GetState()
	_ = sd.DetectState()
	sd.SetCallback(func(newState, prevState AgentState) {})
	sd.Subscribe("test", func(event StateChangeEvent) {})
	sd.Unsubscribe("test")
	sd.Reset()
}
