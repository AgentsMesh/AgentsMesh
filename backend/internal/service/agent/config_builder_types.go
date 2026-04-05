package agent

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
)

// AgentConfigProvider provides agent configuration data for ConfigBuilder
type AgentConfigProvider interface {
	// GetAgent returns an agent by slug
	GetAgent(ctx context.Context, slug string) (*agent.Agent, error)
	// GetEffectiveCredentialsForPod returns credentials for pod injection
	GetEffectiveCredentialsForPod(ctx context.Context, userID int64, agentSlug string, profileID *int64) (agent.EncryptedCredentials, bool, error)
	// ResolveCredentialsByName resolves credentials by profile name from AgentFile CREDENTIAL declaration
	ResolveCredentialsByName(ctx context.Context, userID int64, agentSlug, profileName string) (agent.EncryptedCredentials, bool, error)
}

// ConfigBuildRequest contains all the information needed to build a pod config
type ConfigBuildRequest struct {
	AgentSlug           string
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

	// Local path mode (resume from existing sandbox)
	LocalPath string

	// Initial prompt (from AgentFile PROMPT declaration)
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

	// MergedAgentfileSource is the merged AgentFile source (base + user layer, serialized).
	// Populated by orchestrator's extractFromAgentfileLayer when AgentfileLayer is provided.
	// When empty (resume mode or no layer): buildFromAgentfile falls back to agent's base AgentFile.
	MergedAgentfileSource string

	// CredentialProfile is the CREDENTIAL declaration value extracted from merged AgentFile.
	// Pre-extracted by orchestrator to avoid re-parsing AgentFile in ConfigBuilder.
	// When non-empty, overrides CredentialProfileID for credential resolution.
	CredentialProfile string
}

// ConfigSchemaResponse is the config schema returned to frontend
// Frontend is responsible for i18n translation using slug + field.name as key
type ConfigSchemaResponse struct {
	Fields           []ConfigFieldResponse       `json:"fields"`
	CredentialFields []CredentialFieldResponse    `json:"credential_fields,omitempty"`
}

// CredentialFieldResponse describes a credential field from AgentFile ENV SECRET/TEXT declarations.
// Frontend uses these to dynamically render credential profile forms.
type CredentialFieldResponse struct {
	Name     string `json:"name"`     // Full ENV name, e.g. "ANTHROPIC_API_KEY"
	Type     string `json:"type"`     // "secret" or "text"
	Optional bool   `json:"optional"`
}

// ConfigFieldResponse is a config field returned to frontend
type ConfigFieldResponse struct {
	Name    string                `json:"name"`
	Type    string                `json:"type"`
	Default interface{}           `json:"default,omitempty"`
	Options []FieldOptionResponse `json:"options,omitempty"`
}

// FieldOptionResponse is a field option returned to frontend
type FieldOptionResponse struct {
	Value string `json:"value"`
}
