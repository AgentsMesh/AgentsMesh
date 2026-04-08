package eventbus

import (
	"sync"
	"testing"
)

func TestNewEventRegistry(t *testing.T) {
	t.Run("creates registry with builtin events", func(t *testing.T) {
		r := NewEventRegistry()
		if r == nil {
			t.Fatal("expected non-nil registry")
		}
		if r.definitions == nil {
			t.Error("expected definitions map to be initialized")
		}

		// Verify some builtin events are registered
		builtinEvents := []EventType{
			EventPodCreated,
			EventPodStatusChanged,
			EventTicketCreated,
			EventRunnerOnline,
			EventSystemMaintenance,
		}

		for _, et := range builtinEvents {
			if def := r.Get(et); def == nil {
				t.Errorf("expected builtin event %s to be registered", et)
			}
		}
	})

	t.Run("builtin events have correct categories", func(t *testing.T) {
		r := NewEventRegistry()

		testCases := []struct {
			eventType EventType
			category  EventCategory
		}{
			{EventPodCreated, CategoryEntity},
			{EventPodStatusChanged, CategoryEntity},
			{EventTicketCreated, CategoryEntity},
			{EventRunnerOnline, CategoryEntity},
			{EventSystemMaintenance, CategorySystem},
		}

		for _, tc := range testCases {
			def := r.Get(tc.eventType)
			if def == nil {
				t.Errorf("event %s not found", tc.eventType)
				continue
			}
			if def.Category != tc.category {
				t.Errorf("event %s: expected category %s, got %s", tc.eventType, tc.category, def.Category)
			}
		}
	})

	t.Run("builtin events have correct entity types", func(t *testing.T) {
		r := NewEventRegistry()

		testCases := []struct {
			eventType  EventType
			entityType string
		}{
			{EventPodCreated, "pod"},
			{EventPodStatusChanged, "pod"},
			{EventTicketCreated, "ticket"},
			{EventTicketUpdated, "ticket"},
			{EventRunnerOnline, "runner"},
			{EventSystemMaintenance, ""},
		}

		for _, tc := range testCases {
			def := r.Get(tc.eventType)
			if def == nil {
				t.Errorf("event %s not found", tc.eventType)
				continue
			}
			if def.EntityType != tc.entityType {
				t.Errorf("event %s: expected entity type %s, got %s", tc.eventType, tc.entityType, def.EntityType)
			}
		}
	})
}

func TestEventRegistry_Register(t *testing.T) {
	t.Run("register new event type", func(t *testing.T) {
		r := NewEventRegistry()

		customEvent := EventType("custom:event")
		def := &EventDefinition{
			Type:        customEvent,
			Category:    CategoryEntity,
			EntityType:  "custom",
			Description: "A custom event for testing",
		}

		r.Register(def)

		retrieved := r.Get(customEvent)
		if retrieved == nil {
			t.Fatal("custom event not found after registration")
		}
		if retrieved.Type != customEvent {
			t.Errorf("expected type %s, got %s", customEvent, retrieved.Type)
		}
		if retrieved.Description != "A custom event for testing" {
			t.Errorf("expected description 'A custom event for testing', got '%s'", retrieved.Description)
		}
	})

	t.Run("override existing event definition", func(t *testing.T) {
		r := NewEventRegistry()

		// Override builtin event
		newDef := &EventDefinition{
			Type:        EventPodCreated,
			Category:    CategorySystem, // Change category
			EntityType:  "custom_pod",
			Description: "Overridden pod created event",
		}

		r.Register(newDef)

		retrieved := r.Get(EventPodCreated)
		if retrieved.Category != CategorySystem {
			t.Errorf("expected category %s, got %s", CategorySystem, retrieved.Category)
		}
		if retrieved.EntityType != "custom_pod" {
			t.Errorf("expected entity type custom_pod, got %s", retrieved.EntityType)
		}
	})
}

func TestEventRegistry_Get(t *testing.T) {
	r := NewEventRegistry()

	t.Run("get existing event", func(t *testing.T) {
		def := r.Get(EventPodCreated)
		if def == nil {
			t.Fatal("expected non-nil definition")
		}
		if def.Type != EventPodCreated {
			t.Errorf("expected type %s, got %s", EventPodCreated, def.Type)
		}
	})

	t.Run("get non-existing event returns nil", func(t *testing.T) {
		def := r.Get(EventType("nonexistent:event"))
		if def != nil {
			t.Error("expected nil for non-existing event")
		}
	})
}

