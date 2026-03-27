// Package autopilot implements the AutopilotController for supervised Pod automation.
package autopilot

import "time"

// Default configuration values for Autopilot components.
// These are centralized here for easy discovery and modification.
const (
	// DefaultMCPPort is the default port for MCP HTTP Server.
	DefaultMCPPort = 19000

	// DefaultIterationTimeout is the default timeout for a single iteration.
	DefaultIterationTimeout = 5 * time.Minute

	// DefaultMaxIterations is the default maximum number of iterations.
	DefaultMaxIterations = 10

	// DefaultMinTriggerGap is the minimum time between iteration triggers.
	DefaultMinTriggerGap = 5 * time.Second

	// DefaultMaxConsecutiveErrors is the max consecutive errors before giving up.
	DefaultMaxConsecutiveErrors = 3

	// DefaultAgent is the default control agent command.
	DefaultAgent = "claude"

	// DefaultStateCheckPeriod is the default period for state detection checks.
	DefaultStateCheckPeriod = 500 * time.Millisecond

	// DefaultResumeDelay is the delay before checking Pod state after resume.
	DefaultResumeDelay = 500 * time.Millisecond

	// MinRetryDelay is the minimum retry delay after error.
	MinRetryDelay = 2 * time.Second

	// MaxRetryDelay is the maximum retry delay (exponential backoff cap).
	MaxRetryDelay = 30 * time.Second
)

// SummaryMaxLength is the maximum length for summary strings in events.
const SummaryMaxLength = 1000

// LogOutputMaxLength is the maximum length for log output truncation.
const LogOutputMaxLength = 2000
