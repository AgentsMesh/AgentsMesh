package v1

import (
	"context"

	agentDomain "github.com/AgentsMesh/AgentsMesh/backend/internal/domain/agent"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/agent"
)

// AgentHandler handles agent-related requests
type AgentHandler struct {
	agentTypeSvc  *agent.AgentTypeService
	credentialSvc *agent.CredentialProfileService
	userConfigSvc *agent.UserConfigService
	configBuilder *agent.ConfigBuilder
}

// NewAgentHandler creates a new agent handler
func NewAgentHandler(
	agentTypeSvc *agent.AgentTypeService,
	credentialSvc *agent.CredentialProfileService,
	userConfigSvc *agent.UserConfigService,
) *AgentHandler {
	return &AgentHandler{
		agentTypeSvc:  agentTypeSvc,
		credentialSvc: credentialSvc,
		userConfigSvc: userConfigSvc,
		configBuilder: agent.NewConfigBuilder(&compositeProvider{
			agentTypeSvc:  agentTypeSvc,
			credentialSvc: credentialSvc,
			userConfigSvc: userConfigSvc,
		}),
	}
}

// compositeProvider implements AgentConfigProvider by combining sub-services
type compositeProvider struct {
	agentTypeSvc  *agent.AgentTypeService
	credentialSvc *agent.CredentialProfileService
	userConfigSvc *agent.UserConfigService
}

func (p *compositeProvider) GetAgentType(ctx context.Context, id int64) (*agentDomain.AgentType, error) {
	return p.agentTypeSvc.GetAgentType(ctx, id)
}

func (p *compositeProvider) GetUserEffectiveConfig(ctx context.Context, userID, agentTypeID int64, overrides agentDomain.ConfigValues) agentDomain.ConfigValues {
	return p.userConfigSvc.GetUserEffectiveConfig(ctx, userID, agentTypeID, overrides)
}

func (p *compositeProvider) GetEffectiveCredentialsForPod(ctx context.Context, userID, agentTypeID int64, profileID *int64) (agentDomain.EncryptedCredentials, bool, error) {
	return p.credentialSvc.GetEffectiveCredentialsForPod(ctx, userID, agentTypeID, profileID)
}

// CreateCustomAgentRequest represents custom agent creation request
type CreateCustomAgentRequest struct {
	Slug             string                 `json:"slug" binding:"required,min=2,max=50,alphanum"`
	Name             string                 `json:"name" binding:"required,min=2,max=100"`
	Description      string                 `json:"description"`
	LaunchCommand    string                 `json:"launch_command" binding:"required"`
	DefaultArgs      string                 `json:"default_args"`
	CredentialSchema map[string]interface{} `json:"credential_schema"`
	StatusDetection  map[string]interface{} `json:"status_detection"`
}

// SetUserAgentConfigRequest represents a request to set user's personal config
type SetUserAgentConfigRequest struct {
	ConfigValues map[string]interface{} `json:"config_values" binding:"required"`
}
