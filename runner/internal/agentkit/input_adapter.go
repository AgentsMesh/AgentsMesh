package agentkit

// TerminalInputAdapter adapts raw terminal input for a specific agent's TUI.
type TerminalInputAdapter interface {
	Adapt(data []byte) []byte
}

var inputAdapterRegistry = map[string]TerminalInputAdapter{}

// RegisterInputAdapter registers a TerminalInputAdapter for an agent type.
// Panics if the agent type is already registered.
func RegisterInputAdapter(agentType string, adapter TerminalInputAdapter) {
	if _, exists := inputAdapterRegistry[agentType]; exists {
		panic("agentkit: duplicate input adapter registration: " + agentType)
	}
	inputAdapterRegistry[agentType] = adapter
}

// AdaptTerminalInput looks up the adapter for the given agent type and applies it.
func AdaptTerminalInput(data []byte, agentType string) []byte {
	if adapter, ok := inputAdapterRegistry[agentType]; ok {
		return adapter.Adapt(data)
	}
	return data
}
