package agentpod

// PreparationConfig holds the preparation script configuration
type PreparationConfig struct {
	Script  string `json:"script,omitempty"`
	Timeout int    `json:"timeout,omitempty"` // in seconds
}

// CreatePodCommand represents a command to create a pod on a runner
type CreatePodCommand struct {
	PodKey            string             `json:"pod_id"` // Use pod_id for compatibility with runner
	InitialCommand    string             `json:"initial_command,omitempty"`
	InitialPrompt     string             `json:"initial_prompt,omitempty"`
	PermissionMode    string             `json:"permission_mode,omitempty"`
	TicketSlug        string             `json:"ticket_slug,omitempty"`
	PodSuffix         string             `json:"pod_suffix,omitempty"`
	EnvVars           map[string]string  `json:"env_vars,omitempty"`
	PreparationConfig *PreparationConfig `json:"preparation_config,omitempty"`
}

// TerminatePodCommand represents a command to terminate a pod
type TerminatePodCommand struct {
	PodKey string `json:"pod_id"`
}
