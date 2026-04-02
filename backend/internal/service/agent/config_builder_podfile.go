package agent

import (
	"context"
	"fmt"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	"github.com/anthropics/agentsmesh/podfile/eval"
	"github.com/anthropics/agentsmesh/podfile/parser"
	"github.com/anthropics/agentsmesh/podfile/resolve"
	"github.com/anthropics/agentsmesh/podfile/serialize"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// Sandbox path placeholders — Runner replaces with real paths after sandbox setup.
const (
	PlaceholderSandboxRoot = "{{sandbox_root}}"
	PlaceholderWorkDir     = "{{work_dir}}"
)

// buildFromPodFile evaluates the agent's PodFile with placeholder sandbox paths
// and produces a complete CreatePodCommand. Runner only needs to substitute
// placeholders with real paths — no PodFile parsing needed on Runner side.
func (b *ConfigBuilder) buildFromPodFile(
	ctx context.Context,
	req *ConfigBuildRequest,
	agentDef *agent.Agent,
) (*runnerv1.CreatePodCommand, error) {
	mergedSource := req.MergedPodfileSource
	if mergedSource == "" {
		// Fallback: no PodFile Layer (resume mode). Resolve ConfigOverrides into base PodFile.
		if agentDef.PodfileSource == nil {
			return nil, fmt.Errorf("agent %s has no podfile source", req.AgentSlug)
		}
		baseProg, errs := parser.Parse(*agentDef.PodfileSource)
		if len(errs) > 0 {
			return nil, fmt.Errorf("podfile parse error: %v", errs[0])
		}
		// Fallback: no PodFile Layer (resume mode).
		// userPrefs intentionally omitted — resume continues the previous Pod's config,
		// not the user's current preferences.
		resolve.ResolveConfigValues(baseProg, nil, nil, req.ConfigOverrides)
		mergedSource = serialize.Serialize(baseProg)
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

	// Parse and eval PodFile with placeholder context
	prog, errs := parser.Parse(mergedSource)
	if len(errs) > 0 {
		return nil, fmt.Errorf("podfile parse error: %v", errs[0])
	}

	evalCtx := buildEvalContext(req, creds, isRunnerHost, builtinMCP, installedMCP)
	if err := eval.Eval(prog, evalCtx); err != nil {
		return nil, fmt.Errorf("podfile eval error: %w", err)
	}
	eval.ApplyModeArgs(evalCtx.Result)
	eval.ApplyRemoves(evalCtx.Result)

	return buildResultToProto(req, evalCtx.Result, creds, isRunnerHost), nil
}
