package agent

import (
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

const AiderSlug = "aider"

// AiderBuilder is the builder for Aider agent.
// Aider does NOT support command-line prompts.
// To provide an initial prompt, you would need to use --message-file or similar.
type AiderBuilder struct {
	*BaseAgentBuilder
}

// NewAiderBuilder creates a new AiderBuilder
func NewAiderBuilder() *AiderBuilder {
	return &AiderBuilder{
		BaseAgentBuilder: NewBaseAgentBuilder(AiderSlug),
	}
}

// Slug returns the agent type identifier
func (b *AiderBuilder) Slug() string {
	return AiderSlug
}

// HandleInitialPrompt does nothing for Aider.
// Aider does not support command-line prompts.
// The prompt is ignored - users should interact via the terminal.
func (b *AiderBuilder) HandleInitialPrompt(ctx *BuildContext, args []string) []string {
	// Aider does not support command-line prompt, return args unchanged
	return args
}

// BuildLaunchArgs uses the base implementation
func (b *AiderBuilder) BuildLaunchArgs(ctx *BuildContext) ([]string, error) {
	return b.BaseAgentBuilder.BuildLaunchArgs(ctx)
}

// BuildFilesToCreate returns nil for Aider.
// Aider does not require MCP configuration files by default.
func (b *AiderBuilder) BuildFilesToCreate(ctx *BuildContext) ([]*runnerv1.FileToCreate, error) {
	// Aider doesn't use MCP config files, but we still use base implementation
	// in case FilesTemplate is configured in the database
	return b.BaseAgentBuilder.BuildFilesToCreate(ctx)
}

// BuildEnvVars uses the base implementation
func (b *AiderBuilder) BuildEnvVars(ctx *BuildContext) (map[string]string, error) {
	return b.BaseAgentBuilder.BuildEnvVars(ctx)
}

// PostProcess uses the base implementation
func (b *AiderBuilder) PostProcess(ctx *BuildContext, cmd *runnerv1.CreatePodCommand) error {
	return b.BaseAgentBuilder.PostProcess(ctx, cmd)
}

// SupportsMcp returns false - Aider does not support MCP
func (b *AiderBuilder) SupportsMcp() bool { return false }

// SupportsSkills returns false - Aider does not support Skills
func (b *AiderBuilder) SupportsSkills() bool { return false }

// SupportsPlugin returns false - Aider does not support plugin directory mode
func (b *AiderBuilder) SupportsPlugin() bool { return false }
