package agent

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
)

// AgentConfigProvider provides agent configuration data for ConfigBuilder
// This interface allows for dependency injection and easier testing
type AgentConfigProvider interface {
	// GetAgentType returns an agent type by ID
	GetAgentType(ctx context.Context, id int64) (*agent.AgentType, error)
	// GetUserEffectiveConfig returns the effective config by merging defaults and user config
	GetUserEffectiveConfig(ctx context.Context, userID, agentTypeID int64, overrides agent.ConfigValues) agent.ConfigValues
	// GetEffectiveCredentialsForPod returns credentials for pod injection
	GetEffectiveCredentialsForPod(ctx context.Context, userID, agentTypeID int64, profileID *int64) (agent.EncryptedCredentials, bool, error)
}

// Note: AgentConfigProvider is implemented by compositeProvider in API handlers
// that combine the three sub-services (AgentTypeService, CredentialProfileService, UserConfigService)

// ConfigBuildRequest contains all the information needed to build a pod config
type ConfigBuildRequest struct {
	AgentTypeID         int64
	OrganizationID      int64
	UserID              int64
	CredentialProfileID *int64

	// RepositoryID is the repository this pod belongs to (for loading installed extensions)
	RepositoryID *int64

	// Repository configuration
	RepositoryURL string // Repository clone URL (legacy, for backward compatibility)
	HttpCloneURL  string // HTTPS clone URL
	SshCloneURL   string // SSH clone URL
	SourceBranch  string // Branch to checkout

	// Git authentication
	// CredentialType determines how to authenticate:
	// - "runner_local": Use Runner's local git config, no credentials needed
	// - "oauth" or "pat": Use GitToken
	// - "ssh_key": Use SSHPrivateKey
	CredentialType string
	GitToken       string // For oauth/pat types
	SSHPrivateKey  string // For ssh_key type (private key content)

	// Ticket association
	TicketSlug string

	// Preparation script (from Repository)
	PreparationScript  string
	PreparationTimeout int

	// Local path mode (reserved for future)
	LocalPath string

	// User-provided config overrides
	ConfigOverrides map[string]interface{}

	// Initial prompt (prepended to LaunchArgs)
	InitialPrompt string

	// Runtime info (provided by Runner during handshake)
	MCPPort int
	PodKey  string

	// Terminal size (from browser)
	Cols int32
	Rows int32

	// RunnerAgentVersions maps agent slug to version string.
	// Populated from Runner.AgentVersions during pod creation.
	// Empty map or nil means Runner did not report version info (old Runner).
	RunnerAgentVersions map[string]string
}

// ConfigSchemaResponse is the config schema returned to frontend
// Frontend is responsible for i18n translation using slug + field.name as key
type ConfigSchemaResponse struct {
	Fields []ConfigFieldResponse `json:"fields"`
}

// ConfigFieldResponse is a config field returned to frontend
type ConfigFieldResponse struct {
	Name       string                `json:"name"`
	Type       string                `json:"type"`
	Default    interface{}           `json:"default,omitempty"`
	Required   bool                  `json:"required,omitempty"`
	Options    []FieldOptionResponse `json:"options,omitempty"`
	Validation *agent.Validation     `json:"validation,omitempty"`
	ShowWhen   *agent.Condition      `json:"show_when,omitempty"`
}

// FieldOptionResponse is a field option returned to frontend
type FieldOptionResponse struct {
	Value string `json:"value"`
}
