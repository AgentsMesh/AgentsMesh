package agentkit

var processNameSet = map[string]bool{
	"node": true,
}

// RegisterProcessNames registers process names that identify an agent CLI.
// Panics if a name is already registered by another agent.
func RegisterProcessNames(names ...string) {
	for _, n := range names {
		if processNameSet[n] {
			panic("agentkit: duplicate process name registration: " + n)
		}
		processNameSet[n] = true
	}
}

// IsAgentProcess returns true if the process name belongs to a registered agent.
func IsAgentProcess(name string) bool {
	return processNameSet[name]
}
