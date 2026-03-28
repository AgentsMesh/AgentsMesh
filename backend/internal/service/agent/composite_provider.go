package agent

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
)

// CompositeAgentProvider implements AgentConfigProvider by combining three sub-services.
// This allows callers to work with the split service architecture through a single interface.
type CompositeAgentProvider struct {
	agentSvc      *AgentService
	credentialSvc *CredentialProfileService
	userConfigSvc *UserConfigService
}

// NewCompositeProvider creates an AgentConfigProvider that delegates to the three sub-services.
func NewCompositeProvider(
	agentSvc *AgentService,
	credSvc *CredentialProfileService,
	configSvc *UserConfigService,
) AgentConfigProvider {
	return &CompositeAgentProvider{
		agentSvc:      agentSvc,
		credentialSvc: credSvc,
		userConfigSvc: configSvc,
	}
}

func (p *CompositeAgentProvider) GetAgent(ctx context.Context, slug string) (*agent.Agent, error) {
	return p.agentSvc.GetAgent(ctx, slug)
}

func (p *CompositeAgentProvider) GetUserEffectiveConfig(ctx context.Context, userID int64, agentSlug string, overrides agent.ConfigValues) agent.ConfigValues {
	return p.userConfigSvc.GetUserEffectiveConfig(ctx, userID, agentSlug, overrides)
}

func (p *CompositeAgentProvider) GetEffectiveCredentialsForPod(ctx context.Context, userID int64, agentSlug string, profileID *int64) (agent.EncryptedCredentials, bool, error) {
	return p.credentialSvc.GetEffectiveCredentialsForPod(ctx, userID, agentSlug, profileID)
}

func (p *CompositeAgentProvider) ResolveCredentialsByName(ctx context.Context, userID int64, agentSlug, profileName string) (agent.EncryptedCredentials, bool, error) {
	return p.credentialSvc.ResolveCredentialsByName(ctx, userID, agentSlug, profileName)
}
