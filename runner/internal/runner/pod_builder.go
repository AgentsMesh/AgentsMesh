package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/terminal"
	"github.com/anthropics/agentsmesh/runner/internal/workspace"
)

// PodBuilder builds pods using the Builder pattern.
// It provides a fluent API for configuring and creating pods.
type PodBuilder struct {
	runner *Runner

	// Pod configuration
	podKey        string
	agentType     string
	launchCommand string
	launchArgs    []string
	envVars       map[string]string
	rows          int
	cols          int

	// New protocol fields
	filesToCreate []client.FileToCreate
	workDirConfig *client.WorkDirConfig
}

// NewPodBuilder creates a new pod builder.
func NewPodBuilder(runner *Runner) *PodBuilder {
	return &PodBuilder{
		runner:  runner,
		envVars: make(map[string]string),
		rows:    24,
		cols:    80,
	}
}

// WithPodKey sets the pod key.
func (b *PodBuilder) WithPodKey(key string) *PodBuilder {
	b.podKey = key
	return b
}

// WithAgentType sets the agent type.
func (b *PodBuilder) WithAgentType(agentType string) *PodBuilder {
	b.agentType = agentType
	return b
}

// WithLaunchCommand sets the command to launch.
func (b *PodBuilder) WithLaunchCommand(command string, args []string) *PodBuilder {
	b.launchCommand = command
	b.launchArgs = args
	return b
}

// WithEnvVars sets environment variables.
func (b *PodBuilder) WithEnvVars(envVars map[string]string) *PodBuilder {
	for k, v := range envVars {
		b.envVars[k] = v
	}
	return b
}

// WithEnvVar adds a single environment variable.
func (b *PodBuilder) WithEnvVar(key, value string) *PodBuilder {
	b.envVars[key] = value
	return b
}

// WithTerminalSize sets terminal dimensions.
func (b *PodBuilder) WithTerminalSize(rows, cols int) *PodBuilder {
	if rows > 0 {
		b.rows = rows
	}
	if cols > 0 {
		b.cols = cols
	}
	return b
}

// WithFilesToCreate sets the files to create in the sandbox.
func (b *PodBuilder) WithFilesToCreate(files []client.FileToCreate) *PodBuilder {
	b.filesToCreate = files
	return b
}

// WithWorkDirConfig sets the working directory configuration.
func (b *PodBuilder) WithWorkDirConfig(config *client.WorkDirConfig) *PodBuilder {
	b.workDirConfig = config
	return b
}

// WithNewProtocol is kept for API compatibility but is now a no-op.
// All pods now use the new protocol.
func (b *PodBuilder) WithNewProtocol(useNew bool) *PodBuilder {
	return b
}

// Build creates the pod.
func (b *PodBuilder) Build(ctx context.Context) (*Pod, error) {
	if b.podKey == "" {
		return nil, fmt.Errorf("pod key is required")
	}
	if b.launchCommand == "" {
		return nil, fmt.Errorf("launch command is required")
	}

	logger.Pod().Info("Building pod", "pod_key", b.podKey, "command", b.launchCommand)

	// Setup sandbox and working directory
	sandboxRoot, workingDir, worktreePath, branchName, err := b.setup(ctx)
	if err != nil {
		return nil, err
	}

	// Resolve template variables in launch args
	resolvedArgs := b.resolveArgs(b.launchArgs, sandboxRoot, workingDir)

	// Merge environment variables
	envVars := b.mergeEnvVars()

	// Create terminal
	term, err := terminal.New(terminal.Options{
		Command:  b.launchCommand,
		Args:     resolvedArgs,
		WorkDir:  workingDir,
		Env:      envVars,
		Rows:     b.rows,
		Cols:     b.cols,
		OnOutput: nil, // Will be set by caller
		OnExit:   nil, // Will be set by caller
	})
	if err != nil {
		// Cleanup sandbox on failure
		if sandboxRoot != "" {
			os.RemoveAll(sandboxRoot)
		}
		return nil, &client.PodError{
			Code:    client.ErrCodeCommandStart,
			Message: fmt.Sprintf("failed to create terminal: %v", err),
		}
	}

	// Create pod
	pod := &Pod{
		ID:           b.podKey,
		PodKey:       b.podKey,
		AgentType:    b.agentType,
		Branch:       branchName,
		WorktreePath: worktreePath,
		Terminal:     term,
		StartedAt:    time.Now(),
		Status:       PodStatusInitializing,
	}

	logger.Pod().Info("Pod built", "pod_key", b.podKey, "working_dir", workingDir)

	return pod, nil
}

