package client

import (
	"sync/atomic"
	"testing"
)

func TestHeartbeatMonitor_AckResetsCounter(t *testing.T) {
	var triggered atomic.Bool
	m := NewHeartbeatMonitor(3, func() { triggered.Store(true) })

	m.OnSent() // missed=1
	m.OnSent() // missed=2
	if m.MissedCount() != 2 {
		t.Fatalf("expected 2 missed, got %d", m.MissedCount())
	}

	m.OnAck() // reset
	if m.MissedCount() != 0 {
		t.Fatalf("expected 0 missed after ack, got %d", m.MissedCount())
	}
	if triggered.Load() {
		t.Fatal("should not have triggered reconnect")
	}
}

func TestHeartbeatMonitor_TriggersAtThreshold(t *testing.T) {
	var triggered atomic.Bool
	m := NewHeartbeatMonitor(3, func() { triggered.Store(true) })

	m.OnSent() // missed=1
	m.OnSent() // missed=2
	if triggered.Load() {
		t.Fatal("should not trigger before threshold")
	}

	m.OnSent() // missed=3 → trigger
	if !triggered.Load() {
		t.Fatal("should have triggered at threshold")
	}
}

func TestHeartbeatMonitor_AckPreventsTriggering(t *testing.T) {
	var triggerCount atomic.Int32
	m := NewHeartbeatMonitor(3, func() { triggerCount.Add(1) })

	// Simulate normal heartbeat/ack cycle
	for i := 0; i < 10; i++ {
		m.OnSent()
		m.OnAck()
	}

	if triggerCount.Load() != 0 {
		t.Fatalf("should never trigger with regular acks, but triggered %d times", triggerCount.Load())
	}
}

func TestHeartbeatMonitor_ResetAfterTrigger(t *testing.T) {
	var triggerCount atomic.Int32
	m := NewHeartbeatMonitor(2, func() { triggerCount.Add(1) })

	m.OnSent() // missed=1
	m.OnSent() // missed=2 → trigger
	if triggerCount.Load() != 1 {
		t.Fatalf("expected 1 trigger, got %d", triggerCount.Load())
	}

	// Ack arrives (e.g., just before reconnect completes)
	m.OnAck()
	if m.MissedCount() != 0 {
		t.Fatal("ack should reset counter even after trigger")
	}
}
