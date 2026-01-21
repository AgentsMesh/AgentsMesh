package agent

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"text/template"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// AgentConfigProvider provides agent configuration data for ConfigBuilder
// This interface allows for dependency injection and easier testing
type AgentConfigProvider interface {
	// GetAgentType returns an agent type by ID
	GetAgentType(ctx context.Context, id int64) (*agent.AgentType, error)
	// GetUserEffectiveConfig returns the effective config by merging defaults and user config
	GetUserEffectiveConfig(ctx context.Context, userID, agentTypeID int64, overrides agent.ConfigValues) agent.ConfigValues
	// GetEffectiveCredentialsForPod returns credentials for pod injection
	GetEffectiveCredentialsForPod(ctx context.Context, userID, agentTypeID int64, profileID *int64) (agent.EncryptedCredentials, bool, error)
}

// Note: AgentConfigProvider is implemented by compositeProvider in API handlers
// that combine the three sub-services (AgentTypeService, CredentialProfileService, UserConfigService)

// ConfigBuildRequest contains all the information needed to build a pod config
type ConfigBuildRequest struct {
	AgentTypeID         int64
	OrganizationID      int64
	UserID              int64
	CredentialProfileID *int64

	// Repository configuration
	RepositoryURL string // Repository clone URL
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
	TicketID string

	// Preparation script (from Repository)
	PreparationScript  string
	PreparationTimeout int

	// Local path mode (reserved for future)
	LocalPath string

	// User-provided config overrides
	ConfigOverrides map[string]interface{}

	// Initial prompt (prepended to LaunchArgs)
	InitialPrompt string

	// Runtime info (provided by Runner during handshake)
	MCPPort int
	PodKey  string
}

// ConfigBuilder builds pod configurations from agent type templates
// It uses the Strategy pattern to delegate agent-specific logic to AgentBuilder implementations
type ConfigBuilder struct {
	provider AgentConfigProvider
	registry *AgentBuilderRegistry
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

	// 6. Create build context for the strategy
	buildCtx := NewBuildContext(req, agentType, config, creds, isRunnerHost, templateCtx)

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

	// 9. Build files to create using strategy
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
	}

	// 13. Allow post-processing by the builder
	if err := builder.PostProcess(buildCtx, cmd); err != nil {
		return nil, fmt.Errorf("failed to post-process command: %w", err)
	}

	return cmd, nil
}

