package agent

import (
	"context"
	"fmt"
	"log/slog"

	agentDomain "github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	"github.com/anthropics/agentsmesh/agentfile/eval"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// buildEvalContext creates the AgentFile eval context with placeholder sandbox paths.
// CONFIG values are populated by CONFIG declarations during eval (resolve step already injected defaults).
func buildEvalContext(
	req *ConfigBuildRequest,
	creds agentDomain.EncryptedCredentials,
	isRunnerHost bool,
	builtinMCP, installedMCP map[string]interface{},
) *eval.Context {
	vars := map[string]interface{}{
		"config": make(map[string]interface{}), // filled by CONFIG declarations during eval
		"sandbox": map[string]interface{}{
			"root":     PlaceholderSandboxRoot,
			"work_dir": PlaceholderWorkDir,
		},
		"mcp": map[string]interface{}{
			"enabled":   false, // MCP ON declaration sets to true during eval
			"port":      fmt.Sprintf("%d", req.MCPPort),
			"builtin":   builtinMCP,
			"installed": installedMCP,
		},
		"pod": map[string]interface{}{
			"key": req.PodKey,
		},
	}

	ctx := eval.NewContext(vars)
	ctx.Credentials = credentialsToMap(creds)
	ctx.IsRunnerHost = isRunnerHost
	return ctx
}

// buildResultToProto converts eval.BuildResult to a CreatePodCommand proto.
// Paths contain placeholders ({{sandbox_root}}, {{work_dir}}) for Runner to resolve.
func buildResultToProto(
	req *ConfigBuildRequest,
	br *eval.BuildResult,
	creds agentDomain.EncryptedCredentials,
	isRunnerHost bool,
) *runnerv1.CreatePodCommand {
	// Convert dirs + files to proto FileToCreate list
	var files []*runnerv1.FileToCreate
	for _, dir := range br.Dirs {
		files = append(files, &runnerv1.FileToCreate{Path: dir, IsDirectory: true})
	}
	for _, f := range br.FilesToCreate {
		mode := int32(f.Mode)
		if mode == 0 {
			mode = 0644
		}
		files = append(files, &runnerv1.FileToCreate{
			Path: f.Path, Content: f.Content, Mode: mode,
		})
	}

	// Determine prompt from AgentFile PROMPT declaration
	prompt := br.Prompt
	if prompt == "" {
		prompt = req.Prompt
	}

	// Prompt injection into LaunchArgs is handled by Runner (based on PromptPosition).
	// Backend only passes Prompt and PromptPosition as separate proto fields.

	// Determine interaction mode
	mode := br.Mode
	if mode == "" {
		mode = "pty"
	}

	return &runnerv1.CreatePodCommand{
		PodKey:          req.PodKey,
		LaunchCommand:   br.LaunchCommand,
		LaunchArgs:      br.LaunchArgs,
		EnvVars:         br.EnvVars,
		FilesToCreate:   files,
		SandboxConfig:   buildSandboxConfig(req),
		Cols:            req.Cols,
		Rows:            req.Rows,
		InteractionMode: mode,
		Prompt:          prompt,
		PromptPosition:  br.PromptPosition,
		Credentials:     credentialsToMap(creds),
		IsRunnerHost:    isRunnerHost,
	}
}

// buildSandboxConfig builds sandbox config from request fields.
func buildSandboxConfig(req *ConfigBuildRequest) *runnerv1.SandboxConfig {
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
			slog.Warn("Failed to load MCP servers for agentfile", "error", err)
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

func credentialsToMap(creds agentDomain.EncryptedCredentials) map[string]string {
	if creds == nil {
		return nil
	}
	result := make(map[string]string, len(creds))
	for k, v := range creds {
		result[k] = v
	}
	return result
}
