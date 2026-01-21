package agent

import (
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

const OpenCodeSlug = "opencode"

// OpenCodeBuilder is the builder for OpenCode agent.
// OpenCode CLI syntax: opencode [prompt] [options]
// Similar to Claude Code, the prompt comes before options.
type OpenCodeBuilder struct {
	*BaseAgentBuilder
}

// NewOpenCodeBuilder creates a new OpenCodeBuilder
func NewOpenCodeBuilder() *OpenCodeBuilder {
	return &OpenCodeBuilder{
		BaseAgentBuilder: NewBaseAgentBuilder(OpenCodeSlug),
	}
}

// Slug returns the agent type identifier
func (b *OpenCodeBuilder) Slug() string {
	return OpenCodeSlug
}

// HandleInitialPrompt prepends the initial prompt to launch arguments.
// OpenCode syntax: opencode [prompt] [options]
func (b *OpenCodeBuilder) HandleInitialPrompt(ctx *BuildContext, args []string) []string {
	if ctx.Request.InitialPrompt != "" {
		return append([]string{ctx.Request.InitialPrompt}, args...)
	}
	return args
}

// BuildLaunchArgs uses the base implementation
func (b *OpenCodeBuilder) BuildLaunchArgs(ctx *BuildContext) ([]string, error) {
	return b.BaseAgentBuilder.BuildLaunchArgs(ctx)
}

// BuildFilesToCreate uses the base implementation
func (b *OpenCodeBuilder) BuildFilesToCreate(ctx *BuildContext) ([]*runnerv1.FileToCreate, error) {
	return b.BaseAgentBuilder.BuildFilesToCreate(ctx)
}

// BuildEnvVars uses the base implementation
func (b *OpenCodeBuilder) BuildEnvVars(ctx *BuildContext) (map[string]string, error) {
	return b.BaseAgentBuilder.BuildEnvVars(ctx)
}

// PostProcess uses the base implementation
func (b *OpenCodeBuilder) PostProcess(ctx *BuildContext, cmd *runnerv1.CreatePodCommand) error {
	return b.BaseAgentBuilder.PostProcess(ctx, cmd)
}
