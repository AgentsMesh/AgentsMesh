package agent

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strings"
	"text/template"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
	extensionservice "github.com/anthropics/agentsmesh/backend/internal/service/extension"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// ExtensionProvider provides installed extension capabilities for a repository
type ExtensionProvider interface {
	GetEffectiveMcpServers(ctx context.Context, orgID, userID, repoID int64, agentSlug string) ([]*extension.InstalledMcpServer, error)
	GetEffectiveSkills(ctx context.Context, orgID, userID, repoID int64, agentSlug string) ([]*extensionservice.ResolvedSkill, error)
}

// ConfigBuilder builds pod configurations from agent type templates
// It uses the Strategy pattern to delegate agent-specific logic to AgentBuilder implementations
type ConfigBuilder struct {
	provider          AgentConfigProvider
	registry          *AgentBuilderRegistry
	extensionProvider ExtensionProvider
}

// NewConfigBuilder creates a new ConfigBuilder with default builder registry
func NewConfigBuilder(provider AgentConfigProvider) *ConfigBuilder {
	return &ConfigBuilder{
		provider: provider,
		registry: NewAgentBuilderRegistry(),
	}
}

// NewConfigBuilderWithRegistry creates a ConfigBuilder with a custom registry
// This is useful for testing or when custom builders need to be registered
func NewConfigBuilderWithRegistry(provider AgentConfigProvider, registry *AgentBuilderRegistry) *ConfigBuilder {
	return &ConfigBuilder{
		provider: provider,
		registry: registry,
	}
}

// SetExtensionProvider sets the extension provider for loading MCP servers and skills
func (b *ConfigBuilder) SetExtensionProvider(ep ExtensionProvider) {
	b.extensionProvider = ep
}

// BuildPodCommand builds the complete pod command using the Strategy pattern.
// It delegates agent-specific logic to the appropriate AgentBuilder.
func (b *ConfigBuilder) BuildPodCommand(ctx context.Context, req *ConfigBuildRequest) (*runnerv1.CreatePodCommand, error) {
	// 1. Get agent type
	agentType, err := b.provider.GetAgentType(ctx, req.AgentTypeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent type: %w", err)
	}

	// 2. Get the appropriate builder strategy for this agent type
	builder := b.registry.Get(agentType.Slug)

	// 3. Merge configs: ConfigSchema defaults + user personal config + overrides
	config := b.provider.GetUserEffectiveConfig(ctx, req.UserID, req.AgentTypeID, agent.ConfigValues(req.ConfigOverrides))

	// 4. Get credentials
	creds, isRunnerHost, err := b.provider.GetEffectiveCredentialsForPod(ctx, req.UserID, req.AgentTypeID, req.CredentialProfileID)
	if err != nil {
		return nil, fmt.Errorf("failed to build env vars: %w", err)
	}

	// 5. Build template context
	templateCtx := b.buildTemplateContext(req, config)

	// 5.5. Extract agent version from Runner-reported versions
	agentVersion := ""
	if req.RunnerAgentVersions != nil {
		agentVersion = req.RunnerAgentVersions[agentType.Slug]
	}

	// 6. Create build context for the strategy
	buildCtx := NewBuildContext(req, agentType, config, creds, isRunnerHost, templateCtx, agentVersion)

	// 7. Use builder strategy to build launch args
	launchArgs, err := builder.BuildLaunchArgs(buildCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to build launch args: %w", err)
	}

	// 8. Handle InitialPrompt using agent-specific strategy
	// Different agents handle prompts differently:
	// - Claude Code: prepend to args (claude [prompt] [options])
	// - Gemini CLI: append to args (gemini [options] [prompt])
	// - Aider: does not support command-line prompt
	launchArgs = builder.HandleInitialPrompt(buildCtx, launchArgs)

	// 9. Load extension capabilities (MCP servers, skills)
	b.loadExtensions(ctx, req, builder, buildCtx)

	// 10. Build files to create using strategy
	filesToCreate, err := builder.BuildFilesToCreate(buildCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to build files to create: %w", err)
	}

	// 10. Build env vars using strategy
	envVars, err := builder.BuildEnvVars(buildCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to build env vars: %w", err)
	}

	// 11. Build sandbox config (common for all agents)
	sandboxConfig := b.buildSandboxConfig(req)

	// 12. Create the command
	cmd := &runnerv1.CreatePodCommand{
		PodKey:        req.PodKey,
		LaunchCommand: agentType.LaunchCommand,
		LaunchArgs:    launchArgs,
		EnvVars:       envVars,
		FilesToCreate: filesToCreate,
		SandboxConfig: sandboxConfig,
		InitialPrompt: req.InitialPrompt,
		Cols:          req.Cols,
		Rows:          req.Rows,
	}

	// 14. Allow post-processing by the builder
	if err := builder.PostProcess(buildCtx, cmd); err != nil {
		return nil, fmt.Errorf("failed to post-process command: %w", err)
	}

	// 15. Build resources to download
	resources, err := builder.BuildResourcesToDownload(buildCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to build resources to download: %w", err)
	}
	cmd.ResourcesToDownload = resources

	return cmd, nil
}

