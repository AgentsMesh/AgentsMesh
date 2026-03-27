package client

import (
	"context"
	"os/exec"
	"strings"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/envpath"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// probeAgents probes all agents for availability and version.
// This is a pure function (no lock held) — safe for concurrent use.
func probeAgents(agents []*runnerv1.AgentInfo) []agentProbeResult {
	results := make([]agentProbeResult, 0, len(agents))
	for _, agent := range agents {
		r := agentProbeResult{slug: agent.Slug}

		if agent.Command == "" {
			logger.GRPCTrace().Trace("Agent has no command defined, skipping", "agent", agent.Slug)
			results = append(results, r)
			continue
		}

		path, err := exec.LookPath(agent.Command)
		if err != nil {
			// Fallback: search common user binary directories.
			// This handles cases where the service runs with a minimal PATH
			// (e.g. launchd on macOS provides only /usr/bin:/bin:/usr/sbin:/sbin).
			path = envpath.LookPathFallback(agent.Command)
			if path == "" {
				logger.GRPCTrace().Trace("Agent command not found in PATH or fallback dirs",
					"agent", agent.Slug, "command", agent.Command)
				results = append(results, r)
				continue
			}
			logger.GRPC().Debug("Agent found via fallback path search",
				"agent", agent.Slug, "command", agent.Command, "path", path)
		}

		r.found = true
		r.path = path
		r.version = detectAgentVersion(path)
		results = append(results, r)
	}
	return results
}

// detectAgentVersion runs "<command> --version" and extracts the version string.
// Each attempt uses an independent timeout to avoid one slow attempt starving the next.
// Returns empty string if version detection fails (non-fatal).
func detectAgentVersion(command string) string {
	// Try --version first (most common)
	if version := tryVersionFlag(command, "--version"); version != "" {
		return version
	}
	// Fallback to -V (some agents use this)
	return tryVersionFlag(command, "-V")
}

// tryVersionFlag runs a single version detection attempt with its own timeout.
func tryVersionFlag(command, flag string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, command, flag)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}
	return parseVersionFromOutput(string(output))
}

// parseVersionFromOutput extracts a semver-like version from command output.
// Handles common formats:
//   - "Claude Code v1.2.3"
//   - "codex 1.2.3"
//   - "aider v0.50.1"
//   - "1.2.3"
func parseVersionFromOutput(output string) string {
	// Take only the first line
	if idx := strings.IndexAny(output, "\n\r"); idx != -1 {
		output = output[:idx]
	}

	// Look for version pattern: optional 'v' prefix followed by digits.digits.digits
	fields := strings.Fields(output)
	for _, field := range fields {
		cleaned := strings.TrimPrefix(field, "v")
		parts := strings.Split(cleaned, ".")
		if len(parts) >= 2 && len(parts) <= 4 {
			allNumeric := true
			for _, ch := range parts[0] {
				if ch < '0' || ch > '9' {
					allNumeric = false
					break
				}
			}
			if allNumeric && len(parts[0]) > 0 {
				return cleaned
			}
		}
	}

	// Fallback: return the raw first line (trimmed)
	return strings.TrimSpace(output)
}
