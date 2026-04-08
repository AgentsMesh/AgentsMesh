package eventbus

import (
	"sync"
)

// EventDefinition contains metadata about an event type
type EventDefinition struct {
	// Type is the event type identifier
	Type EventType
	// Category determines routing (entity=broadcast, notification=targeted)
	Category EventCategory
	// EntityType is the related entity type (pod, ticket, runner, channel, "")
	EntityType string
	// Description is a human-readable description
	Description string
}

// EventRegistry manages event type definitions
type EventRegistry struct {
	definitions map[EventType]*EventDefinition
	mu          sync.RWMutex
}

// NewEventRegistry creates a new event registry
func NewEventRegistry() *EventRegistry {
	r := &EventRegistry{
		definitions: make(map[EventType]*EventDefinition),
	}
	r.registerBuiltinEvents()
	return r
}

// Register registers a new event type definition
func (r *EventRegistry) Register(def *EventDefinition) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.definitions[def.Type] = def
}

// Get returns the definition for an event type
func (r *EventRegistry) Get(eventType EventType) *EventDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.definitions[eventType]
}

// GetCategory returns the category for an event type
func (r *EventRegistry) GetCategory(eventType EventType) EventCategory {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if def, ok := r.definitions[eventType]; ok {
		return def.Category
	}
	// Default to entity category if not registered
	return CategoryEntity
}

// ListByCategory returns all event types in a category
func (r *EventRegistry) ListByCategory(category EventCategory) []EventType {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var types []EventType
	for _, def := range r.definitions {
		if def.Category == category {
			types = append(types, def.Type)
		}
	}
	return types
}

// ListAll returns all registered event types
func (r *EventRegistry) ListAll() []EventType {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]EventType, 0, len(r.definitions))
	for t := range r.definitions {
		types = append(types, t)
	}
	return types
}

// registerBuiltinEvents registers all built-in event types
func (r *EventRegistry) registerBuiltinEvents() {
	// Pod events
	r.definitions[EventPodCreated] = &EventDefinition{
		Type:        EventPodCreated,
		Category:    CategoryEntity,
		EntityType:  "pod",
		Description: "A new pod has been created",
	}
	r.definitions[EventPodStatusChanged] = &EventDefinition{
		Type:        EventPodStatusChanged,
		Category:    CategoryEntity,
		EntityType:  "pod",
		Description: "Pod status has changed",
	}
	r.definitions[EventPodAgentChanged] = &EventDefinition{
		Type:        EventPodAgentChanged,
		Category:    CategoryEntity,
		EntityType:  "pod",
		Description: "Pod agent status has changed",
	}
	r.definitions[EventPodTerminated] = &EventDefinition{
		Type:        EventPodTerminated,
		Category:    CategoryEntity,
		EntityType:  "pod",
		Description: "Pod has been terminated",
	}

	// Ticket events
	r.definitions[EventTicketCreated] = &EventDefinition{
		Type:        EventTicketCreated,
		Category:    CategoryEntity,
		EntityType:  "ticket",
		Description: "A new ticket has been created",
	}
	r.definitions[EventTicketUpdated] = &EventDefinition{
		Type:        EventTicketUpdated,
		Category:    CategoryEntity,
		EntityType:  "ticket",
		Description: "Ticket has been updated",
	}
	r.definitions[EventTicketStatusChanged] = &EventDefinition{
		Type:        EventTicketStatusChanged,
		Category:    CategoryEntity,
		EntityType:  "ticket",
		Description: "Ticket status has changed",
	}
	r.definitions[EventTicketMoved] = &EventDefinition{
		Type:        EventTicketMoved,
		Category:    CategoryEntity,
		EntityType:  "ticket",
		Description: "Ticket has been moved (kanban drag)",
	}
	r.definitions[EventTicketDeleted] = &EventDefinition{
		Type:        EventTicketDeleted,
		Category:    CategoryEntity,
		EntityType:  "ticket",
		Description: "Ticket has been deleted",
	}

	// Runner events
	r.definitions[EventRunnerOnline] = &EventDefinition{
		Type:        EventRunnerOnline,
		Category:    CategoryEntity,
		EntityType:  "runner",
		Description: "Runner has come online",
	}
	r.definitions[EventRunnerOffline] = &EventDefinition{
		Type:        EventRunnerOffline,
		Category:    CategoryEntity,
		EntityType:  "runner",
		Description: "Runner has gone offline",
	}
	r.definitions[EventRunnerUpdated] = &EventDefinition{
		Type:        EventRunnerUpdated,
		Category:    CategoryEntity,
		EntityType:  "runner",
		Description: "Runner has been updated",
	}

	// System events
	r.definitions[EventSystemMaintenance] = &EventDefinition{
		Type:        EventSystemMaintenance,
		Category:    CategorySystem,
		EntityType:  "",
		Description: "System maintenance notification",
	}

	// AutopilotController events
	r.definitions[EventAutopilotStatusChanged] = &EventDefinition{
		Type:        EventAutopilotStatusChanged,
		Category:    CategoryEntity,
		EntityType:  "autopilot_controller",
		Description: "AutopilotController status has changed",
	}
	r.definitions[EventAutopilotIteration] = &EventDefinition{
		Type:        EventAutopilotIteration,
		Category:    CategoryEntity,
		EntityType:  "autopilot_controller",
		Description: "AutopilotController iteration event",
	}
	r.definitions[EventAutopilotCreated] = &EventDefinition{
		Type:        EventAutopilotCreated,
		Category:    CategoryEntity,
		EntityType:  "autopilot_controller",
		Description: "A new AutopilotController has been created",
	}
	r.definitions[EventAutopilotTerminated] = &EventDefinition{
		Type:        EventAutopilotTerminated,
		Category:    CategoryEntity,
		EntityType:  "autopilot_controller",
		Description: "AutopilotController has been terminated",
	}
	r.definitions[EventAutopilotThinking] = &EventDefinition{
		Type:        EventAutopilotThinking,
		Category:    CategoryEntity,
		EntityType:  "autopilot_controller",
		Description: "AutopilotController Control Agent thinking event",
	}
}

// DefaultRegistry is the global default event registry
var DefaultRegistry = NewEventRegistry()