// setup sets up the sandbox and working directory.
// Returns (sandboxRoot, workingDir, worktreePath, branchName, error).
func (b *PodBuilder) setup(ctx context.Context) (string, string, string, string, error) {
	// 1. Create sandbox root directory
	sandboxRoot := filepath.Join(b.runner.cfg.WorkspaceRoot, "sandboxes", b.podKey)
	if err := os.MkdirAll(sandboxRoot, 0755); err != nil {
		return "", "", "", "", &client.PodError{
			Code:    client.ErrCodeSandboxCreate,
			Message: fmt.Sprintf("failed to create sandbox directory: %v", err),
		}
	}

	// 2. Setup working directory based on WorkDirConfig
	workingDir, worktreePath, branchName, err := b.setupWorkDir(ctx, sandboxRoot)
	if err != nil {
		os.RemoveAll(sandboxRoot)
		return "", "", "", "", err
	}

	// 3. Create files from FilesToCreate
	if err := b.createFiles(sandboxRoot, workingDir); err != nil {
		os.RemoveAll(sandboxRoot)
		return "", "", "", "", err
	}

	return sandboxRoot, workingDir, worktreePath, branchName, nil
}

// setupWorkDir sets up the working directory based on WorkDirConfig.
func (b *PodBuilder) setupWorkDir(ctx context.Context, sandboxRoot string) (string, string, string, error) {
	if b.workDirConfig == nil {
		// Default to tempdir
		tempDir := filepath.Join(sandboxRoot, "workspace")
		if err := os.MkdirAll(tempDir, 0755); err != nil {
			return "", "", "", &client.PodError{
				Code:    client.ErrCodeSandboxCreate,
				Message: fmt.Sprintf("failed to create temp workspace: %v", err),
			}
		}
		return tempDir, "", "", nil
	}

	switch b.workDirConfig.Type {
	case "worktree":
		return b.setupGitWorktree(ctx, sandboxRoot)
	case "tempdir":
		tempDir := filepath.Join(sandboxRoot, "workspace")
		if err := os.MkdirAll(tempDir, 0755); err != nil {
			return "", "", "", &client.PodError{
				Code:    client.ErrCodeSandboxCreate,
				Message: fmt.Sprintf("failed to create temp workspace: %v", err),
			}
		}
		return tempDir, "", "", nil
	case "local":
		localPath := b.workDirConfig.LocalPath
		if localPath == "" {
			localPath = b.runner.cfg.WorkspaceRoot
		}
		// Verify the path exists
		if _, err := os.Stat(localPath); os.IsNotExist(err) {
			return "", "", "", &client.PodError{
				Code:    client.ErrCodeWorkDirNotExist,
				Message: fmt.Sprintf("local path does not exist: %s", localPath),
				Details: map[string]string{"path": localPath},
			}
		}
		return localPath, "", "", nil
	default:
		return "", "", "", &client.PodError{
			Code:    client.ErrCodeUnknown,
			Message: fmt.Sprintf("unknown work_dir type: %s", b.workDirConfig.Type),
		}
	}
}

