package agentpod

import (
	agentDomain "github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	podDomain "github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
)

// Precondition: sourceAgent MUST be the Agent for sourcePod.AgentSlug (handleResumeMode
// + agentResolver enforce); mismatched pair would silently corrupt the merge.
// Priority (later wins): userPrefs → legacy Model/PermissionMode cols → ResolvedConfig snapshot (minus system keys).
func mergeSourcePodConfigPrefs(userPrefs map[string]interface{}, sourcePod *podDomain.Pod, sourceAgent *agentDomain.Agent) map[string]interface{} {
	if sourcePod == nil {
		return userPrefs
	}

	legacy := legacyColumnPrefs(sourcePod, sourceAgent)
	snapshot := nonSystemConfigPrefs(sourcePod.ResolvedConfig)

	return agentDomain.MergeConfigs(userPrefs, legacy, snapshot)
}

func legacyColumnPrefs(sourcePod *podDomain.Pod, sourceAgent *agentDomain.Agent) map[string]interface{} {
	if sourceAgent == nil || !sourceAgent.UsesLegacyColumns {
		return nil
	}
	prefs := make(map[string]interface{}, 2)
	if sourcePod.Model != nil && *sourcePod.Model != "" {
		prefs[agentDomain.ConfigKeyModel] = *sourcePod.Model
	}
	if sourcePod.PermissionMode != nil && *sourcePod.PermissionMode != "" {
		prefs[agentDomain.ConfigKeyPermissionMode] = *sourcePod.PermissionMode
	}
	return prefs
}

func nonSystemConfigPrefs(snapshot agentDomain.ConfigValues) map[string]interface{} {
	if len(snapshot) == 0 {
		return nil
	}
	out := make(map[string]interface{}, len(snapshot))
	for k, v := range snapshot {
		if isSystemConfigKey(k) {
			continue
		}
		out[k] = v
	}
	return out
}
