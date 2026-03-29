package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	"github.com/anthropics/agentsmesh/podfile/extract"
	"github.com/anthropics/agentsmesh/podfile/parser"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// buildFromPodFile builds the CreatePodCommand from PodFile source + context.
// MergedPodfileSource (from orchestrator) is preferred. Falls back to base PodFile for resume mode.
func (b *ConfigBuilder) buildFromPodFile(
	ctx context.Context,
	req *ConfigBuildRequest,
	agentDef *agent.Agent,
) (*runnerv1.CreatePodCommand, error) {
	mergedSource := req.MergedPodfileSource
	if mergedSource == "" {
		// Resume mode or no PodfileLayer: use base PodFile directly
		if agentDef.PodfileSource == nil {
			return nil, fmt.Errorf("agent %s has no podfile source", req.AgentSlug)
		}
		mergedSource = *agentDef.PodfileSource
	}

	// Extract CREDENTIAL from merged source
	prog, errs := parser.Parse(mergedSource)
	if len(errs) > 0 {
		return nil, fmt.Errorf("podfile parse error: %v", errs[0])
	}
	spec := extract.Extract(prog)

	// Get credentials — PodFile CREDENTIAL overrides req.CredentialProfileID
	var creds agent.EncryptedCredentials
	var isRunnerHost bool
	var err error
	if spec.CredentialProfile != "" {
		creds, isRunnerHost, err = b.provider.ResolveCredentialsByName(ctx, req.UserID, req.AgentSlug, spec.CredentialProfile)
	} else {
		creds, isRunnerHost, err = b.provider.GetEffectiveCredentialsForPod(ctx, req.UserID, req.AgentSlug, req.CredentialProfileID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get credentials: %w", err)
	}

	// Build MCP context as JSON
	builtinMCP, installedMCP := b.buildMCPContext(ctx, req, agentDef.Slug)
	builtinJSON, _ := json.Marshal(builtinMCP)
	installedJSON, _ := json.Marshal(installedMCP)

	// Convert config values to string map for proto
	config := b.provider.GetUserEffectiveConfig(ctx, req.UserID, req.AgentSlug, agent.ConfigValues(req.ConfigOverrides))
	configValues := configToStringMap(config)

	// Build sandbox config (repo/branch/git creds)
	sandboxConfig := b.buildSandboxConfig(req)

	return &runnerv1.CreatePodCommand{
		PodKey:           req.PodKey,
		PodfileSource:    mergedSource,
		ConfigValues:     configValues,
		Credentials:      credentialsToMap(creds),
		IsRunnerHost:     isRunnerHost,
		McpPort:          int32(req.MCPPort),
		McpBuiltinJson:   string(builtinJSON),
		McpInstalledJson: string(installedJSON),
		SandboxConfig:    sandboxConfig,
		InitialPrompt:    req.InitialPrompt,
		Cols:             req.Cols,
		Rows:             req.Rows,
	}, nil
}

func configToStringMap(config agent.ConfigValues) map[string]string {
	result := make(map[string]string, len(config))
	for k, v := range config {
		switch val := v.(type) {
		case string:
			result[k] = val
		case bool:
			result[k] = fmt.Sprintf("%t", val)
		case float64:
			result[k] = fmt.Sprintf("%v", val)
		default:
			b, _ := json.Marshal(val)
			result[k] = string(b)
		}
	}
	return result
}

// buildSandboxConfig builds sandbox config from request fields.
func (b *ConfigBuilder) buildSandboxConfig(req *ConfigBuildRequest) *runnerv1.SandboxConfig {
	repoURL := req.RepositoryURL
	if repoURL == "" && req.HttpCloneURL == "" && req.SshCloneURL == "" && req.LocalPath == "" {
		return nil
	}

	timeout := int32(req.PreparationTimeout)
	if timeout <= 0 {
		timeout = 300
	}

	return &runnerv1.SandboxConfig{
		RepositoryUrl:      repoURL,
		HttpCloneUrl:       req.HttpCloneURL,
		SshCloneUrl:        req.SshCloneURL,
		SourceBranch:       req.SourceBranch,
		CredentialType:     req.CredentialType,
		GitToken:           req.GitToken,
		SshPrivateKey:      req.SSHPrivateKey,
		TicketSlug:         req.TicketSlug,
		PreparationScript:  req.PreparationScript,
		PreparationTimeout: timeout,
		LocalPath:          req.LocalPath,
	}
}

// buildMCPContext loads MCP server configurations.
func (b *ConfigBuilder) buildMCPContext(ctx context.Context, req *ConfigBuildRequest, agentSlug string) (map[string]interface{}, map[string]interface{}) {
	builtinMCP := map[string]interface{}{
		"agentsmesh": map[string]interface{}{
			"type": "http",
			"url":  fmt.Sprintf("http://127.0.0.1:%d/mcp", req.MCPPort),
			"headers": map[string]interface{}{
				"X-Pod-Key": req.PodKey,
			},
		},
	}

	installedMCP := map[string]interface{}{}
	if b.extensionProvider != nil && req.RepositoryID != nil {
		servers, err := b.extensionProvider.GetEffectiveMcpServers(ctx, req.OrganizationID, req.UserID, *req.RepositoryID, agentSlug)
		if err != nil {
			slog.Warn("Failed to load MCP servers for podfile", "error", err)
		} else {
			for _, srv := range servers {
				if !srv.IsEnabled {
					continue
				}
				installedMCP[srv.Slug] = srv.ToMcpConfig()
			}
		}
	}

	return builtinMCP, installedMCP
}

func credentialsToMap(creds agent.EncryptedCredentials) map[string]string {
	if creds == nil {
		return nil
	}
	result := make(map[string]string, len(creds))
	for k, v := range creds {
		result[k] = v
	}
	return result
}
