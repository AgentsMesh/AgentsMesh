package testkit

import (
	"sync"
)

// CapturedEvent represents a captured event for test assertions.
type CapturedEvent struct {
	Topic   string
	Payload interface{}
}

// CaptureEventBus is a lightweight event bus for integration tests that
// captures all published events for later assertion.
type CaptureEventBus struct {
	mu     sync.Mutex
	events []CapturedEvent
}

// NewCaptureEventBus creates a new capture event bus.
func NewCaptureEventBus() *CaptureEventBus {
	return &CaptureEventBus{}
}

// Publish captures an event.
func (b *CaptureEventBus) Publish(topic string, payload interface{}) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = append(b.events, CapturedEvent{Topic: topic, Payload: payload})
}

// Events returns all captured events.
func (b *CaptureEventBus) Events() []CapturedEvent {
	b.mu.Lock()
	defer b.mu.Unlock()
	cp := make([]CapturedEvent, len(b.events))
	copy(cp, b.events)
	return cp
}

// HasEvent returns true if any event with the given topic was captured.
func (b *CaptureEventBus) HasEvent(topic string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, e := range b.events {
		if e.Topic == topic {
			return true
		}
	}
	return false
}

// EventCount returns the number of events with the given topic.
func (b *CaptureEventBus) EventCount(topic string) int {
	b.mu.Lock()
	defer b.mu.Unlock()
	count := 0
	for _, e := range b.events {
		if e.Topic == topic {
			count++
		}
	}
	return count
}

// Reset clears all captured events.
func (b *CaptureEventBus) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = nil
}
