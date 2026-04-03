package agent

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
)

// CompositeAgentProvider implements AgentConfigProvider by combining sub-services.
type CompositeAgentProvider struct {
	agentSvc      *AgentService
	credentialSvc *CredentialProfileService
}

// NewCompositeProvider creates an AgentConfigProvider that delegates to sub-services.
func NewCompositeProvider(
	agentSvc *AgentService,
	credSvc *CredentialProfileService,
	configSvc *UserConfigService,
) AgentConfigProvider {
	return &CompositeAgentProvider{
		agentSvc:      agentSvc,
		credentialSvc: credSvc,
	}
}

func (p *CompositeAgentProvider) GetAgent(ctx context.Context, slug string) (*agent.Agent, error) {
	return p.agentSvc.GetAgent(ctx, slug)
}

func (p *CompositeAgentProvider) GetEffectiveCredentialsForPod(ctx context.Context, userID int64, agentSlug string, profileID *int64) (agent.EncryptedCredentials, bool, error) {
	return p.credentialSvc.GetEffectiveCredentialsForPod(ctx, userID, agentSlug, profileID)
}

func (p *CompositeAgentProvider) ResolveCredentialsByName(ctx context.Context, userID int64, agentSlug, profileName string) (agent.EncryptedCredentials, bool, error) {
	return p.credentialSvc.ResolveCredentialsByName(ctx, userID, agentSlug, profileName)
}
