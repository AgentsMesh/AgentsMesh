// Package podfile provides the top-level types for PodFile processing results.
package podfile

// PodSpec is the Backend-facing output of PodFile declaration extraction.
// It contains everything the frontend needs to render Pod creation UI:
// agent info, config form fields, credential fields, skills, etc.
//
// PodSpec is produced by extract.Extract() and is JSON-serializable.
type PodSpec struct {
	Agent       AgentSpec       `json:"agent"`
	Config      []ConfigSpec    `json:"config,omitempty"`
	Env         []EnvSpec       `json:"env,omitempty"`
	Repo        *RepoSpec       `json:"repo,omitempty"`
	MCP         *MCPSpec        `json:"mcp,omitempty"`
	Skills      []string        `json:"skills,omitempty"`
	Setup       *SetupSpec      `json:"setup,omitempty"`
}

// AgentSpec describes which agent CLI to use.
type AgentSpec struct {
	Command    string `json:"command"`
	Executable string `json:"executable,omitempty"`
}

// ConfigSpec describes a user-configurable parameter (→ UI form field).
type ConfigSpec struct {
	Name     string      `json:"name"`
	Type     string      `json:"type"` // boolean, string, number, secret, select
	Default  interface{} `json:"default,omitempty"`
	Options  []string    `json:"options,omitempty"` // for select type
}

// EnvSpec describes an environment variable declaration.
type EnvSpec struct {
	Name     string `json:"name"`
	Source   string `json:"source,omitempty"` // "secret" or "text" (credential)
	Value    string `json:"value,omitempty"`  // fixed value (mutually exclusive with Source)
	Optional bool   `json:"optional,omitempty"`
}

// RepoSpec describes the default repository configuration.
type RepoSpec struct {
	URL            string `json:"url,omitempty"`
	Branch         string `json:"branch,omitempty"`
	CredentialType string `json:"credential_type,omitempty"`
}

// MCPSpec describes MCP configuration.
type MCPSpec struct {
	Enabled bool `json:"enabled"`
}

// SetupSpec describes the workspace preparation script.
type SetupSpec struct {
	Script  string `json:"script"`
	Timeout int    `json:"timeout"`
}
