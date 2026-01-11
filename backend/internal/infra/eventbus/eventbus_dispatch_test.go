package eventbus

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestEventBus_dispatchLocal(t *testing.T) {
	t.Run("dispatches to type-specific handlers", func(t *testing.T) {
		eb := NewEventBus(nil, nil)
		defer eb.Close()

		var received *Event
		var wg sync.WaitGroup
		wg.Add(1)

		eb.Subscribe(EventPodCreated, func(e *Event) {
			received = e
			wg.Done()
		})

		event := &Event{
			Type:           EventPodCreated,
			Category:       CategoryEntity,
			OrganizationID: 1,
			EntityID:       "pod-123",
		}

		eb.dispatchLocal(event)

		// Wait with timeout
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			if received == nil {
				t.Error("handler was not called")
			} else if received.EntityID != "pod-123" {
				t.Errorf("expected EntityID pod-123, got %s", received.EntityID)
			}
		case <-time.After(time.Second):
			t.Error("handler was not called within timeout")
		}
	})

	t.Run("dispatches to category handlers", func(t *testing.T) {
		eb := NewEventBus(nil, nil)
		defer eb.Close()

		var received *Event
		var wg sync.WaitGroup
		wg.Add(1)

		eb.SubscribeCategory(CategoryEntity, func(e *Event) {
			received = e
			wg.Done()
		})

		event := &Event{
			Type:           EventTicketUpdated,
			Category:       CategoryEntity,
			OrganizationID: 1,
			EntityID:       "ticket-456",
		}

		eb.dispatchLocal(event)

		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			if received == nil {
				t.Error("category handler was not called")
			} else if received.EntityID != "ticket-456" {
				t.Errorf("expected EntityID ticket-456, got %s", received.EntityID)
			}
		case <-time.After(time.Second):
			t.Error("category handler was not called within timeout")
		}
	})

	t.Run("dispatches to both type and category handlers", func(t *testing.T) {
		eb := NewEventBus(nil, nil)
		defer eb.Close()

		var typeHandlerCalled, categoryHandlerCalled int32
		var wg sync.WaitGroup
		wg.Add(2)

		eb.Subscribe(EventRunnerOnline, func(e *Event) {
			atomic.StoreInt32(&typeHandlerCalled, 1)
			wg.Done()
		})

		eb.SubscribeCategory(CategoryEntity, func(e *Event) {
			atomic.StoreInt32(&categoryHandlerCalled, 1)
			wg.Done()
		})

		event := &Event{
			Type:           EventRunnerOnline,
			Category:       CategoryEntity,
			OrganizationID: 1,
		}

		eb.dispatchLocal(event)

		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			if atomic.LoadInt32(&typeHandlerCalled) != 1 {
				t.Error("type handler was not called")
			}
			if atomic.LoadInt32(&categoryHandlerCalled) != 1 {
				t.Error("category handler was not called")
			}
		case <-time.After(time.Second):
			t.Error("handlers were not called within timeout")
		}
	})

	t.Run("no handlers for event type - no error", func(t *testing.T) {
		eb := NewEventBus(nil, nil)
		defer eb.Close()

		event := &Event{
			Type:           EventPodCreated,
			Category:       CategoryEntity,
			OrganizationID: 1,
		}

		// Should not panic
		eb.dispatchLocal(event)
	})
}

func TestEventBus_safeCallHandler(t *testing.T) {
	t.Run("recovers from panic", func(t *testing.T) {
		eb := NewEventBus(nil, nil)
		defer eb.Close()

		var wg sync.WaitGroup
		wg.Add(1)

		// This handler panics
		panicHandler := func(e *Event) {
			defer wg.Done()
			panic("test panic")
		}

		event := &Event{
			Type:           EventPodCreated,
			Category:       CategoryEntity,
			OrganizationID: 1,
			EntityType:     "pod",
			EntityID:       "pod-123",
		}

		// Should not panic
		eb.safeCallHandler(panicHandler, event)

		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			// Success - panic was recovered
		case <-time.After(time.Second):
			t.Error("handler did not complete within timeout")
		}
	})

	t.Run("normal handler executes successfully", func(t *testing.T) {
		eb := NewEventBus(nil, nil)
		defer eb.Close()

		var called bool
		handler := func(e *Event) {
			called = true
		}

		event := &Event{
			Type:           EventPodCreated,
			Category:       CategoryEntity,
			OrganizationID: 1,
		}

		eb.safeCallHandler(handler, event)

		if !called {
			t.Error("handler was not called")
		}
	})
}
