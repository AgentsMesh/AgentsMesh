package agent

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
)

type CompositeAgentProvider struct {
	agentSvc      *AgentService
	credentialSvc *CredentialProfileService
}

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