// buildEnvVars builds environment variables including credentials
// Deprecated: Use AgentBuilder.BuildEnvVars instead. Kept for backward compatibility.
func (b *ConfigBuilder) buildEnvVars(ctx context.Context, req *ConfigBuildRequest, agentType *agent.AgentType) (map[string]string, error) {
	envVars := make(map[string]string)

	// Get credentials from profile
	creds, isRunnerHost, err := b.provider.GetEffectiveCredentialsForPod(ctx, req.UserID, req.AgentTypeID, req.CredentialProfileID)
	if err != nil {
		return nil, err
	}

	// If using RunnerHost mode, don't inject credentials
	if isRunnerHost {
		return envVars, nil
	}

	// Map credentials to env vars based on credential schema
	for _, field := range agentType.CredentialSchema {
		if value, ok := creds[field.Name]; ok && value != "" {
			envVars[field.EnvVar] = value
		}
	}

	return envVars, nil
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

// buildLaunchArgs builds launch arguments from CommandTemplate
// Deprecated: Use AgentBuilder.BuildLaunchArgs instead. Kept for backward compatibility.
func (b *ConfigBuilder) buildLaunchArgs(cmdTemplate agent.CommandTemplate, config agent.ConfigValues, templateCtx map[string]interface{}) ([]string, error) {
	var args []string

	for _, rule := range cmdTemplate.Args {
		// Check condition
		if rule.Condition != nil && !rule.Condition.Evaluate(config) {
			continue
		}

		// Render each arg template
		for _, argTemplate := range rule.Args {
			rendered, err := b.renderTemplate(argTemplate, templateCtx)
			if err != nil {
				return nil, fmt.Errorf("failed to render arg template %q: %w", argTemplate, err)
			}
			if rendered != "" {
				args = append(args, rendered)
			}
		}
	}

	return args, nil
}

// buildFilesToCreateProto builds the list of files to create directly as Proto type
// Deprecated: Use AgentBuilder.BuildFilesToCreate instead. Kept for backward compatibility.
func (b *ConfigBuilder) buildFilesToCreateProto(filesTemplate agent.FilesTemplate, config agent.ConfigValues, templateCtx map[string]interface{}) ([]*runnerv1.FileToCreate, error) {
	var files []*runnerv1.FileToCreate

	for _, ft := range filesTemplate {
		// Check condition
		if ft.Condition != nil && !ft.Condition.Evaluate(config) {
			continue
		}

		// For directories, just add the path
		if ft.IsDirectory {
			files = append(files, &runnerv1.FileToCreate{
				Path:        ft.PathTemplate,
				IsDirectory: true,
			})
			continue
		}

		// Render content template
		content, err := b.renderTemplate(ft.ContentTemplate, templateCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to render content template for %q: %w", ft.PathTemplate, err)
		}

		mode := ft.Mode
		if mode == 0 {
			mode = 0644 // Default permission
		}

		files = append(files, &runnerv1.FileToCreate{
			Path:    ft.PathTemplate,
			Content: content,
			Mode:    int32(mode),
		})
	}

	return files, nil
}

// buildSandboxConfig builds the sandbox configuration directly as Proto type
func (b *ConfigBuilder) buildSandboxConfig(req *ConfigBuildRequest) *runnerv1.SandboxConfig {
	// Only create SandboxConfig if there's repository or local path config
	if req.RepositoryURL == "" && req.LocalPath == "" {
		return nil
	}

	timeout := int32(req.PreparationTimeout)
	if timeout <= 0 {
		timeout = 300 // Default 5 minutes
	}

	return &runnerv1.SandboxConfig{
		RepositoryUrl:      req.RepositoryURL,
		SourceBranch:       req.SourceBranch,
		CredentialType:     req.CredentialType,
		GitToken:           req.GitToken,
		SshPrivateKey:      req.SSHPrivateKey,
		TicketId:           req.TicketID,
		PreparationScript:  req.PreparationScript,
		PreparationTimeout: timeout,
		LocalPath:          req.LocalPath,
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

// ConfigSchemaResponse is the config schema returned to frontend
// Frontend is responsible for i18n translation using slug + field.name as key
type ConfigSchemaResponse struct {
	Fields []ConfigFieldResponse `json:"fields"`
}

// ConfigFieldResponse is a config field returned to frontend
type ConfigFieldResponse struct {
	Name       string                `json:"name"`
	Type       string                `json:"type"`
	Default    interface{}           `json:"default,omitempty"`
	Required   bool                  `json:"required,omitempty"`
	Options    []FieldOptionResponse `json:"options,omitempty"`
	Validation *agent.Validation     `json:"validation,omitempty"`
	ShowWhen   *agent.Condition      `json:"show_when,omitempty"`
}

// FieldOptionResponse is a field option returned to frontend
type FieldOptionResponse struct {
	Value string `json:"value"`
}

// GetConfigSchema returns the raw config schema for an agent type
// Frontend is responsible for i18n translation using: agent.{slug}.fields.{field.name}.label
func (b *ConfigBuilder) GetConfigSchema(ctx context.Context, agentTypeID int64) (*ConfigSchemaResponse, error) {
	agentType, err := b.provider.GetAgentType(ctx, agentTypeID)
	if err != nil {
		return nil, err
	}

	return b.buildConfigSchemaResponse(&agentType.ConfigSchema), nil
}

// buildConfigSchemaResponse converts internal ConfigSchema to API response
func (b *ConfigBuilder) buildConfigSchemaResponse(schema *agent.ConfigSchema) *ConfigSchemaResponse {
	result := &ConfigSchemaResponse{
		Fields: make([]ConfigFieldResponse, 0, len(schema.Fields)),
	}

	for _, field := range schema.Fields {
		fieldResponse := ConfigFieldResponse{
			Name:       field.Name,
			Type:       field.Type,
			Default:    field.Default,
			Required:   field.Required,
			Validation: field.Validation,
			ShowWhen:   field.ShowWhen,
		}

		// Convert options (without label - frontend will translate)
		if len(field.Options) > 0 {
			fieldResponse.Options = make([]FieldOptionResponse, 0, len(field.Options))
			for _, opt := range field.Options {
				fieldResponse.Options = append(fieldResponse.Options, FieldOptionResponse{
					Value: opt.Value,
				})
			}
		}

		result.Fields = append(result.Fields, fieldResponse)
	}

	return result
}
