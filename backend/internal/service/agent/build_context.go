package agent

import (
	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
	extensionservice "github.com/anthropics/agentsmesh/backend/internal/service/extension"
)

// BuildContext contains all data needed during the pod configuration build process.
// It provides a unified context for AgentBuilder implementations.
type BuildContext struct {
	// Request contains the original build request
	Request *ConfigBuildRequest

	// AgentType contains the agent type definition from database
	AgentType *agent.AgentType

	// Config contains the merged configuration values
	// (schema defaults + user config + request overrides)
	Config agent.ConfigValues

	// Credentials contains decrypted credential values
	Credentials agent.EncryptedCredentials

	// IsRunnerHost indicates if using Runner's local credentials
	// When true, credentials should not be injected as env vars
	IsRunnerHost bool

	// TemplateCtx contains the template rendering context
	// Includes: config, sandbox placeholders, mcp_port, pod_key
	TemplateCtx map[string]interface{}

	// AgentVersion is the version of the target agent on the Runner.
	// Empty string means the Runner did not report version info (old Runner).
	// Used by AgentBuilder to adapt CLI arguments for version compatibility.
	AgentVersion string

	// McpServers contains installed MCP servers for this repo
	McpServers []*extension.InstalledMcpServer

	// ResolvedSkills contains resolved skills with download URLs
	ResolvedSkills []*extensionservice.ResolvedSkill
}

// NewBuildContext creates a new BuildContext with the given parameters
func NewBuildContext(
	req *ConfigBuildRequest,
	agentType *agent.AgentType,
	config agent.ConfigValues,
	credentials agent.EncryptedCredentials,
	isRunnerHost bool,
	templateCtx map[string]interface{},
	agentVersion string,
) *BuildContext {
	return &BuildContext{
		Request:        req,
		AgentType:      agentType,
		Config:         config,
		Credentials:    credentials,
		IsRunnerHost:   isRunnerHost,
		TemplateCtx:    templateCtx,
		AgentVersion:   agentVersion,
		McpServers:     make([]*extension.InstalledMcpServer, 0),
		ResolvedSkills: make([]*extensionservice.ResolvedSkill, 0),
	}
}
