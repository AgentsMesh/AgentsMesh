package eventbus

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewEventBus(t *testing.T) {
	t.Run("with nil redis client and nil logger", func(t *testing.T) {
		eb := NewEventBus(nil, nil)
		if eb == nil {
			t.Fatal("expected non-nil EventBus")
		}
		if eb.logger == nil {
			t.Error("expected default logger to be set")
		}
		if eb.instanceID == "" {
			t.Error("expected instanceID to be generated")
		}
		if eb.handlers == nil {
			t.Error("expected handlers map to be initialized")
		}
		if eb.categoryHandlers == nil {
			t.Error("expected categoryHandlers map to be initialized")
		}
		if eb.subscribedOrgs == nil {
			t.Error("expected subscribedOrgs map to be initialized")
		}
		eb.Close()
	})

	t.Run("instanceID format contains hostname", func(t *testing.T) {
		eb := NewEventBus(nil, nil)
		defer eb.Close()

		// instanceID format: hostname-uuid8chars
		if len(eb.instanceID) < 10 {
			t.Errorf("instanceID too short: %s", eb.instanceID)
		}
	})
}

func TestEventBus_Subscribe(t *testing.T) {
	eb := NewEventBus(nil, nil)
	defer eb.Close()

	t.Run("subscribe single handler", func(t *testing.T) {
		eb.Subscribe(EventPodCreated, func(e *Event) {
			// Handler registered
		})

		eb.mu.RLock()
		handlers := eb.handlers[EventPodCreated]
		eb.mu.RUnlock()

		if len(handlers) != 1 {
			t.Errorf("expected 1 handler, got %d", len(handlers))
		}
	})

	t.Run("subscribe multiple handlers to same event", func(t *testing.T) {
		eb2 := NewEventBus(nil, nil)
		defer eb2.Close()

		var count int32
		for i := 0; i < 3; i++ {
			eb2.Subscribe(EventTicketCreated, func(e *Event) {
				atomic.AddInt32(&count, 1)
			})
		}

		eb2.mu.RLock()
		handlers := eb2.handlers[EventTicketCreated]
		eb2.mu.RUnlock()

		if len(handlers) != 3 {
			t.Errorf("expected 3 handlers, got %d", len(handlers))
		}
	})
}

func TestEventBus_SubscribeCategory(t *testing.T) {
	eb := NewEventBus(nil, nil)
	defer eb.Close()

	t.Run("subscribe to category", func(t *testing.T) {
		eb.SubscribeCategory(CategoryEntity, func(e *Event) {
			// Category handler registered
		})

		eb.mu.RLock()
		handlers := eb.categoryHandlers[CategoryEntity]
		eb.mu.RUnlock()

		if len(handlers) != 1 {
			t.Errorf("expected 1 category handler, got %d", len(handlers))
		}
	})

	t.Run("subscribe multiple handlers to same category", func(t *testing.T) {
		eb2 := NewEventBus(nil, nil)
		defer eb2.Close()

		for i := 0; i < 2; i++ {
			eb2.SubscribeCategory(CategoryNotification, func(e *Event) {})
		}

		eb2.mu.RLock()
		handlers := eb2.categoryHandlers[CategoryNotification]
		eb2.mu.RUnlock()

		if len(handlers) != 2 {
			t.Errorf("expected 2 category handlers, got %d", len(handlers))
		}
	})
}

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
			Type:           EventTerminalNotification, // Notification category
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

func TestEventBus_SubscribeOrg(t *testing.T) {
	t.Run("tracks subscribed orgs without redis", func(t *testing.T) {
		eb := NewEventBus(nil, nil)
		defer eb.Close()

		eb.SubscribeOrg(123)

		eb.orgsMu.RLock()
		subscribed := eb.subscribedOrgs[123]
		eb.orgsMu.RUnlock()

		if !subscribed {
			t.Error("org 123 should be subscribed")
		}
	})

	t.Run("idempotent subscription", func(t *testing.T) {
		eb := NewEventBus(nil, nil)
		defer eb.Close()

		eb.SubscribeOrg(456)
		eb.SubscribeOrg(456) // Subscribe again

		eb.orgsMu.RLock()
		subscribed := eb.subscribedOrgs[456]
		eb.orgsMu.RUnlock()

		if !subscribed {
			t.Error("org 456 should still be subscribed")
		}
	})
}

func TestEventBus_UnsubscribeOrg(t *testing.T) {
	eb := NewEventBus(nil, nil)
	defer eb.Close()

	eb.SubscribeOrg(789)
	eb.UnsubscribeOrg(789)

	eb.orgsMu.RLock()
	subscribed := eb.subscribedOrgs[789]
	eb.orgsMu.RUnlock()

	if subscribed {
		t.Error("org 789 should be unsubscribed")
	}
}

func TestEventBus_Registry(t *testing.T) {
	eb := NewEventBus(nil, nil)
	defer eb.Close()

	registry := eb.Registry()
	if registry == nil {
		t.Fatal("expected non-nil registry")
	}

	// Verify it's the default registry
	if registry != DefaultRegistry {
		t.Error("expected default registry")
	}
}

func TestEventBus_Close(t *testing.T) {
	eb := NewEventBus(nil, nil)

	// Subscribe a handler
	eb.Subscribe(EventPodCreated, func(e *Event) {})

	// Close should not panic
	eb.Close()

	// Verify context is cancelled
	select {
	case <-eb.ctx.Done():
		// Success - context was cancelled
	default:
		t.Error("context should be cancelled after Close()")
	}
}

func TestEventBus_redisChannel(t *testing.T) {
	eb := NewEventBus(nil, nil)
	defer eb.Close()

	channel := eb.redisChannel(42)
	expected := "events:org:42"

	if channel != expected {
		t.Errorf("expected channel %s, got %s", expected, channel)
	}
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
