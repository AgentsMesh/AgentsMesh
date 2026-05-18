package agentpod

import (
	agentDomain "github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	podDomain "github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
)

// mergeSourcePodConfigPrefs projects a terminated source pod's persisted CONFIG
// snapshot back into userPrefs for the AgentFile resolve pass on resume.
//
// Pre-condition (enforced by handleResumeMode + agentResolver): sourceAgent
// MUST be the Agent definition for sourcePod.AgentSlug. Passing a mismatched
// (sourcePod, sourceAgent) pair would silently corrupt the merge.
//
// Merge priority (later overrides earlier):
//  1. userPrefs (user's personal agent preferences)
//  2. legacy Model/PermissionMode columns from sourcePod (Claude-family only)
//  3. sourcePod.ResolvedConfig snapshot, minus system-injected keys
func mergeSourcePodConfigPrefs(userPrefs map[string]interface{}, sourcePod *podDomain.Pod, sourceAgent *agentDomain.Agent) map[string]interface{} {
	if sourcePod == nil {
		return userPrefs
	}

	legacy := legacyColumnPrefs(sourcePod, sourceAgent)
	snapshot := nonSystemConfigPrefs(sourcePod.ResolvedConfig)

	return agentDomain.MergeConfigs(userPrefs, legacy, snapshot)
}

// legacyColumnPrefs bridges legacy Claude-family columns (Pod.Model,
// Pod.PermissionMode) back into CONFIG so they survive the AgentFile resolve
// pass on resume. Without this, resuming a Claude pod created before CONFIG
// snapshot existed would silently lose the user's chosen model / permission
// mode. Returns nil for agents that don't use legacy columns.
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

// nonSystemConfigPrefs strips system-injected keys (see systemConfigKeySet)
// from a snapshot before re-feeding it into the next resolve pass. System keys
// are minted fresh per resolve (session_id, resume_*), so carrying them over
// would leak stale session / resume state into the new pod.
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
