package ticket

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/infra/eventbus"
)

func TestNewEventBusPublisher(t *testing.T) {
	t.Run("creates publisher with nil eventbus", func(t *testing.T) {
		publisher := NewEventBusPublisher(nil, nil)
		if publisher == nil {
			t.Fatal("expected non-nil publisher")
		}
		if publisher.eventBus != nil {
			t.Error("expected nil eventBus")
		}
		if publisher.logger == nil {
			t.Error("expected default logger to be set")
		}
	})

	t.Run("creates publisher with eventbus", func(t *testing.T) {
		eb := eventbus.NewEventBus(nil, nil)
		defer eb.Close()

		publisher := NewEventBusPublisher(eb, nil)
		if publisher == nil {
			t.Fatal("expected non-nil publisher")
		}
		if publisher.eventBus != eb {
			t.Error("expected eventBus to be set")
		}
	})
}

func TestEventBusPublisher_PublishTicketEvent(t *testing.T) {
	t.Run("publishes TicketEventCreated", func(t *testing.T) {
		eb := eventbus.NewEventBus(nil, nil)
		defer eb.Close()

		var received *eventbus.Event
		var wg sync.WaitGroup
		wg.Add(1)

		eb.Subscribe(eventbus.EventTicketCreated, func(e *eventbus.Event) {
			received = e
			wg.Done()
		})

		publisher := NewEventBusPublisher(eb, nil)
		publisher.PublishTicketEvent(context.Background(), TicketEventCreated, 1, "AM-001", "backlog", "")

		waitWithTimeout(t, &wg, time.Second)

		if received == nil {
			t.Fatal("event not received")
		}
		if received.Type != eventbus.EventTicketCreated {
			t.Errorf("expected type %s, got %s", eventbus.EventTicketCreated, received.Type)
		}
		if received.OrganizationID != 1 {
			t.Errorf("expected org ID 1, got %d", received.OrganizationID)
		}
		if received.EntityType != "ticket" {
			t.Errorf("expected entity type 'ticket', got '%s'", received.EntityType)
		}
		if received.EntityID != "AM-001" {
			t.Errorf("expected entity ID 'AM-001', got '%s'", received.EntityID)
		}
	})

	t.Run("publishes TicketEventUpdated", func(t *testing.T) {
		eb := eventbus.NewEventBus(nil, nil)
		defer eb.Close()

		var received *eventbus.Event
		var wg sync.WaitGroup
		wg.Add(1)

		eb.Subscribe(eventbus.EventTicketUpdated, func(e *eventbus.Event) {
			received = e
			wg.Done()
		})

		publisher := NewEventBusPublisher(eb, nil)
		publisher.PublishTicketEvent(context.Background(), TicketEventUpdated, 2, "AM-002", "in_progress", "backlog")

		waitWithTimeout(t, &wg, time.Second)

		if received == nil {
			t.Fatal("event not received")
		}
		if received.Type != eventbus.EventTicketUpdated {
			t.Errorf("expected type %s, got %s", eventbus.EventTicketUpdated, received.Type)
		}
	})

	t.Run("publishes TicketEventStatusChanged", func(t *testing.T) {
		eb := eventbus.NewEventBus(nil, nil)
		defer eb.Close()

		var received *eventbus.Event
		var wg sync.WaitGroup
		wg.Add(1)

		eb.Subscribe(eventbus.EventTicketStatusChanged, func(e *eventbus.Event) {
			received = e
			wg.Done()
		})

		publisher := NewEventBusPublisher(eb, nil)
		publisher.PublishTicketEvent(context.Background(), TicketEventStatusChanged, 3, "AM-003", "done", "in_progress")

		waitWithTimeout(t, &wg, time.Second)

		if received == nil {
			t.Fatal("event not received")
		}
		if received.Type != eventbus.EventTicketStatusChanged {
			t.Errorf("expected type %s, got %s", eventbus.EventTicketStatusChanged, received.Type)
		}
	})

	t.Run("publishes TicketEventMoved", func(t *testing.T) {
		eb := eventbus.NewEventBus(nil, nil)
		defer eb.Close()

		var received *eventbus.Event
		var wg sync.WaitGroup
		wg.Add(1)

		eb.Subscribe(eventbus.EventTicketMoved, func(e *eventbus.Event) {
			received = e
			wg.Done()
		})

		publisher := NewEventBusPublisher(eb, nil)
		publisher.PublishTicketEvent(context.Background(), TicketEventMoved, 4, "AM-004", "review", "in_progress")

		waitWithTimeout(t, &wg, time.Second)

		if received == nil {
			t.Fatal("event not received")
		}
		if received.Type != eventbus.EventTicketMoved {
			t.Errorf("expected type %s, got %s", eventbus.EventTicketMoved, received.Type)
		}
	})

	t.Run("publishes TicketEventDeleted", func(t *testing.T) {
		eb := eventbus.NewEventBus(nil, nil)
		defer eb.Close()

		var received *eventbus.Event
		var wg sync.WaitGroup
		wg.Add(1)

		eb.Subscribe(eventbus.EventTicketDeleted, func(e *eventbus.Event) {
			received = e
			wg.Done()
		})

		publisher := NewEventBusPublisher(eb, nil)
		publisher.PublishTicketEvent(context.Background(), TicketEventDeleted, 5, "AM-005", "", "done")

		waitWithTimeout(t, &wg, time.Second)

		if received == nil {
			t.Fatal("event not received")
		}
		if received.Type != eventbus.EventTicketDeleted {
			t.Errorf("expected type %s, got %s", eventbus.EventTicketDeleted, received.Type)
		}
	})

	t.Run("handles nil eventbus gracefully", func(t *testing.T) {
		publisher := NewEventBusPublisher(nil, nil)

		// Should not panic
		publisher.PublishTicketEvent(context.Background(), TicketEventCreated, 1, "AM-001", "backlog", "")
	})

	t.Run("handles unknown event type gracefully", func(t *testing.T) {
		eb := eventbus.NewEventBus(nil, nil)
		defer eb.Close()

		publisher := NewEventBusPublisher(eb, nil)

		// Unknown event type (not in the switch)
		unknownType := TicketEventType(999)

		// Should not panic, just log warning
		publisher.PublishTicketEvent(context.Background(), unknownType, 1, "AM-001", "backlog", "")
	})
}

