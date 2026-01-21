package agent

import (
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

const ClaudeCodeSlug = "claude-code"

// ClaudeCodeBuilder is the builder for Claude Code agent.
// Claude Code CLI syntax: claude [prompt] [options]
// The prompt must come BEFORE options.
type ClaudeCodeBuilder struct {
	*BaseAgentBuilder
}

// NewClaudeCodeBuilder creates a new ClaudeCodeBuilder
func NewClaudeCodeBuilder() *ClaudeCodeBuilder {
	return &ClaudeCodeBuilder{
		BaseAgentBuilder: NewBaseAgentBuilder(ClaudeCodeSlug),
	}
}

// Slug returns the agent type identifier
func (b *ClaudeCodeBuilder) Slug() string {
	return ClaudeCodeSlug
}

// HandleInitialPrompt prepends the initial prompt to launch arguments.
// Claude Code syntax: claude [prompt] [options]
func (b *ClaudeCodeBuilder) HandleInitialPrompt(ctx *BuildContext, args []string) []string {
	if ctx.Request.InitialPrompt != "" {
		return append([]string{ctx.Request.InitialPrompt}, args...)
	}
	return args
}

// BuildLaunchArgs uses the base implementation
func (b *ClaudeCodeBuilder) BuildLaunchArgs(ctx *BuildContext) ([]string, error) {
	return b.BaseAgentBuilder.BuildLaunchArgs(ctx)
}

// BuildFilesToCreate uses the base implementation
func (b *ClaudeCodeBuilder) BuildFilesToCreate(ctx *BuildContext) ([]*runnerv1.FileToCreate, error) {
	return b.BaseAgentBuilder.BuildFilesToCreate(ctx)
}

// BuildEnvVars uses the base implementation
func (b *ClaudeCodeBuilder) BuildEnvVars(ctx *BuildContext) (map[string]string, error) {
	return b.BaseAgentBuilder.BuildEnvVars(ctx)
}

// PostProcess uses the base implementation
func (b *ClaudeCodeBuilder) PostProcess(ctx *BuildContext, cmd *runnerv1.CreatePodCommand) error {
	return b.BaseAgentBuilder.PostProcess(ctx, cmd)
}
