package interfaces

// AgentInfo describes an agent for Runner initialization.
type AgentInfo struct {
	Slug          string
	Name          string
	Executable    string
	LaunchCommand string
}

// AgentsProvider provides agent information for initialization handshake.
// This interface is used by both gRPC server and connection manager.
type AgentsProvider interface {
	GetAgentsForRunner() []AgentInfo
}
