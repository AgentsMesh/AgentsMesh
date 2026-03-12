package tokenusage

import "strings"

// parserRegistry maps agent command/slug to its parser.
var parserRegistry = map[string]TokenParser{
	"claude":     &ClaudeParser{},
	"claude-code": &ClaudeParser{},
	"codex":      &CodexParser{},
	"codex-cli":  &CodexParser{},
	"aider":      &AiderParser{},
	"opencode":   &OpenCodeParser{},
}

// GetParser returns a TokenParser for the given agent type.
// The agent type is typically the LaunchCommand (e.g., "claude", "aider").
// Returns nil if no parser is registered for the agent type.
func GetParser(agentType string) TokenParser {
	// Normalize: lowercase, strip path prefix (e.g., "/usr/bin/claude" -> "claude")
	name := strings.ToLower(agentType)
	if idx := strings.LastIndexAny(name, "/\\"); idx >= 0 {
		name = name[idx+1:]
	}
	return parserRegistry[name]
}
