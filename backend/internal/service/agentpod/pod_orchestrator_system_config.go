package agentpod

var systemConfigKeySet = map[string]struct{}{
	"session_id":     {},
	"resume_enabled": {},
	"resume_session": {},
}

func isSystemConfigKey(name string) bool {
	_, ok := systemConfigKeySet[name]
	return ok
}

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
