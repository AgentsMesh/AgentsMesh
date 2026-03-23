package agent

import (
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

const GeminiCLISlug = "gemini-cli"

// GeminiCLIBuilder is the builder for Gemini CLI agent.
// Gemini CLI syntax: gemini [options] [prompt]
// Unlike Claude Code, the prompt comes AFTER options.
type GeminiCLIBuilder struct {
	*BaseAgentBuilder
}

// NewGeminiCLIBuilder creates a new GeminiCLIBuilder
func NewGeminiCLIBuilder() *GeminiCLIBuilder {
	return &GeminiCLIBuilder{
		BaseAgentBuilder: NewBaseAgentBuilder(GeminiCLISlug),
	}
}

// Slug returns the agent type identifier
func (b *GeminiCLIBuilder) Slug() string {
	return GeminiCLISlug
}

// HandleInitialPrompt appends the initial prompt to launch arguments.
// Gemini CLI syntax: gemini [options] [prompt]
// In ACP mode, the prompt is sent via JSON-RPC (session/prompt), not CLI args.
func (b *GeminiCLIBuilder) HandleInitialPrompt(ctx *BuildContext, args []string) []string {
	if ctx.Request.InteractionMode == "acp" {
		return args
	}
	if ctx.Request.InitialPrompt != "" {
		return append(args, ctx.Request.InitialPrompt)
	}
	return args
}

// BuildLaunchArgs uses the base implementation
func (b *GeminiCLIBuilder) BuildLaunchArgs(ctx *BuildContext) ([]string, error) {
	return b.BaseAgentBuilder.BuildLaunchArgs(ctx)
}

// BuildFilesToCreate uses the base implementation
func (b *GeminiCLIBuilder) BuildFilesToCreate(ctx *BuildContext) ([]*runnerv1.FileToCreate, error) {
	return b.BaseAgentBuilder.BuildFilesToCreate(ctx)
}

// BuildEnvVars uses the base implementation
func (b *GeminiCLIBuilder) BuildEnvVars(ctx *BuildContext) (map[string]string, error) {
	return b.BaseAgentBuilder.BuildEnvVars(ctx)
}

// PostProcess appends the --acp flag in ACP mode.
// Gemini CLI natively supports ACP JSON-RPC 2.0 via --acp.
func (b *GeminiCLIBuilder) PostProcess(ctx *BuildContext, cmd *runnerv1.CreatePodCommand) error {
	if ctx.Request.InteractionMode == "acp" {
		cmd.LaunchArgs = append(cmd.LaunchArgs, "--acp")
	}
	return nil
}
