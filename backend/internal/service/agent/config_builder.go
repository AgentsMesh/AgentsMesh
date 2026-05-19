package agent

import (
	"context"
	"fmt"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
	extensionservice "github.com/anthropics/agentsmesh/backend/internal/service/extension"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

type ExtensionProvider interface {
	GetEffectiveMcpServers(ctx context.Context, orgID, userID, repoID int64, agentSlug string) ([]*extension.InstalledMcpServer, error)
	GetEffectiveSkills(ctx context.Context, orgID, userID, repoID int64, agentSlug string) ([]*extensionservice.ResolvedSkill, error)
}

type ConfigBuilder struct {
	provider          AgentConfigProvider
	extensionProvider ExtensionProvider
}

func NewConfigBuilder(provider AgentConfigProvider) *ConfigBuilder {
	return &ConfigBuilder{provider: provider}
}

func (b *ConfigBuilder) SetExtensionProvider(ep ExtensionProvider) {
	b.extensionProvider = ep
}

func (b *ConfigBuilder) BuildPodCommand(ctx context.Context, req *ConfigBuildRequest) (*runnerv1.CreatePodCommand, error) {
	agentDef, err := b.provider.GetAgent(ctx, req.AgentSlug)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	if agentDef.AgentfileSource == nil || *agentDef.AgentfileSource == "" {
		return nil, fmt.Errorf("agent %q has no AgentFile defined", agentDef.Slug)
	}

	return b.buildFromAgentfile(ctx, req, agentDef)
}
