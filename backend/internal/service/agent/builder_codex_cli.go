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

// codexApprovalValueMap maps internal approval mode names to Codex CLI 0.100+ policy values.
// Codex CLI 0.100.0 (Rust rewrite) renamed the approval policy values:
//   - suggest   → on-request  (agent proposes, user approves)
//   - auto-edit → on-failure  (auto-proceed, stop only on failure)
//   - full-auto → never       (fully autonomous)
var codexApprovalValueMap = map[string]string{
	"suggest":   "on-request",
	"auto-edit": "on-failure",
	"full-auto": "never",
}

// codexNewApprovalVersion is the first Codex CLI version that uses the new approval policy values.
const codexNewApprovalVersion = "0.100.0"

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

	// Codex CLI >= 0.100.0 (Rust rewrite) renamed approval policy values.
	// Map internal names (suggest/auto-edit/full-auto) to new CLI values.
	if ctx.AgentVersion != "" && CompareVersions(ctx.AgentVersion, codexNewApprovalVersion) >= 0 {
		args = mapCodexApprovalValues(args)
	}

	return args, nil
}

// mapCodexApprovalValues replaces internal approval mode names with Codex CLI 0.100+ values.
func mapCodexApprovalValues(args []string) []string {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "--ask-for-approval" {
			if mapped, ok := codexApprovalValueMap[args[i+1]]; ok {
				result := make([]string, len(args))
				copy(result, args)
				result[i+1] = mapped
				return result
			}
		}
	}
	return args
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
