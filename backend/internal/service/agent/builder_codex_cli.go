package agent

import (
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

const CodexCLISlug = "codex-cli"

// codexVersionRules defines version-specific arg transformations for Codex CLI.
// The DB command_template uses the LATEST syntax; these rules downgrade for older versions.
//
// Codex CLI breaking changes:
//   - v0.1.2025042500: --approval-mode renamed to --ask-for-approval (-a)
var codexVersionRules = []VersionArgRule{
	{
		VersionBelow: "0.1.2025042500",
		Transforms: []ArgTransform{
			{
				OldFlag: "--approval-mode",
				NewFlag: "--ask-for-approval",
				// Values remain the same: suggest, auto-edit, full-auto
			},
		},
	},
}

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

// BuildLaunchArgs builds launch arguments with version-specific adaptation.
// Uses the base implementation to render from DB command_template (latest syntax),
// then applies version-specific transformations for older Codex CLI versions.
func (b *CodexCLIBuilder) BuildLaunchArgs(ctx *BuildContext) ([]string, error) {
	args, err := b.BaseAgentBuilder.BuildLaunchArgs(ctx)
	if err != nil {
		return nil, err
	}

	// Adapt args for the installed Codex CLI version
	args = AdaptArgsForVersion(args, ctx.AgentVersion, codexVersionRules)
	return args, nil
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
