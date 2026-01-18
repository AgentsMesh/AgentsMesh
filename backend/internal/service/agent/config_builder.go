package agent

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"text/template"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
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

// PodConfig represents the computed configuration to send to Runner
type PodConfig struct {
	LaunchCommand string            `json:"launch_command"`
	LaunchArgs    []string          `json:"launch_args"`
	EnvVars       map[string]string `json:"env_vars"`
	FilesToCreate []FileToCreate    `json:"files_to_create"`
	WorkDirConfig WorkDirConfig     `json:"work_dir_config"`
}

// FileToCreate represents a file to be created in the sandbox
type FileToCreate struct {
	PathTemplate string `json:"path_template"`
	Content      string `json:"content"`
	Mode         int    `json:"mode,omitempty"`
	IsDirectory  bool   `json:"is_directory,omitempty"`
}

// WorkDirConfig represents the working directory configuration
type WorkDirConfig struct {
	Type          string `json:"type"` // "worktree", "tempdir", "local"
	RepositoryURL string `json:"repository_url,omitempty"`
	Branch        string `json:"branch,omitempty"`
	TicketID      string `json:"ticket_id,omitempty"`
	GitToken      string `json:"git_token,omitempty"`
	SSHKeyPath    string `json:"ssh_key_path,omitempty"`
	LocalPath     string `json:"local_path,omitempty"`
}

// ConfigBuildRequest contains all the information needed to build a pod config
type ConfigBuildRequest struct {
	AgentTypeID         int64
	OrganizationID      int64
	UserID              int64
	CredentialProfileID *int64

	// Repository info
	RepositoryURL string
	Branch        string
	TicketID      string
	GitToken      string
	SSHKeyPath    string
	LocalPath     string

	// User-provided config overrides
	ConfigOverrides map[string]interface{}

	// Initial prompt (appended to LaunchArgs)
	InitialPrompt string

	// Runtime info (provided by Runner during handshake)
	MCPPort int
	PodKey  string
}

// ConfigBuilder builds pod configurations from agent type templates
type ConfigBuilder struct {
	provider AgentConfigProvider
}

// NewConfigBuilder creates a new ConfigBuilder
func NewConfigBuilder(provider AgentConfigProvider) *ConfigBuilder {
	return &ConfigBuilder{
		provider: provider,
	}
}

// BuildPodConfig builds the complete pod configuration
func (b *ConfigBuilder) BuildPodConfig(ctx context.Context, req *ConfigBuildRequest) (*PodConfig, error) {
	// 1. Get agent type
	agentType, err := b.provider.GetAgentType(ctx, req.AgentTypeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent type: %w", err)
	}

	// 2. Merge configs: ConfigSchema defaults + user personal config + overrides
	config := b.provider.GetUserEffectiveConfig(ctx, req.UserID, req.AgentTypeID, agent.ConfigValues(req.ConfigOverrides))

	// 3. Get credentials and inject as env vars
	envVars, err := b.buildEnvVars(ctx, req, agentType)
	if err != nil {
		return nil, fmt.Errorf("failed to build env vars: %w", err)
	}

	// 4. Build template context
	templateCtx := b.buildTemplateContext(req, config)

	// 5. Build launch args from CommandTemplate
	launchArgs, err := b.buildLaunchArgs(agentType.CommandTemplate, config, templateCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to build launch args: %w", err)
	}

	// Prepend InitialPrompt to launch args if provided.
	// Claude Code CLI syntax: claude [prompt] [options]
	// The prompt must come BEFORE options, not after.
	if req.InitialPrompt != "" {
		launchArgs = append([]string{req.InitialPrompt}, launchArgs...)
	}

	// 6. Build files to create from FilesTemplate
	filesToCreate, err := b.buildFilesToCreate(agentType.FilesTemplate, config, templateCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to build files to create: %w", err)
	}

	// 7. Build work dir config
	workDirConfig := b.buildWorkDirConfig(req)

	return &PodConfig{
		LaunchCommand: agentType.LaunchCommand,
		LaunchArgs:    launchArgs,
		EnvVars:       envVars,
		FilesToCreate: filesToCreate,
		WorkDirConfig: workDirConfig,
	}, nil
}

// buildEnvVars builds environment variables including credentials
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

// buildFilesToCreate builds the list of files to create from FilesTemplate
func (b *ConfigBuilder) buildFilesToCreate(filesTemplate agent.FilesTemplate, config agent.ConfigValues, templateCtx map[string]interface{}) ([]FileToCreate, error) {
	var files []FileToCreate

	for _, ft := range filesTemplate {
		// Check condition
		if ft.Condition != nil && !ft.Condition.Evaluate(config) {
			continue
		}

		// For directories, just add the path
		if ft.IsDirectory {
			files = append(files, FileToCreate{
				PathTemplate: ft.PathTemplate,
				IsDirectory:  true,
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

		files = append(files, FileToCreate{
			PathTemplate: ft.PathTemplate,
			Content:      content,
			Mode:         mode,
		})
	}

	return files, nil
}

// buildWorkDirConfig builds the working directory configuration
func (b *ConfigBuilder) buildWorkDirConfig(req *ConfigBuildRequest) WorkDirConfig {
	// Determine work dir type
	workDirType := "tempdir"
	if req.RepositoryURL != "" {
		workDirType = "worktree"
	} else if req.LocalPath != "" {
		workDirType = "local"
	}

	return WorkDirConfig{
		Type:          workDirType,
		RepositoryURL: req.RepositoryURL,
		Branch:        req.Branch,
		TicketID:      req.TicketID,
		GitToken:      req.GitToken,
		SSHKeyPath:    req.SSHKeyPath,
		LocalPath:     req.LocalPath,
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
