package client

import (
	"sync"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
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
	mu     sync.RWMutex
	agents []*runnerv1.AgentInfo       // known agents from server
	cache  map[string]*AgentProbeEntry // slug -> last known state
}

// NewAgentProbe creates a new AgentProbe instance.
func NewAgentProbe() *AgentProbe {
	return &AgentProbe{
		cache: make(map[string]*AgentProbeEntry),
	}
}

// ProbeAll performs a full probe of all agents and populates the cache.
// Called during initialization after receiving AgentInfo from server.
// Returns (available agent slugs, full version info list).
func (p *AgentProbe) ProbeAll(agents []*runnerv1.AgentInfo) ([]string, []*runnerv1.AgentVersionInfo) {
	// Phase 1: Probe all agents without holding the lock (subprocess execution)
	results := probeAgents(agents)

	// Phase 2: Update cache under lock
	p.mu.Lock()
	defer p.mu.Unlock()

	p.agents = agents
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

// ProbeAndDiff re-probes all known agents and returns only the changes.
// Returns nil if no changes detected (saves heartbeat bandwidth).
// Called before each heartbeat.
//
// Subprocess execution happens outside the lock to avoid blocking readers
// during version detection (which may take seconds per agent).
func (p *AgentProbe) ProbeAndDiff() []*runnerv1.AgentVersionInfo {
	// Phase 1: Read agents under read lock
	p.mu.RLock()
	agents := p.agents
	p.mu.RUnlock()

	if len(agents) == 0 {
		return nil
	}

	// Phase 2: Probe all agents without holding any lock (subprocess execution)
	results := probeAgents(agents)

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
	// This handles the case where agents changed between sessions
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

// Detection helpers (probeAgents, detectAgentVersion, etc.) are in agent_probe_detect.go
