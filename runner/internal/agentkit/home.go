package agentkit

// AgentHomeSpec describes how to isolate an agent's home directory per-pod.
type AgentHomeSpec struct {
	EnvVar      string
	UserDirName string
	MergeConfig func(configPath, platformContent string) error
}

var agentHomeSpecs []AgentHomeSpec

// RegisterAgentHome registers an agent home isolation spec.
// Panics if a spec with the same EnvVar is already registered.
func RegisterAgentHome(spec AgentHomeSpec) {
	for _, s := range agentHomeSpecs {
		if s.EnvVar == spec.EnvVar {
			panic("agentkit: duplicate agent home registration for env var: " + spec.EnvVar)
		}
	}
	agentHomeSpecs = append(agentHomeSpecs, spec)
}

// MatchAgentHome finds the first registered spec whose EnvVar is present in envVars.
// Returns the spec and the resolved env value, or nil if no match.
func MatchAgentHome(envVars map[string]string) (*AgentHomeSpec, string) {
	for i := range agentHomeSpecs {
		if v, ok := envVars[agentHomeSpecs[i].EnvVar]; ok && v != "" {
			return &agentHomeSpecs[i], v
		}
	}
	return nil, ""
}
