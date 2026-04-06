// Package terminal provides terminal state detection for AI agents.
package detector

// AgentState represents the detected state of an AI agent running in the terminal.
type AgentState string

const (
	// StateUnknown indicates the agent state is unknown (initial state before detection).
	StateUnknown AgentState = "unknown"
	// StateNotRunning indicates the agent is not running (not in alt screen).
	StateNotRunning AgentState = "not_running"
	// StateExecuting indicates the agent is actively executing (screen changing).
	StateExecuting AgentState = "executing"
	// StateWaiting indicates the agent is waiting for user input.
	StateWaiting AgentState = "waiting"
)
