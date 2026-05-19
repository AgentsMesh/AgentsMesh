package agentpod

type PreparationConfig struct {
	Script  string `json:"script,omitempty"`
	Timeout int    `json:"timeout,omitempty"` // in seconds
}

type CreatePodCommand struct {
	PodKey            string             `json:"pod_id"` // Use pod_id for compatibility with runner
	InitialCommand    string             `json:"initial_command,omitempty"`
	Prompt            string             `json:"prompt,omitempty"`
	PermissionMode    string             `json:"permission_mode,omitempty"`
	TicketSlug        string             `json:"ticket_slug,omitempty"`
	PodSuffix         string             `json:"pod_suffix,omitempty"`
	EnvVars           map[string]string  `json:"env_vars,omitempty"`
	PreparationConfig *PreparationConfig `json:"preparation_config,omitempty"`
}

type TerminatePodCommand struct {
	PodKey string `json:"pod_id"`
}
