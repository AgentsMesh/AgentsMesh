package runner

import "strings"

// TerminalInputAdapter adapts raw terminal input for a specific agent's TUI.
// Agents with full-screen TUIs (raw mode) may need input sanitization
// to prevent embedded newlines from triggering premature submission.
//
// Implement this interface and register via RegisterInputAdapter to add
// support for new agent types (OCP: open for extension, closed for modification).
type TerminalInputAdapter interface {
	// Adapt transforms raw terminal input bytes for the agent's TUI.
	// Return data unchanged if no adaptation is needed.
	Adapt(data []byte) []byte
}

// inputAdapterRegistry maps agent type slugs to their input adapters.
var inputAdapterRegistry = map[string]TerminalInputAdapter{}

// RegisterInputAdapter registers a TerminalInputAdapter for an agent type.
// Call this from init() functions in agent-specific files.
func RegisterInputAdapter(agentType string, adapter TerminalInputAdapter) {
	inputAdapterRegistry[agentType] = adapter
}

// adaptTerminalInput looks up the adapter for the given agent type and applies it.
// Returns data unchanged if no adapter is registered (default passthrough).
func adaptTerminalInput(data []byte, agentType string) []byte {
	if adapter, ok := inputAdapterRegistry[agentType]; ok {
		return adapter.Adapt(data)
	}
	return data
}

// --- Codex CLI adapter ---

func init() {
	adapter := &codexInputAdapter{}
	RegisterInputAdapter("codex", adapter)
	RegisterInputAdapter("codex-cli", adapter)
}

// codexInputAdapter sanitizes input for Codex CLI's ratatui TUI.
// Codex Rust uses a full-screen TUI in raw mode where:
//   - \n (0x0A) and \r (0x0D) are both interpreted as Enter
//   - Multi-line input gets split into multiple submissions
//
// This replaces embedded newlines with spaces to keep the prompt as a
// single submission, preserving the trailing \r (Enter) if present.
type codexInputAdapter struct{}

func (a *codexInputAdapter) Adapt(data []byte) []byte {
	if len(data) == 0 {
		return data
	}

	// Check if data ends with Enter (\r or \n)
	endsWithEnter := data[len(data)-1] == '\r' || data[len(data)-1] == '\n'

	// Replace all newlines with spaces in the body
	s := string(data)
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.TrimSpace(s)

	if s == "" {
		if endsWithEnter {
			return []byte("\r")
		}
		return data
	}

	if endsWithEnter {
		return append([]byte(s), '\r')
	}

	return []byte(s)
}