func TestEventRegistry_GetCategory(t *testing.T) {
	r := NewEventRegistry()

	t.Run("get category for entity event", func(t *testing.T) {
		category := r.GetCategory(EventPodCreated)
		if category != CategoryEntity {
			t.Errorf("expected %s, got %s", CategoryEntity, category)
		}
	})

	t.Run("get category for system event", func(t *testing.T) {
		category := r.GetCategory(EventSystemMaintenance)
		if category != CategorySystem {
			t.Errorf("expected %s, got %s", CategorySystem, category)
		}
	})

	t.Run("get category for non-existing event returns default", func(t *testing.T) {
		category := r.GetCategory(EventType("nonexistent:event"))
		if category != CategoryEntity {
			t.Errorf("expected default category %s, got %s", CategoryEntity, category)
		}
	})
}

func TestEventRegistry_ListByCategory(t *testing.T) {
	r := NewEventRegistry()

	t.Run("list entity events", func(t *testing.T) {
		types := r.ListByCategory(CategoryEntity)
		if len(types) == 0 {
			t.Error("expected non-empty list of entity events")
		}

		// Verify all returned events are entity category
		for _, et := range types {
			def := r.Get(et)
			if def == nil {
				t.Errorf("event %s not found", et)
				continue
			}
			if def.Category != CategoryEntity {
				t.Errorf("event %s has category %s, expected %s", et, def.Category, CategoryEntity)
			}
		}
	})

	t.Run("list events for non-existing category returns empty", func(t *testing.T) {
		types := r.ListByCategory(EventCategory("nonexistent"))
		if len(types) != 0 {
			t.Errorf("expected empty list, got %d events", len(types))
		}
	})
}

func TestEventRegistry_ListAll(t *testing.T) {
	r := NewEventRegistry()

	t.Run("returns all registered events", func(t *testing.T) {
		types := r.ListAll()
		if len(types) == 0 {
			t.Error("expected non-empty list of events")
		}

		// Count expected builtin events (based on registerBuiltinEvents)
		// Pod: 7, Channel: 3, Ticket: 5, Runner: 3 = 18 (entity)
		// Autopilot+MR+Pipeline+Loop registered in types but check actual count
		expectedCount := 18

		if len(types) != expectedCount {
			t.Errorf("expected %d events, got %d", expectedCount, len(types))
		}
	})

	t.Run("includes custom registered events", func(t *testing.T) {
		r := NewEventRegistry()
		customEvent := EventType("custom:test")
		r.Register(&EventDefinition{
			Type:     customEvent,
			Category: CategoryEntity,
		})

		types := r.ListAll()
		found := false
		for _, et := range types {
			if et == customEvent {
				found = true
				break
			}
		}

		if !found {
			t.Error("custom event not found in ListAll()")
		}
	})
}

func TestEventRegistry_ConcurrentAccess(t *testing.T) {
	r := NewEventRegistry()
	var wg sync.WaitGroup
	numGoroutines := 100

	// Concurrent reads
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			_ = r.Get(EventPodCreated)
			_ = r.GetCategory(EventPodCreated)
			_ = r.ListByCategory(CategoryEntity)
			_ = r.ListAll()
		}()
	}
	wg.Wait()

	// Concurrent writes
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			r.Register(&EventDefinition{
				Type:     EventType("concurrent:event:" + string(rune(idx))),
				Category: CategoryEntity,
			})
		}(i)
	}
	wg.Wait()

	// Concurrent reads and writes
	wg.Add(numGoroutines * 2)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			_ = r.Get(EventPodCreated)
			_ = r.ListByCategory(CategoryEntity)
		}()
		go func(idx int) {
			defer wg.Done()
			r.Register(&EventDefinition{
				Type:     EventType("mixed:event:" + string(rune(idx))),
				Category: CategoryEntity,
			})
		}(i)
	}
	wg.Wait()
}

func TestDefaultRegistry(t *testing.T) {
	t.Run("is initialized", func(t *testing.T) {
		if DefaultRegistry == nil {
			t.Fatal("DefaultRegistry should be initialized")
		}
	})

	t.Run("contains builtin events", func(t *testing.T) {
		def := DefaultRegistry.Get(EventPodCreated)
		if def == nil {
			t.Error("DefaultRegistry should contain EventPodCreated")
		}
	})
}
