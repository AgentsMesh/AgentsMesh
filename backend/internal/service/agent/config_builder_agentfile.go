package agent

import (
	"context"
	"fmt"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	"github.com/anthropics/agentsmesh/agentfile/eval"
	"github.com/anthropics/agentsmesh/agentfile/parser"
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

	return buildResultToProto(req, evalCtx.Result, creds, isRunnerHost), nil
}
