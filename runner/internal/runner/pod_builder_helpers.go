package runner

import "github.com/anthropics/agentsmesh/runner/internal/logger"

// mergeEnvVars merges environment variables: resolved PATH < config env < command env.
func (b *PodBuilder) mergeEnvVars(sandboxRoot string) map[string]string {
	result := make(map[string]string)

	// Inject resolved login shell PATH so PTY processes can find tools
	// even when runner runs as a systemd/launchd service with minimal PATH.
	if b.deps.Config != nil && b.deps.Config.ResolvedPATH != "" {
		result["PATH"] = b.deps.Config.ResolvedPATH
	}

	if b.deps.Config != nil {
		for k, v := range b.deps.Config.AgentEnvVars {
			result[k] = v
		}
	}

	if b.cmd != nil {
		for k, v := range b.cmd.EnvVars {
			result[k] = v
		}
	}

	return result
}

// sendProgress sends a pod initialization progress event (best-effort).
func (b *PodBuilder) sendProgress(phase string, progress int, message string) {
	if b.cmd == nil || b.cmd.PodKey == "" || b.deps.ProgressSender == nil {
		return
	}

	if err := b.deps.ProgressSender.SendPodInitProgress(b.cmd.PodKey, phase, int32(progress), message); err != nil {
		logger.Pod().Debug("Failed to send init progress", "pod_key", b.cmd.PodKey, "phase", phase, "error", err)
	}
}