// setupGitWorktree creates a git worktree for the pod.
func (b *PodBuilder) setupGitWorktree(ctx context.Context, sandboxRoot string) (string, string, string, error) {
	cfg := b.workDirConfig
	if cfg.RepositoryURL == "" {
		return "", "", "", &client.PodError{
			Code:    client.ErrCodeGitClone,
			Message: "repository_url is required for worktree type",
		}
	}

	// Use workspace manager if available
	if b.runner.workspace != nil {
		// Set git credentials if provided
		opts := []workspace.WorktreeOption{}
		if cfg.GitToken != "" {
			opts = append(opts, workspace.WithGitToken(cfg.GitToken))
		}
		if cfg.SSHKeyPath != "" {
			opts = append(opts, workspace.WithSSHKeyPath(cfg.SSHKeyPath))
		}

		worktreePath, err := b.runner.workspace.CreateWorktreeWithOptions(
			ctx,
			cfg.RepositoryURL,
			cfg.Branch,
			b.podKey,
			opts...,
		)
		if err != nil {
			// Determine error type
			errMsg := err.Error()
			errCode := client.ErrCodeGitWorktree
			if strings.Contains(errMsg, "authentication") || strings.Contains(errMsg, "Permission denied") {
				errCode = client.ErrCodeGitAuth
			} else if strings.Contains(errMsg, "clone") {
				errCode = client.ErrCodeGitClone
			}
			return "", "", "", &client.PodError{
				Code:    errCode,
				Message: fmt.Sprintf("failed to create worktree: %v", err),
				Details: map[string]string{
					"repository": cfg.RepositoryURL,
					"branch":     cfg.Branch,
				},
			}
		}

		branchName := cfg.Branch
		if branchName == "" {
			branchName = "main"
		}
		return worktreePath, worktreePath, branchName, nil
	}

	// Fallback: workspace manager not available
	return "", "", "", &client.PodError{
		Code:    client.ErrCodeGitWorktree,
		Message: "workspace manager not available for git operations",
	}
}

// createFiles creates files from the FilesToCreate list.
func (b *PodBuilder) createFiles(sandboxRoot, workDir string) error {
	for _, f := range b.filesToCreate {
		// Resolve path template
		path := b.resolvePath(f.PathTemplate, sandboxRoot, workDir)

		if f.IsDirectory {
			if err := os.MkdirAll(path, 0755); err != nil {
				return &client.PodError{
					Code:    client.ErrCodeFileCreate,
					Message: fmt.Sprintf("failed to create directory: %v", err),
					Details: map[string]string{"path": path},
				}
			}
			continue
		}

		// Ensure parent directory exists
		parentDir := filepath.Dir(path)
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			return &client.PodError{
				Code:    client.ErrCodeFileCreate,
				Message: fmt.Sprintf("failed to create parent directory: %v", err),
				Details: map[string]string{"path": parentDir},
			}
		}

		// Determine file mode
		mode := os.FileMode(0644)
		if f.Mode != 0 {
			mode = os.FileMode(f.Mode)
		}

		// Write file
		if err := os.WriteFile(path, []byte(f.Content), mode); err != nil {
			return &client.PodError{
				Code:    client.ErrCodeFileCreate,
				Message: fmt.Sprintf("failed to write file: %v", err),
				Details: map[string]string{"path": path},
			}
		}

		logger.Pod().Debug("Created file", "path", path, "mode", fmt.Sprintf("%o", mode))
	}

	return nil
}

// resolvePath resolves path template variables.
func (b *PodBuilder) resolvePath(pathTemplate, sandboxRoot, workDir string) string {
	path := pathTemplate
	path = strings.ReplaceAll(path, "{{.sandbox.root_path}}", sandboxRoot)
	path = strings.ReplaceAll(path, "{{.sandbox.work_dir}}", workDir)
	return path
}

// resolveArgs resolves template variables in command line arguments.
func (b *PodBuilder) resolveArgs(args []string, sandboxRoot, workDir string) []string {
	resolved := make([]string, len(args))
	for i, arg := range args {
		resolved[i] = b.resolvePath(arg, sandboxRoot, workDir)
	}
	return resolved
}

// mergeEnvVars merges all environment variable sources.
func (b *PodBuilder) mergeEnvVars() map[string]string {
	result := make(map[string]string)

	// Add config env vars first (lowest priority)
	if b.runner.cfg != nil {
		for k, v := range b.runner.cfg.AgentEnvVars {
			result[k] = v
		}
	}

	// Add builder env vars (highest priority)
	for k, v := range b.envVars {
		result[k] = v
	}

	return result
}
