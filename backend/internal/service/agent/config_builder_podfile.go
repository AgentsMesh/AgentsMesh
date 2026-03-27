package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	"github.com/anthropics/agentsmesh/podfile/merge"
	"github.com/anthropics/agentsmesh/podfile/parser"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// buildFromPodFile merges PodFile layers and sends merged source + context to Runner.
// Runner evaluates the PodFile with real sandbox paths.
func (b *ConfigBuilder) buildFromPodFile(
	ctx context.Context,
	req *ConfigBuildRequest,
	agentDef *agent.Agent,
) (*runnerv1.CreatePodCommand, error) {
	// 1. Parse base PodFile
	baseProg, errs := parser.Parse(*agentDef.PodfileSource)
	if len(errs) > 0 {
		return nil, fmt.Errorf("podfile parse errors: %v", errs)
	}

	// 2. Merge config: defaults + user personal config + request overrides
	config := b.provider.GetUserEffectiveConfig(ctx, req.UserID, req.AgentSlug, agent.ConfigValues(req.ConfigOverrides))

	// 3. Build user layer from config overrides + repo/branch
	userLayerSrc := buildUserLayer(config, req)
	userProg, parseErrors := parser.Parse(userLayerSrc)
	if len(parseErrors) > 0 {
		return nil, fmt.Errorf("invalid user config layer: %s", parseErrors[0])
	}

	// 4. Validate AST merge (for future AST serializer)
	_ = merge.Merge(baseProg, userProg)

	// 5. Send concatenated source to Runner.
	// Runner re-parses and evals the full source. Concatenation works correctly because:
	// - Declaration eval uses assignment semantics (later overrides earlier)
	// - SKILLS eval uses append (union semantics, same as merge)
	// - REMOVE is collected and applied post-eval via ApplyRemoves
	// - Statements (arg, file, etc.) are appended, which is the intended behavior
	// TODO: implement AST serializer to replace concatenation for full correctness
	mergedSource := *agentDef.PodfileSource + "\n" + userLayerSrc

	// 6. Get credentials
	creds, isRunnerHost, err := b.provider.GetEffectiveCredentialsForPod(ctx, req.UserID, req.AgentSlug, req.CredentialProfileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get credentials: %w", err)
	}

	// 7. Build MCP context as JSON
	builtinMCP, installedMCP := b.buildMCPContext(ctx, req, agentDef.Slug)
	builtinJSON, _ := json.Marshal(builtinMCP)
	installedJSON, _ := json.Marshal(installedMCP)

	// 8. Convert config values to string map for proto
	configValues := configToStringMap(config)

	// 9. Build sandbox config (repo/branch/git creds — not in PodFile)
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
		InteractionMode:  req.InteractionMode,
		Cols:             req.Cols,
		Rows:             req.Rows,
	}, nil
}

// buildUserLayer constructs a PodFile Layer from user config overrides and repo info.
func buildUserLayer(config agent.ConfigValues, req *ConfigBuildRequest) string {
	var lines []string
	for k, v := range config {
		lines = append(lines, fmt.Sprintf("CONFIG %s = %s", k, formatLiteralValue(v)))
	}
	if req.RepositoryURL != "" {
		lines = append(lines, fmt.Sprintf("REPO \"%s\"", req.RepositoryURL))
	}
	if req.SourceBranch != "" {
		lines = append(lines, fmt.Sprintf("BRANCH \"%s\"", req.SourceBranch))
	}
	if req.CredentialType != "" {
		lines = append(lines, fmt.Sprintf("GIT_CREDENTIAL %s", req.CredentialType))
	}
	return strings.Join(lines, "\n")
}

func formatLiteralValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		// Escape special characters to prevent PodFile injection
		escaped := strings.ReplaceAll(val, `\`, `\\`)
		escaped = strings.ReplaceAll(escaped, `"`, `\"`)
		escaped = strings.ReplaceAll(escaped, "\n", `\n`)
		escaped = strings.ReplaceAll(escaped, "\t", `\t`)
		return fmt.Sprintf("\"%s\"", escaped)
	case bool:
		if val {
			return "true"
		}
		return "false"
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%g", val)
	default:
		return fmt.Sprintf("\"%v\"", val)
	}
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

// buildSandboxConfig builds sandbox config, using PodFile declarations as fallback.
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
