package eventbus

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"
)

func TestEventBus_Publish(t *testing.T) {
	t.Run("sets timestamp if not set", func(t *testing.T) {
		eb := NewEventBus(nil, nil)
		defer eb.Close()

		event := &Event{
			Type:           EventPodCreated,
			OrganizationID: 1,
		}

		before := time.Now().UnixMilli()
		err := eb.Publish(context.Background(), event)
		after := time.Now().UnixMilli()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if event.Timestamp < before || event.Timestamp > after {
			t.Errorf("timestamp %d not in range [%d, %d]", event.Timestamp, before, after)
		}
	})

	t.Run("preserves existing timestamp", func(t *testing.T) {
		eb := NewEventBus(nil, nil)
		defer eb.Close()

		existingTs := int64(1234567890)
		event := &Event{
			Type:           EventPodCreated,
			OrganizationID: 1,
			Timestamp:      existingTs,
		}

		err := eb.Publish(context.Background(), event)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if event.Timestamp != existingTs {
			t.Errorf("timestamp changed from %d to %d", existingTs, event.Timestamp)
		}
	})

	t.Run("sets category from registry if not set", func(t *testing.T) {
		eb := NewEventBus(nil, nil)
		defer eb.Close()

		event := &Event{
			Type:           EventPodNotification, // Notification category
			OrganizationID: 1,
		}

		err := eb.Publish(context.Background(), event)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if event.Category != CategoryNotification {
			t.Errorf("expected category %s, got %s", CategoryNotification, event.Category)
		}
	})

	t.Run("preserves existing category", func(t *testing.T) {
		eb := NewEventBus(nil, nil)
		defer eb.Close()

		event := &Event{
			Type:           EventPodCreated,
			Category:       CategorySystem, // Override default
			OrganizationID: 1,
		}

		err := eb.Publish(context.Background(), event)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if event.Category != CategorySystem {
			t.Errorf("category changed from %s to %s", CategorySystem, event.Category)
		}
	})

	t.Run("sets source instance ID", func(t *testing.T) {
		eb := NewEventBus(nil, nil)
		defer eb.Close()

		event := &Event{
			Type:           EventPodCreated,
			OrganizationID: 1,
		}

		err := eb.Publish(context.Background(), event)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if event.SourceInstanceID != eb.instanceID {
			t.Errorf("expected source instance ID %s, got %s", eb.instanceID, event.SourceInstanceID)
		}
	})
}

func TestEventBus_PublishWithHandlers(t *testing.T) {
	t.Run("handlers receive published event", func(t *testing.T) {
		eb := NewEventBus(nil, nil)
		defer eb.Close()

		var received *Event
		var wg sync.WaitGroup
		wg.Add(1)

		eb.Subscribe(EventTicketCreated, func(e *Event) {
			received = e
			wg.Done()
		})

		data, _ := json.Marshal(map[string]string{"title": "Test Ticket"})
		event := &Event{
			Type:           EventTicketCreated,
			OrganizationID: 1,
			EntityType:     "ticket",
			EntityID:       "AM-001",
			Data:           data,
		}

		err := eb.Publish(context.Background(), event)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			if received == nil {
				t.Fatal("handler did not receive event")
			}
			if received.EntityID != "AM-001" {
				t.Errorf("expected EntityID AM-001, got %s", received.EntityID)
			}
		case <-time.After(time.Second):
			t.Error("handler did not receive event within timeout")
		}
	})
}

func TestEventBus_ConcurrentAccess(t *testing.T) {
	eb := NewEventBus(nil, nil)
	defer eb.Close()

	var wg sync.WaitGroup
	numGoroutines := 100

	// Concurrent subscribes
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			eb.Subscribe(EventPodCreated, func(e *Event) {})
		}(i)
	}
	wg.Wait()

	// Concurrent publishes
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			event := &Event{
				Type:           EventPodCreated,
				OrganizationID: int64(idx),
			}
			_ = eb.Publish(context.Background(), event)
		}(i)
	}
	wg.Wait()

	eb.mu.RLock()
	handlers := eb.handlers[EventPodCreated]
	eb.mu.RUnlock()

	if len(handlers) != numGoroutines {
		t.Errorf("expected %d handlers, got %d", numGoroutines, len(handlers))
	}
}
