package agent

import (
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// AgentBuilder defines the strategy interface for building pod configurations.
// Each Agent type can have its own implementation to handle specific behaviors
// like InitialPrompt handling, environment variables, and file generation.
type AgentBuilder interface {
	// Slug returns the agent type identifier (e.g., "claude-code", "aider")
	Slug() string

	// BuildLaunchArgs constructs launch arguments from configuration.
	// Default implementation uses CommandTemplate; subclasses can override.
	BuildLaunchArgs(ctx *BuildContext) ([]string, error)

	// BuildFilesToCreate constructs configuration files to create.
	// Default implementation uses FilesTemplate; subclasses can override.
	BuildFilesToCreate(ctx *BuildContext) ([]*runnerv1.FileToCreate, error)

	// BuildEnvVars constructs environment variables for the pod.
	// Default implementation maps credentials based on CredentialSchema.
	BuildEnvVars(ctx *BuildContext) (map[string]string, error)

	// HandleInitialPrompt processes the initial prompt.
	// Different agents handle prompts differently:
	// - Claude Code: prepend to args (claude [prompt] [options])
	// - Gemini CLI: append to args (gemini [options] [prompt])
	// - Aider: does not support command-line prompt
	HandleInitialPrompt(ctx *BuildContext, args []string) []string

	// PostProcess allows final adjustments to the CreatePodCommand.
	// Called after all other build steps are complete.
	PostProcess(ctx *BuildContext, cmd *runnerv1.CreatePodCommand) error

	// SupportsMcp returns whether this agent type supports MCP servers
	SupportsMcp() bool

	// SupportsSkills returns whether this agent type supports Skills
	SupportsSkills() bool

	// SupportsPlugin returns whether this agent type supports plugin directory mode
	SupportsPlugin() bool

	// BuildResourcesToDownload constructs resources that need to be downloaded for the pod
	BuildResourcesToDownload(ctx *BuildContext) ([]*runnerv1.ResourceToDownload, error)
}
