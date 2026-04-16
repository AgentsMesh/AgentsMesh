package tokenusage

import "strings"

var parserRegistry = map[string]TokenParser{}

// RegisterParser registers a TokenParser for the given agent slugs.
// Typically called from init() in agent subpackages.
// Panics if any slug is already registered (detects accidental duplicates).
func RegisterParser(slugs []string, parser TokenParser) {
	for _, s := range slugs {
		if _, exists := parserRegistry[s]; exists {
			panic("tokenusage: duplicate parser registration: " + s)
		}
		parserRegistry[s] = parser
	}
}

// GetParser returns a TokenParser for the given agent.
// The agent is typically the LaunchCommand (e.g., "claude", "aider").
// Returns nil if no parser is registered for the agent.
func GetParser(agent string) TokenParser {
	// Normalize: lowercase, strip path prefix (e.g., "/usr/bin/claude" -> "claude")
	name := strings.ToLower(agent)
	if idx := strings.LastIndexAny(name, "/\\"); idx >= 0 {
		name = name[idx+1:]
	}
	return parserRegistry[name]
}
