package agent

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// BaseAgentBuilder provides default implementations for AgentBuilder interface.
// It uses templates from AgentType (CommandTemplate, FilesTemplate, CredentialSchema)
// to build pod configurations. Specific agent builders can embed this struct
// and override methods that need customization.
type BaseAgentBuilder struct {
	slug string
}

// NewBaseAgentBuilder creates a new BaseAgentBuilder with the given slug
func NewBaseAgentBuilder(slug string) *BaseAgentBuilder {
	return &BaseAgentBuilder{slug: slug}
}

// Slug returns the agent type identifier
func (b *BaseAgentBuilder) Slug() string {
	return b.slug
}

// BuildLaunchArgs builds launch arguments using CommandTemplate.
// Each ArgRule is evaluated against config, and matching rules have their
// argument templates rendered and added to the result.
func (b *BaseAgentBuilder) BuildLaunchArgs(ctx *BuildContext) ([]string, error) {
	var args []string

	for _, rule := range ctx.AgentType.CommandTemplate.Args {
		// Check condition
		if rule.Condition != nil && !rule.Condition.Evaluate(ctx.Config) {
			continue
		}

		// Render each arg template
		for _, argTemplate := range rule.Args {
			rendered, err := b.renderTemplate(argTemplate, ctx.TemplateCtx)
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

// BuildFilesToCreate builds the list of files using FilesTemplate.
// Each FileTemplate is evaluated against config, and matching templates
// have their content rendered.
func (b *BaseAgentBuilder) BuildFilesToCreate(ctx *BuildContext) ([]*runnerv1.FileToCreate, error) {
	var files []*runnerv1.FileToCreate

	for _, ft := range ctx.AgentType.FilesTemplate {
		// Check condition
		if ft.Condition != nil && !ft.Condition.Evaluate(ctx.Config) {
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
		content, err := b.renderTemplate(ft.ContentTemplate, ctx.TemplateCtx)
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

// BuildEnvVars builds environment variables from credentials.
// Maps credential values to environment variable names based on CredentialSchema.
// Returns empty map if IsRunnerHost is true (use local credentials).
func (b *BaseAgentBuilder) BuildEnvVars(ctx *BuildContext) (map[string]string, error) {
	envVars := make(map[string]string)

	// If using RunnerHost mode, don't inject credentials
	if ctx.IsRunnerHost {
		return envVars, nil
	}

	// Map credentials to env vars based on credential schema
	for _, field := range ctx.AgentType.CredentialSchema {
		if value, ok := ctx.Credentials[field.Name]; ok && value != "" {
			envVars[field.EnvVar] = value
		}
	}

	return envVars, nil
}

// HandleInitialPrompt prepends the initial prompt to launch arguments.
// This is the default behavior used by Claude Code: claude [prompt] [options]
// Override this method for agents with different prompt handling.
func (b *BaseAgentBuilder) HandleInitialPrompt(ctx *BuildContext, args []string) []string {
	if ctx.Request.InitialPrompt != "" {
		return append([]string{ctx.Request.InitialPrompt}, args...)
	}
	return args
}

// PostProcess is a no-op by default.
// Override this method to perform final adjustments to CreatePodCommand.
func (b *BaseAgentBuilder) PostProcess(ctx *BuildContext, cmd *runnerv1.CreatePodCommand) error {
	return nil
}

// SupportsMcp returns true by default - most agents support MCP
func (b *BaseAgentBuilder) SupportsMcp() bool { return true }

// SupportsSkills returns true by default
func (b *BaseAgentBuilder) SupportsSkills() bool { return true }

// SupportsPlugin returns false by default - only Claude Code uses plugin dir
func (b *BaseAgentBuilder) SupportsPlugin() bool { return false }

// BuildResourcesToDownload returns nil by default - no resources to download
func (b *BaseAgentBuilder) BuildResourcesToDownload(ctx *BuildContext) ([]*runnerv1.ResourceToDownload, error) {
	return nil, nil
}

// renderTemplate renders a Go template string with the given context.
// Returns the original string if no template markers are present.
func (b *BaseAgentBuilder) renderTemplate(templateStr string, ctx map[string]interface{}) (string, error) {
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

// Helper method to get a config value as string
func (b *BaseAgentBuilder) getConfigString(config agent.ConfigValues, key string) string {
	if val, ok := config[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// Helper method to get a config value as bool
func (b *BaseAgentBuilder) getConfigBool(config agent.ConfigValues, key string) bool {
	if val, ok := config[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}
