package client

import (
	"context"
	"os/exec"
	"strings"
	"sync"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/envpath"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// AgentProbeEntry represents cached probe result for a single agent.
type AgentProbeEntry struct {
	Slug    string
	Version string
	Path    string
}

// agentProbeResult holds the result of probing a single agent (used for lock-free collection).
type agentProbeResult struct {
	slug    string
	version string
	path    string
	found   bool // false if command not in PATH
}

// AgentProbe encapsulates agent detection logic and maintains a session-level cache.
// It is used during initialization (full probe) and before each heartbeat (diff probe).
//
// Thread-safe: all public methods are safe for concurrent use.
// Subprocess execution happens outside the lock to avoid blocking readers.
type AgentProbe struct {
	mu         sync.RWMutex
	agentTypes []*runnerv1.AgentTypeInfo  // known agent types from server
	cache      map[string]*AgentProbeEntry // slug -> last known state
}

// NewAgentProbe creates a new AgentProbe instance.
func NewAgentProbe() *AgentProbe {
	return &AgentProbe{
		cache: make(map[string]*AgentProbeEntry),
	}
}

// ProbeAll performs a full probe of all agent types and populates the cache.
// Called during initialization after receiving AgentTypeInfo from server.
// Returns (available agent slugs, full version info list).
func (p *AgentProbe) ProbeAll(agentTypes []*runnerv1.AgentTypeInfo) ([]string, []*runnerv1.AgentVersionInfo) {
	// Phase 1: Probe all agents without holding the lock (subprocess execution)
	results := probeAgentTypes(agentTypes)

	// Phase 2: Update cache under lock
	p.mu.Lock()
	defer p.mu.Unlock()

	p.agentTypes = agentTypes
	p.cache = make(map[string]*AgentProbeEntry)

	var available []string
	var versions []*runnerv1.AgentVersionInfo

	for _, r := range results {
		if !r.found {
			continue
		}

		p.cache[r.slug] = &AgentProbeEntry{
			Slug:    r.slug,
			Version: r.version,
			Path:    r.path,
		}

		available = append(available, r.slug)
		versions = append(versions, &runnerv1.AgentVersionInfo{
			Slug:    r.slug,
			Version: r.version,
			Path:    r.path,
		})

		logger.GRPC().Info("Agent detected", "agent", r.slug, "version", r.version, "path", r.path)
	}

	return available, versions
}

// ProbeAndDiff re-probes all known agent types and returns only the changes.
// Returns nil if no changes detected (saves heartbeat bandwidth).
// Called before each heartbeat.
//
// Subprocess execution happens outside the lock to avoid blocking readers
// during version detection (which may take seconds per agent).
func (p *AgentProbe) ProbeAndDiff() []*runnerv1.AgentVersionInfo {
	// Phase 1: Read agent types under read lock
	p.mu.RLock()
	agentTypes := p.agentTypes
	p.mu.RUnlock()

	if len(agentTypes) == 0 {
		return nil
	}

	// Phase 2: Probe all agents without holding any lock (subprocess execution)
	results := probeAgentTypes(agentTypes)

	// Phase 3: Compare with cache and update under write lock
	p.mu.Lock()
	defer p.mu.Unlock()

	var changes []*runnerv1.AgentVersionInfo

	// Check for new/changed agents
	probed := make(map[string]bool)
	for _, r := range results {
		probed[r.slug] = true

		if !r.found {
			// Agent no longer available — report removal if it was cached
			if _, existed := p.cache[r.slug]; existed {
				delete(p.cache, r.slug)
				changes = append(changes, &runnerv1.AgentVersionInfo{
					Slug: r.slug, Version: "", Path: "",
				})
				logger.GRPC().Info("Agent no longer available", "agent", r.slug)
			}
			continue
		}

		prev, existed := p.cache[r.slug]
		if !existed || prev.Version != r.version || prev.Path != r.path {
			p.cache[r.slug] = &AgentProbeEntry{
				Slug: r.slug, Version: r.version, Path: r.path,
			}
			changes = append(changes, &runnerv1.AgentVersionInfo{
				Slug: r.slug, Version: r.version, Path: r.path,
			})
			if existed {
				logger.GRPC().Info("Agent version changed",
					"agent", r.slug,
					"old_version", prev.Version, "new_version", r.version,
					"old_path", prev.Path, "new_path", r.path)
			} else {
				logger.GRPC().Info("New agent detected", "agent", r.slug, "version", r.version, "path", r.path)
			}
		}
	}

	// Check for agents that disappeared (in cache but not probed)
	// This handles the case where agentTypes changed between sessions
	for slug := range p.cache {
		if !probed[slug] {
			delete(p.cache, slug)
			changes = append(changes, &runnerv1.AgentVersionInfo{
				Slug: slug, Version: "", Path: "",
			})
			logger.GRPC().Info("Agent no longer available", "agent", slug)
		}
	}

	return changes
}

// GetAvailableAgents returns the current list of available agent slugs.
func (p *AgentProbe) GetAvailableAgents() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var slugs []string
	for slug := range p.cache {
		slugs = append(slugs, slug)
	}
	return slugs
}

// GetAgentVersions returns the full version info for all cached agents.
func (p *AgentProbe) GetAgentVersions() []*runnerv1.AgentVersionInfo {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var versions []*runnerv1.AgentVersionInfo
	for _, entry := range p.cache {
		versions = append(versions, &runnerv1.AgentVersionInfo{
			Slug:    entry.Slug,
			Version: entry.Version,
			Path:    entry.Path,
		})
	}
	return versions
}

// probeAgentTypes probes all agent types for availability and version.
// This is a pure function (no lock held) — safe for concurrent use.
func probeAgentTypes(agentTypes []*runnerv1.AgentTypeInfo) []agentProbeResult {
	results := make([]agentProbeResult, 0, len(agentTypes))
	for _, agent := range agentTypes {
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
			// Check first part is numeric
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
