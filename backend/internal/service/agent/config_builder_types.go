package agent

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
)

type AgentConfigProvider interface {
	GetAgent(ctx context.Context, slug string) (*agent.Agent, error)
	GetEffectiveCredentialsForPod(ctx context.Context, userID int64, agentSlug string, profileID *int64) (agent.EncryptedCredentials, bool, error)
	ResolveCredentialsByName(ctx context.Context, userID int64, agentSlug, profileName string) (agent.EncryptedCredentials, bool, error)
}

type ConfigBuildRequest struct {
	AgentSlug           string
	OrganizationID      int64
	UserID              int64
	CredentialProfileID *int64

	RepositoryID *int64

	HttpCloneURL string // HTTPS clone URL
	SshCloneURL  string // SSH clone URL
	SourceBranch string // Branch to checkout

	CredentialType string
	GitToken       string // For oauth/pat types
	SSHPrivateKey  string // For ssh_key type (private key content)

	TicketSlug string

	PreparationScript  string
	PreparationTimeout int

	LocalPath string

	Prompt string

	MCPPort int
	PodKey  string

	Cols int32
	Rows int32

	RunnerAgentVersions map[string]string

	MergedAgentfileSource string

	CredentialProfile string
}

type ConfigSchemaResponse struct {
	Fields           []ConfigFieldResponse     `json:"fields"`
	CredentialFields []CredentialFieldResponse `json:"credential_fields,omitempty"`
}

type CredentialFieldResponse struct {
	Name     string `json:"name"` // Full ENV name, e.g. "ANTHROPIC_API_KEY"
	Type     string `json:"type"` // "secret" or "text"
	Optional bool   `json:"optional"`
}

type ConfigFieldResponse struct {
	Name    string                `json:"name"`
	Type    string                `json:"type"`
	Default interface{}           `json:"default,omitempty"`
	Options []FieldOptionResponse `json:"options,omitempty"`
}

type FieldOptionResponse struct {
	Value string `json:"value"`
}
