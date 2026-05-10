package tokenusage

import (
	"sort"
	"strings"
)

var parserRegistry = map[string]TokenParser{}

// RegisterParser registers a TokenParser for the given agent slugs.
// Typically called from init() in agent subpackages.
// Slugs are normalized to lower case to match GetParser's lookup. Panics if
// a slug is already registered as a parser OR opted out (mutually exclusive).
func RegisterParser(slugs []string, parser TokenParser) {
	for _, s := range slugs {
		key := strings.ToLower(s)
		if _, exists := parserRegistry[key]; exists {
			panic("tokenusage: duplicate parser registration: " + key)
		}
		if _, exists := optOutSet[key]; exists {
			panic("tokenusage: " + key + " is opt-out; cannot register parser")
		}
		parserRegistry[key] = parser
	}
}

// GetParser returns a TokenParser for the given agent.
// The agent is typically the LaunchCommand (e.g., "claude", "aider").
// Returns nil if no parser is registered for the agent.
func GetParser(agent string) TokenParser {
	name := strings.ToLower(agent)
	if idx := strings.LastIndexAny(name, "/\\"); idx >= 0 {
		name = name[idx+1:]
	}
	return parserRegistry[name]
}

// RegisteredParserSlugs returns every slug currently in the parser
// registry, sorted for deterministic iteration. Exported so the
// cross-agent contract test can drive itself off the registry rather than
// a hard-coded list — adding a new agent without a fixture is then a CI
// failure, not a silent miss.
func RegisteredParserSlugs() []string {
	slugs := make([]string, 0, len(parserRegistry))
	for k := range parserRegistry {
		slugs = append(slugs, k)
	}
	sort.Strings(slugs)
	return slugs
}
