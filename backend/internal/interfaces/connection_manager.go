package interfaces

// AgentTypeInfo describes an agent type for Runner initialization.
type AgentTypeInfo struct {
	Slug          string
	Name          string
	Executable    string
	LaunchCommand string
}

// AgentTypesProvider provides agent type information for initialization handshake.
// This interface is used by both gRPC server and connection manager.
type AgentTypesProvider interface {
	GetAgentTypesForRunner() []AgentTypeInfo
}
