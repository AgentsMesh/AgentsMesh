package agent

import (
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

const CodexCLISlug = "codex-cli"

// CodexCLIBuilder is the builder for Codex CLI agent.
// Codex CLI syntax: codex [prompt] [options]
// Similar to Claude Code, the prompt comes before options.
type CodexCLIBuilder struct {
	*BaseAgentBuilder
}

// NewCodexCLIBuilder creates a new CodexCLIBuilder
func NewCodexCLIBuilder() *CodexCLIBuilder {
	return &CodexCLIBuilder{
		BaseAgentBuilder: NewBaseAgentBuilder(CodexCLISlug),
	}
}

// Slug returns the agent type identifier
func (b *CodexCLIBuilder) Slug() string {
	return CodexCLISlug
}

// HandleInitialPrompt prepends the initial prompt to launch arguments.
// Codex CLI syntax: codex [prompt] [options]
func (b *CodexCLIBuilder) HandleInitialPrompt(ctx *BuildContext, args []string) []string {
	if ctx.Request.InitialPrompt != "" {
		return append([]string{ctx.Request.InitialPrompt}, args...)
	}
	return args
}

// BuildLaunchArgs uses the base implementation
func (b *CodexCLIBuilder) BuildLaunchArgs(ctx *BuildContext) ([]string, error) {
	return b.BaseAgentBuilder.BuildLaunchArgs(ctx)
}

// BuildFilesToCreate uses the base implementation
func (b *CodexCLIBuilder) BuildFilesToCreate(ctx *BuildContext) ([]*runnerv1.FileToCreate, error) {
	return b.BaseAgentBuilder.BuildFilesToCreate(ctx)
}

// BuildEnvVars uses the base implementation
func (b *CodexCLIBuilder) BuildEnvVars(ctx *BuildContext) (map[string]string, error) {
	return b.BaseAgentBuilder.BuildEnvVars(ctx)
}

// PostProcess uses the base implementation
func (b *CodexCLIBuilder) PostProcess(ctx *BuildContext, cmd *runnerv1.CreatePodCommand) error {
	return b.BaseAgentBuilder.PostProcess(ctx, cmd)
}