func TestEventBusPublisher_EventDataPayload(t *testing.T) {
	t.Run("event contains correct data payload", func(t *testing.T) {
		eb := eventbus.NewEventBus(nil, nil)
		defer eb.Close()

		var received *eventbus.Event
		var wg sync.WaitGroup
		wg.Add(1)

		eb.Subscribe(eventbus.EventTicketStatusChanged, func(e *eventbus.Event) {
			received = e
			wg.Done()
		})

		publisher := NewEventBusPublisher(eb, nil)
		publisher.PublishTicketEvent(context.Background(), TicketEventStatusChanged, 10, "PRJ-100", "done", "in_progress")

		waitWithTimeout(t, &wg, time.Second)

		if received == nil {
			t.Fatal("event not received")
		}

		// Verify data payload
		if received.Data == nil {
			t.Fatal("expected non-nil data")
		}

		// Check that it's valid JSON containing expected fields
		dataStr := string(received.Data)
		if !containsStr(dataStr, "PRJ-100") {
			t.Errorf("data should contain identifier 'PRJ-100', got: %s", dataStr)
		}
		if !containsStr(dataStr, "done") {
			t.Errorf("data should contain status 'done', got: %s", dataStr)
		}
		if !containsStr(dataStr, "in_progress") {
			t.Errorf("data should contain previous status 'in_progress', got: %s", dataStr)
		}
	})
}

func TestTicketEventType_Constants(t *testing.T) {
	// Verify event type constants have expected iota values
	if TicketEventCreated != 0 {
		t.Errorf("expected TicketEventCreated=0, got %d", TicketEventCreated)
	}
	if TicketEventUpdated != 1 {
		t.Errorf("expected TicketEventUpdated=1, got %d", TicketEventUpdated)
	}
	if TicketEventStatusChanged != 2 {
		t.Errorf("expected TicketEventStatusChanged=2, got %d", TicketEventStatusChanged)
	}
	if TicketEventMoved != 3 {
		t.Errorf("expected TicketEventMoved=3, got %d", TicketEventMoved)
	}
	if TicketEventDeleted != 4 {
		t.Errorf("expected TicketEventDeleted=4, got %d", TicketEventDeleted)
	}
}

func TestEventBusPublisher_AllEventTypesMapping(t *testing.T) {
	// Test that all TicketEventTypes map correctly to eventbus.EventTypes
	testCases := []struct {
		ticketEventType TicketEventType
		expectedType    eventbus.EventType
	}{
		{TicketEventCreated, eventbus.EventTicketCreated},
		{TicketEventUpdated, eventbus.EventTicketUpdated},
		{TicketEventStatusChanged, eventbus.EventTicketStatusChanged},
		{TicketEventMoved, eventbus.EventTicketMoved},
		{TicketEventDeleted, eventbus.EventTicketDeleted},
	}

	for _, tc := range testCases {
		t.Run(string(tc.expectedType), func(t *testing.T) {
			eb := eventbus.NewEventBus(nil, nil)
			defer eb.Close()

			var received *eventbus.Event
			var wg sync.WaitGroup
			wg.Add(1)

			eb.Subscribe(tc.expectedType, func(e *eventbus.Event) {
				received = e
				wg.Done()
			})

			publisher := NewEventBusPublisher(eb, nil)
			publisher.PublishTicketEvent(context.Background(), tc.ticketEventType, 1, "TEST-001", "status", "prev_status")

			waitWithTimeout(t, &wg, time.Second)

			if received == nil {
				t.Fatalf("event not received for %s", tc.expectedType)
			}
			if received.Type != tc.expectedType {
				t.Errorf("expected type %s, got %s", tc.expectedType, received.Type)
			}
		})
	}
}

func TestEventBusPublisher_ConcurrentPublish(t *testing.T) {
	eb := eventbus.NewEventBus(nil, nil)
	defer eb.Close()

	var receivedCount int32
	var mu sync.Mutex
	var wg sync.WaitGroup

	numEvents := 100

	eb.Subscribe(eventbus.EventTicketCreated, func(e *eventbus.Event) {
		mu.Lock()
		receivedCount++
		mu.Unlock()
	})

	publisher := NewEventBusPublisher(eb, nil)

	wg.Add(numEvents)
	for i := 0; i < numEvents; i++ {
		go func(idx int) {
			defer wg.Done()
			publisher.PublishTicketEvent(context.Background(), TicketEventCreated, int64(idx), "AM-"+string(rune('A'+idx%26)), "backlog", "")
		}(i)
	}

	wg.Wait()

	// Wait a bit for all handlers to complete
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	count := receivedCount
	mu.Unlock()

	if count != int32(numEvents) {
		t.Errorf("expected %d events received, got %d", numEvents, count)
	}
}

// Helper function to wait with timeout
func waitWithTimeout(t *testing.T, wg *sync.WaitGroup, timeout time.Duration) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(timeout):
		t.Fatal("timeout waiting for event")
	}
}

// Helper function to check if string contains substring
func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
