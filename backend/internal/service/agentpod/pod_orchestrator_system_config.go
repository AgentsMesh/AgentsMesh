package agentpod

// systemConfigKeySet is the canonical set of system-injected AgentFile CONFIG
// keys. newSystemOverrides (injected at resolve time) and isSystemConfigKey
// (filter when snapshotting source pod config on resume) must agree — this map
// is the single source of truth.
var systemConfigKeySet = map[string]struct{}{
	"session_id":     {},
	"resume_enabled": {},
	"resume_session": {},
}

func isSystemConfigKey(name string) bool {
	_, ok := systemConfigKeySet[name]
	return ok
}

// newSystemOverrides builds the system-injected CONFIG values for AgentFile
// resolve. Keys are guaranteed to be a subset of systemConfigKeySet.
func newSystemOverrides(sessionID string, isResumeMode, resumeAgentSession bool) map[string]interface{} {
	overrides := make(map[string]interface{}, len(systemConfigKeySet))
	if !isResumeMode {
		overrides["session_id"] = sessionID
		return overrides
	}
	overrides["session_id"] = ""
	if resumeAgentSession {
		overrides["resume_enabled"] = true
		overrides["resume_session"] = sessionID
	}
	return overrides
}
