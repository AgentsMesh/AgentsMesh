package runner

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// buildACPPod creates a pod configured for ACP (Agent Communication Protocol) interaction.
// The ACPClient is NOT created here — it will be created by wireAndStartACPPod()
// in the MessageHandler, which wires Relay-forwarding event callbacks.
func (b *PodBuilder) buildACPPod(_ context.Context, sandboxRoot, workingDir, branchName string, resolvedArgs []string, envVars map[string]string, launchCommand string) (*Pod, error) {
	b.sendProgress("starting_acp", 80, "Preparing ACP agent...")

	// Build environment: start with current process env, then overlay envVars.
	// Use a map to deduplicate keys (envVars takes priority).
	envMap := make(map[string]string, len(envVars))
	for _, e := range os.Environ() {
		if k, _, ok := strings.Cut(e, "="); ok {
			envMap[k] = e
		}
	}
	for k, v := range envVars {
		envMap[k] = fmt.Sprintf("%s=%s", k, v)
	}
	envSlice := make([]string, 0, len(envMap))
	for _, v := range envMap {
		envSlice = append(envSlice, v)
	}

	pod := &Pod{
		ID:              b.cmd.PodKey,
		PodKey:          b.cmd.PodKey,
		Agent:           launchCommand,
		InteractionMode: InteractionModeACP,
		Branch:          branchName,
		SandboxPath:     sandboxRoot,
		LaunchCommand:   launchCommand,
		LaunchArgs:      resolvedArgs,
		WorkDir:         workingDir,
		LaunchEnv:       envSlice,
		StartedAt:       time.Now(),
		Status:          PodStatusInitializing,
		// ACPClient, IO are set by wireAndStartACPPod()
		// PTY fields (Terminal, VirtualTerminal, Aggregator, PTYLogger) left nil
	}

	logger.Pod().Info("Pod built (ACP)", "pod_key", b.cmd.PodKey, "working_dir", workingDir)
	b.sendProgress("acp_ready", 100, "ACP agent ready")

	return pod, nil
}
