// Package workspace provides workspace preparation utilities.
// Preparer executes initialization steps before agent starts.
package workspace

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/envfilter"
	"github.com/anthropics/agentsmesh/runner/internal/envpath"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// Module logger for workspace
var log = logger.Workspace()

// PreparationStep represents a single step in the workspace preparation process.
// Implements the Strategy pattern for extensible preparation steps.
type PreparationStep interface {
	// Name returns the step name for logging and error reporting.
	Name() string

	// Execute runs the preparation step.
	// Returns an error if the step fails, which should abort pod creation.
	Execute(ctx context.Context, prepCtx *PreparationContext) error
}

// PreparationContext contains all the context needed for workspace preparation.
type PreparationContext struct {
	PodID        string            // Pod identifier
	TicketSlug   string            // Ticket slug (e.g., "TBD-123")
	BranchName   string            // Git branch name
	WorkspaceDir string            // Workspace directory path
	MainRepoDir  string            // Path to the main git repository (for bare repo operations)
	BaseEnvVars  map[string]string // Base environment variables (e.g., AI provider credentials)
}

// GetEnvVars returns all environment variables for script execution.
// Includes base variables plus workspace-specific variables.
func (c *PreparationContext) GetEnvVars() map[string]string {
	result := make(map[string]string)

	// Copy base environment variables
	for k, v := range c.BaseEnvVars {
		result[k] = v
	}

	// Add workspace-specific variables
	result["WORKSPACE_DIR"] = c.WorkspaceDir
	if c.MainRepoDir != "" {
		result["MAIN_REPO_DIR"] = c.MainRepoDir
	}
	if c.TicketSlug != "" {
		result["TICKET_SLUG"] = c.TicketSlug
	}
	if c.BranchName != "" {
		result["BRANCH_NAME"] = c.BranchName
	}

	return result
}

// String returns a string representation for logging.
func (c *PreparationContext) String() string {
	return fmt.Sprintf(
		"PreparationContext{PodID: %s, Ticket: %s, WorkspaceDir: %s}",
		c.PodID, c.TicketSlug, c.WorkspaceDir,
	)
}

// PreparationError represents an error during workspace preparation.
type PreparationError struct {
	Step   string // Name of the step that failed
	Cause  error  // Underlying error
	Output string // Command output if applicable
}

// Error implements the error interface.
func (e *PreparationError) Error() string {
	if e.Output != "" {
		return fmt.Sprintf("preparation step '%s' failed: %v\nOutput: %s", e.Step, e.Cause, e.Output)
	}
	return fmt.Sprintf("preparation step '%s' failed: %v", e.Step, e.Cause)
}

// Unwrap returns the underlying error.
func (e *PreparationError) Unwrap() error {
	return e.Cause
}

// Preparer orchestrates the execution of preparation steps.
type Preparer struct {
	steps []PreparationStep
}

// NewPreparer creates a new Preparer with the given steps.
func NewPreparer(steps ...PreparationStep) *Preparer {
	return &Preparer{
		steps: steps,
	}
}

// NewPreparerFromScript creates a Preparer from a script and timeout.
// Returns nil if script is empty.
func NewPreparerFromScript(script string, timeoutSeconds int) *Preparer {
	if script == "" {
		return nil
	}

	timeout := time.Duration(timeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}

	step := NewScriptPreparationStep(script, timeout)
	return NewPreparer(step)
}

// Prepare executes all preparation steps in order.
// Stops and returns error on first failure.
func (p *Preparer) Prepare(ctx context.Context, prepCtx *PreparationContext) error {
	if len(p.steps) == 0 {
		return nil
	}

	log.Info("Starting workspace preparation",
		"pod_id", prepCtx.PodID, "step_count", len(p.steps))

	for i, step := range p.steps {
		log.Info("Executing preparation step",
			"pod_id", prepCtx.PodID, "step", step.Name(), "step_num", i+1, "total", len(p.steps))

		if err := step.Execute(ctx, prepCtx); err != nil {
			log.Error("Preparation step failed",
				"pod_id", prepCtx.PodID, "step", step.Name(), "error", err)
			return err
		}
	}

	log.Info("Workspace preparation completed", "pod_id", prepCtx.PodID)
	return nil
}

// AddStep adds a preparation step to the preparer.
func (p *Preparer) AddStep(step PreparationStep) {
	p.steps = append(p.steps, step)
}

// StepCount returns the number of steps in the preparer.
func (p *Preparer) StepCount() int {
	return len(p.steps)
}

// ScriptPreparationStep executes a shell script as a preparation step.
type ScriptPreparationStep struct {
	script  string
	timeout time.Duration
}

// NewScriptPreparationStep creates a new ScriptPreparationStep.
func NewScriptPreparationStep(script string, timeout time.Duration) *ScriptPreparationStep {
	if timeout <= 0 {
		timeout = 5 * time.Minute // Default timeout
	}
	return &ScriptPreparationStep{
		script:  script,
		timeout: timeout,
	}
}

// Name returns the step name.
func (s *ScriptPreparationStep) Name() string {
	return "script"
}

// Execute runs the script with the preparation context.
func (s *ScriptPreparationStep) Execute(ctx context.Context, prepCtx *PreparationContext) error {
	if s.script == "" {
		return nil
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	log.Info("Executing preparation script",
		"pod_id", prepCtx.PodID, "workspace_dir", prepCtx.WorkspaceDir, "timeout", s.timeout.String())

	// Create command using platform-appropriate shell
	shell, flag := envpath.ShellCommand()
	cmd := exec.CommandContext(ctx, shell, flag, s.script)
	cmd.Dir = prepCtx.WorkspaceDir
	cmd.Env = s.buildEnv(prepCtx)

	// Execute and capture output
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err != nil {
		log.Error("Preparation script failed",
			"pod_id", prepCtx.PodID, "error", err, "output", outputStr)

		return &PreparationError{
			Step:   s.Name(),
			Cause:  err,
			Output: outputStr,
		}
	}

	log.Info("Preparation script completed",
		"pod_id", prepCtx.PodID, "output_len", len(outputStr))

	if outputStr != "" {
		log.Debug("Script output", "output", outputStr)
	}

	return nil
}

// buildEnv builds the environment variables for script execution.
func (s *ScriptPreparationStep) buildEnv(prepCtx *PreparationContext) []string {
	// Start with filtered environment (removes sensitive/unnecessary vars)
	env := envfilter.FilterEnv(os.Environ())

	// Add extra paths for common tools
	env = s.addToolPaths(env)

	// Add preparation context variables
	for k, v := range prepCtx.GetEnvVars() {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	return env
}

// addToolPaths adds common tool paths to the environment.
func (s *ScriptPreparationStep) addToolPaths(env []string) []string {
	extraDirs := envpath.UserBinaryDirs()

	for i, e := range env {
		if strings.HasPrefix(e, "PATH=") {
			currentPath := strings.TrimPrefix(e, "PATH=")
			env[i] = "PATH=" + envpath.PrependToPath(currentPath, extraDirs...)
			return env
		}
	}

	// PATH not found in env — construct a minimal one
	env = append(env, "PATH="+envpath.PrependToPath(envpath.DefaultSystemPath(), extraDirs...))
	return env
}
