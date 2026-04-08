package eventbus

import (
	"sync/atomic"
	"testing"
)

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
			eb2.SubscribeCategory(CategoryEntity, func(e *Event) {})
		}

		eb2.mu.RLock()
		handlers := eb2.categoryHandlers[CategoryEntity]
		eb2.mu.RUnlock()

		if len(handlers) != 2 {
			t.Errorf("expected 2 category handlers, got %d", len(handlers))
		}
	})
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
