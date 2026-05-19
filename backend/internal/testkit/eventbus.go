package testkit

import (
	"sync"
)

type CapturedEvent struct {
	Topic   string
	Payload interface{}
}

type CaptureEventBus struct {
	mu     sync.Mutex
	events []CapturedEvent
}

func NewCaptureEventBus() *CaptureEventBus {
	return &CaptureEventBus{}
}

func (b *CaptureEventBus) Publish(topic string, payload interface{}) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = append(b.events, CapturedEvent{Topic: topic, Payload: payload})
}

func (b *CaptureEventBus) Events() []CapturedEvent {
	b.mu.Lock()
	defer b.mu.Unlock()
	cp := make([]CapturedEvent, len(b.events))
	copy(cp, b.events)
	return cp
}

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

func (b *CaptureEventBus) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = nil
}