// buildTemplateContext builds the context for template rendering
func (b *ConfigBuilder) buildTemplateContext(req *ConfigBuildRequest, config agent.ConfigValues) map[string]interface{} {
	return map[string]interface{}{
		"config": config,
		"sandbox": map[string]interface{}{
			"root_path": "{{.sandbox.root_path}}", // Placeholder, resolved by Runner
			"work_dir":  "{{.sandbox.work_dir}}",  // Placeholder, resolved by Runner
		},
		"mcp_port": req.MCPPort,
		"pod_key":  req.PodKey,
	}
}

// buildSandboxConfig builds the sandbox configuration directly as Proto type
func (b *ConfigBuilder) buildSandboxConfig(req *ConfigBuildRequest) *runnerv1.SandboxConfig {
	// Only create SandboxConfig if there's repository or local path config
	if req.RepositoryURL == "" && req.HttpCloneURL == "" && req.SshCloneURL == "" && req.LocalPath == "" {
		return nil
	}

	timeout := int32(req.PreparationTimeout)
	if timeout <= 0 {
		timeout = 300 // Default 5 minutes
	}

	return &runnerv1.SandboxConfig{
		RepositoryUrl:      req.RepositoryURL,
		HttpCloneUrl:       req.HttpCloneURL,
		SshCloneUrl:        req.SshCloneURL,
		SourceBranch:       req.SourceBranch,
		CredentialType:     req.CredentialType,
		GitToken:           req.GitToken,
		SshPrivateKey:      req.SSHPrivateKey,
		TicketSlug:         req.TicketSlug,
		PreparationScript:  req.PreparationScript,
		PreparationTimeout: timeout,
		LocalPath:          req.LocalPath,
	}
}

func (b *ConfigBuilder) loadExtensions(ctx context.Context, req *ConfigBuildRequest, builder AgentBuilder, buildCtx *BuildContext) {
	if b.extensionProvider == nil || req.RepositoryID == nil {
		return
	}

	orgID := req.OrganizationID
	userID := req.UserID
	repoID := *req.RepositoryID

	if builder.SupportsMcp() {
		servers, err := b.extensionProvider.GetEffectiveMcpServers(ctx, orgID, userID, repoID, builder.Slug())
		if err != nil {
			slog.Warn("Failed to load effective MCP servers", "error", err, "org_id", orgID, "repo_id", repoID)
		} else {
			buildCtx.McpServers = servers
		}
	}

	if builder.SupportsSkills() {
		skills, err := b.extensionProvider.GetEffectiveSkills(ctx, orgID, userID, repoID, builder.Slug())
		if err != nil {
			slog.Warn("Failed to load effective skills", "error", err, "org_id", orgID, "repo_id", repoID)
		} else {
			buildCtx.ResolvedSkills = skills
		}
	}
}

// renderTemplate renders a Go template string with the given context
func (b *ConfigBuilder) renderTemplate(templateStr string, ctx map[string]interface{}) (string, error) {
	// Skip if no template markers
	if !strings.Contains(templateStr, "{{") {
		return templateStr, nil
	}

	tmpl, err := template.New("").Parse(templateStr)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, ctx); err != nil {
		return "", err
	}

	return buf.String(), nil
}
