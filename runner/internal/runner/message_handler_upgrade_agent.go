package runner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/envpath"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

const (
	agentUpgradeTimeout = 5 * time.Minute

	upgradeAgentStatusSuccess = "success"
	upgradeAgentStatusFailed  = "failed"
)

// OnUpgradeAgent upgrades a single installed agent by running its package-manager
// command (the backend reads it from the agents-table SSOT and ships the argv).
// Unlike a runner self-upgrade this never restarts the runner or touches running
// pods — the global binary is replaced in place and picked up by future pods.
// The argv is exec'd directly without a shell to avoid injection; the agent
// version is re-probed and reported so the UI refreshes via the next heartbeat.
func (h *RunnerMessageHandler) OnUpgradeAgent(cmd *runnerv1.UpgradeAgentCommand) error {
	log := logger.Runner()
	log.Info("Received upgrade_agent", "request_id", cmd.RequestId, "agent_slug", cmd.AgentSlug)

	if len(cmd.UpgradeArgv) == 0 {
		h.sendUpgradeAgentResult(cmd, false, "", "no upgrade command provided")
		return nil
	}

	// Serialize upgrades on this runner: concurrent `npm i -g` / `pip install`
	// race on the package manager's global cache lock. Different agents queue
	// rather than corrupt each other. The timeout starts only once the lock is
	// held, so a queued upgrade isn't charged for the wait.
	h.agentUpgradeMu.Lock()
	defer h.agentUpgradeMu.Unlock()

	resolved := resolveUpgradeTool(cmd.UpgradeArgv[0])
	if resolved == "" {
		h.sendUpgradeAgentResult(cmd, false, "", fmt.Sprintf("upgrade tool %q not found in PATH", cmd.UpgradeArgv[0]))
		return nil
	}

	ctx, cancel := context.WithTimeout(h.runner.GetRunContext(), agentUpgradeTimeout)
	defer cancel()

	c := exec.CommandContext(ctx, resolved, cmd.UpgradeArgv[1:]...)
	// Enhance PATH so a launchd/systemd minimal env can still resolve npm/pip/node.
	c.Env = append(os.Environ(), "PATH="+envpath.PrependToPath(os.Getenv("PATH"), envpath.UserBinaryDirs()...))
	out, runErr := c.CombinedOutput()
	if runErr != nil {
		h.sendUpgradeAgentResult(cmd, false, "", fmt.Sprintf("%v: %s", runErr, tailUpgradeOutput(out, 500)))
		return nil
	}

	newVersion := probeUpgradedVersion(cmd.Executable)
	log.Info("Agent upgraded", "agent_slug", cmd.AgentSlug, "new_version", newVersion)
	h.sendUpgradeAgentResult(cmd, true, newVersion, fmt.Sprintf("upgraded %s", cmd.AgentSlug))
	return nil
}

// resolveUpgradeTool resolves a binary to an absolute, executable path:
// PATH → user-bin fallback → Windows extension validation (.cmd/.exe). The
// ValidateExecutable step mirrors probeAgents (agent_probe_detect.go) so the
// upgrade path resolves binaries identically to version probing — without it,
// Windows would exec an extensionless npm shim. Empty = not found.
func resolveUpgradeTool(bin string) string {
	path, err := envpath.SafeLookPath(bin)
	if err != nil || path == "" {
		path = envpath.LookPathFallback(bin)
		if path == "" {
			return ""
		}
	}
	return envpath.ValidateExecutable(path)
}

// probeUpgradedVersion re-resolves the agent executable and reads its version
// after an upgrade. Empty when the binary/version can't be read (non-fatal —
// the next heartbeat's probe reconciles).
func probeUpgradedVersion(executable string) string {
	if executable == "" {
		return ""
	}
	if p := resolveUpgradeTool(executable); p != "" {
		return client.ProbeAgentVersion(p)
	}
	return ""
}

func (h *RunnerMessageHandler) sendUpgradeAgentResult(cmd *runnerv1.UpgradeAgentCommand, success bool, newVersion, detail string) {
	event := &runnerv1.UpgradeAgentResultEvent{
		RequestId:  cmd.RequestId,
		AgentSlug:  cmd.AgentSlug,
		NewVersion: newVersion,
	}
	if success {
		event.Status = upgradeAgentStatusSuccess
		event.Message = detail
	} else {
		event.Status = upgradeAgentStatusFailed
		event.Error = detail
	}
	if err := h.conn.SendUpgradeAgentResult(event); err != nil {
		logger.Runner().Error("Failed to send upgrade_agent result",
			"request_id", cmd.RequestId, "agent_slug", cmd.AgentSlug, "error", err)
	}
}

// tailUpgradeOutput returns the last n bytes of command output for error context.
func tailUpgradeOutput(b []byte, n int) string {
	if len(b) > n {
		b = b[len(b)-n:]
	}
	return strings.TrimSpace(string(b))
}
