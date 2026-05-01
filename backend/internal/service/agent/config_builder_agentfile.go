package agent

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/anthropics/agentsmesh/agentfile/eval"
	"github.com/anthropics/agentsmesh/agentfile/parser"
	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// Sandbox path placeholders — Runner replaces with real paths after sandbox setup.
const (
	PlaceholderSandboxRoot = "{{sandbox_root}}"
	PlaceholderWorkDir     = "{{work_dir}}"
)

// buildFromAgentfile evaluates the agent's AgentFile with placeholder sandbox paths
// and produces a complete CreatePodCommand. Runner only needs to substitute
// placeholders with real paths — no AgentFile parsing needed on Runner side.
func (b *ConfigBuilder) buildFromAgentfile(
	ctx context.Context,
	req *ConfigBuildRequest,
	agentDef *agent.Agent,
) (*runnerv1.CreatePodCommand, error) {
	mergedSource := req.MergedAgentfileSource
	if mergedSource == "" {
		return nil, fmt.Errorf("agent %s: MergedAgentfileSource is empty (AgentFile resolve should always produce it)", req.AgentSlug)
	}

	// Get credentials
	var creds agent.EncryptedCredentials
	var isRunnerHost bool
	var err error
	if req.CredentialProfile != "" {
		creds, isRunnerHost, err = b.provider.ResolveCredentialsByName(ctx, req.UserID, req.AgentSlug, req.CredentialProfile)
	} else {
		creds, isRunnerHost, err = b.provider.GetEffectiveCredentialsForPod(ctx, req.UserID, req.AgentSlug, req.CredentialProfileID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get credentials: %w", err)
	}

	// Build MCP context
	builtinMCP, installedMCP := b.buildMCPContext(ctx, req, agentDef.Slug)

	// Parse and eval AgentFile with placeholder context
	prog, errs := parser.Parse(mergedSource)
	if len(errs) > 0 {
		return nil, fmt.Errorf("agentfile parse error: %v", errs[0])
	}

	evalCtx := buildEvalContext(req, creds, isRunnerHost, builtinMCP, installedMCP)
	if err := eval.Eval(prog, evalCtx); err != nil {
		return nil, fmt.Errorf("agentfile eval error: %w", err)
	}
	eval.ApplyModeArgs(evalCtx.Result)
	eval.ApplyRemoves(evalCtx.Result)

	// AgentFile SETUP is the most specific source for preparation scripts.
	// Preserve repository-level preparation as a fallback when SETUP is absent.
	effectiveReq := *req
	if evalCtx.Result.Setup.Script != "" {
		effectiveReq.PreparationScript = evalCtx.Result.Setup.Script
		effectiveReq.PreparationTimeout = evalCtx.Result.Setup.Timeout
	}

	cmd := buildResultToProto(&effectiveReq, evalCtx.Result, creds, isRunnerHost)
	cmd.ResourcesToDownload = b.buildSkillResources(ctx, req, agentDef.Slug)
	return cmd, nil
}

func (b *ConfigBuilder) buildSkillResources(ctx context.Context, req *ConfigBuildRequest, agentSlug string) []*runnerv1.ResourceToDownload {
	if b.extensionProvider == nil || req.RepositoryID == nil {
		return nil
	}

	skills, err := b.extensionProvider.GetEffectiveSkills(ctx, req.OrganizationID, req.UserID, *req.RepositoryID, agentSlug)
	if err != nil {
		slog.WarnContext(ctx, "Failed to load skills for agentfile", "agent_slug", agentSlug, "error", err)
		return nil
	}

	resources := make([]*runnerv1.ResourceToDownload, 0, len(skills))
	for _, skill := range skills {
		if skill == nil {
			continue
		}
		if skill.ContentSha == "" || skill.DownloadURL == "" || skill.Slug == "" {
			slog.WarnContext(ctx, "Skipping skill with incomplete download metadata",
				"agent_slug", agentSlug, "skill_slug", skill.Slug)
			continue
		}
		resources = append(resources, &runnerv1.ResourceToDownload{
			Sha:          skill.ContentSha,
			DownloadUrl:  skill.DownloadURL,
			TargetPath:   skillTargetPath(agentSlug, skill.Slug),
			ResourceType: "skill_package",
			SizeBytes:    skill.PackageSize,
		})
	}
	return resources
}

func skillTargetPath(agentSlug, skillSlug string) string {
	switch agentSlug {
	case "codex-cli", "codex":
		return "{{.sandbox.root_path}}/codex-home/skills/" + skillSlug
	case "claude-code", "claude":
		return "{{.sandbox.work_dir}}/.claude/skills/" + skillSlug
	default:
		return "{{.sandbox.root_path}}/skills/" + skillSlug
	}
}
