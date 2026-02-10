package agent

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
)

// CompositeAgentProvider implements AgentConfigProvider by combining three sub-services.
// This allows callers to work with the split service architecture through a single interface.
type CompositeAgentProvider struct {
	agentTypeSvc  *AgentTypeService
	credentialSvc *CredentialProfileService
	userConfigSvc *UserConfigService
}

// NewCompositeProvider creates an AgentConfigProvider that delegates to the three sub-services.
func NewCompositeProvider(
	agentTypeSvc *AgentTypeService,
	credSvc *CredentialProfileService,
	configSvc *UserConfigService,
) AgentConfigProvider {
	return &CompositeAgentProvider{
		agentTypeSvc:  agentTypeSvc,
		credentialSvc: credSvc,
		userConfigSvc: configSvc,
	}
}

func (p *CompositeAgentProvider) GetAgentType(ctx context.Context, id int64) (*agent.AgentType, error) {
	return p.agentTypeSvc.GetAgentType(ctx, id)
}

func (p *CompositeAgentProvider) GetUserEffectiveConfig(ctx context.Context, userID, agentTypeID int64, overrides agent.ConfigValues) agent.ConfigValues {
	return p.userConfigSvc.GetUserEffectiveConfig(ctx, userID, agentTypeID, overrides)
}

func (p *CompositeAgentProvider) GetEffectiveCredentialsForPod(ctx context.Context, userID, agentTypeID int64, profileID *int64) (agent.EncryptedCredentials, bool, error) {
	return p.credentialSvc.GetEffectiveCredentialsForPod(ctx, userID, agentTypeID, profileID)
}
