package v1

import (
	"context"

	agentDomain "github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	"github.com/anthropics/agentsmesh/backend/internal/service/agent"
)

// AgentHandler handles agent-related requests
type AgentHandler struct {
	agentSvc  *agent.AgentService
	credentialSvc *agent.CredentialProfileService
	userConfigSvc *agent.UserConfigService
	configBuilder *agent.ConfigBuilder
}

// NewAgentHandler creates a new agent handler
func NewAgentHandler(
	agentSvc *agent.AgentService,
	credentialSvc *agent.CredentialProfileService,
	userConfigSvc *agent.UserConfigService,
) *AgentHandler {
	return &AgentHandler{
		agentSvc:  agentSvc,
		credentialSvc: credentialSvc,
		userConfigSvc: userConfigSvc,
		configBuilder: agent.NewConfigBuilder(&compositeProvider{
			agentSvc:  agentSvc,
			credentialSvc: credentialSvc,
		}),
	}
}

// compositeProvider implements AgentConfigProvider by combining sub-services
type compositeProvider struct {
	agentSvc  *agent.AgentService
	credentialSvc *agent.CredentialProfileService
}

func (p *compositeProvider) GetAgent(ctx context.Context, slug string) (*agentDomain.Agent, error) {
	return p.agentSvc.GetAgent(ctx, slug)
}

func (p *compositeProvider) GetEffectiveCredentialsForPod(ctx context.Context, userID int64, agentSlug string, profileID *int64) (agentDomain.EncryptedCredentials, bool, error) {
	return p.credentialSvc.GetEffectiveCredentialsForPod(ctx, userID, agentSlug, profileID)
}

func (p *compositeProvider) ResolveCredentialsByName(ctx context.Context, userID int64, agentSlug, profileName string) (agentDomain.EncryptedCredentials, bool, error) {
	return p.credentialSvc.ResolveCredentialsByName(ctx, userID, agentSlug, profileName)
}

// CreateCustomAgentRequest represents custom agent creation request.
// When AgentfileSource is provided, LaunchCommand becomes optional (extracted from AgentFile).
type CreateCustomAgentRequest struct {
	Slug          string `json:"slug" binding:"required,min=2,max=50,alphanum"`
	Name          string `json:"name" binding:"required,min=2,max=100"`
	Description   string `json:"description"`
	AgentfileSource string `json:"agentfile_source"`
	LaunchCommand string `json:"launch_command"`
	DefaultArgs   string `json:"default_args"`
}

// SetUserAgentConfigRequest represents a request to set user's personal config
type SetUserAgentConfigRequest struct {
	ConfigValues map[string]interface{} `json:"config_values" binding:"required"`
}
