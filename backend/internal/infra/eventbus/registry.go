package eventbus

import (
	"sync"
)

type EventDefinition struct {
	Type EventType
	Category EventCategory
	EntityType string
	Description string
}

type EventRegistry struct {
	definitions map[EventType]*EventDefinition
	mu          sync.RWMutex
}

func NewEventRegistry() *EventRegistry {
	r := &EventRegistry{
		definitions: make(map[EventType]*EventDefinition),
	}
	r.registerBuiltinEvents()
	return r
}

func (r *EventRegistry) Register(def *EventDefinition) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.definitions[def.Type] = def
}

func (r *EventRegistry) Get(eventType EventType) *EventDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.definitions[eventType]
}

func (r *EventRegistry) GetCategory(eventType EventType) EventCategory {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if def, ok := r.definitions[eventType]; ok {
		return def.Category
	}
	return CategoryEntity
}

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

func (r *EventRegistry) ListAll() []EventType {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]EventType, 0, len(r.definitions))
	for t := range r.definitions {
		types = append(types, t)
	}
	return types
}

func (r *EventRegistry) registerBuiltinEvents() {
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
	r.definitions[EventPodRestarting] = &EventDefinition{
		Type:        EventPodRestarting,
		Category:    CategoryEntity,
		EntityType:  "pod",
		Description: "Perpetual pod is restarting",
	}

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

	r.definitions[EventSystemMaintenance] = &EventDefinition{
		Type:        EventSystemMaintenance,
		Category:    CategorySystem,
		EntityType:  "",
		Description: "System maintenance notification",
	}

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

var DefaultRegistry = NewEventRegistry()
