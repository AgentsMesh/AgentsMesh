package interfaces

type AgentInfo struct {
	Slug          string
	Name          string
	Executable    string
	LaunchCommand string
}

type AgentsProvider interface {
	GetAgentsForRunner() []AgentInfo
}
