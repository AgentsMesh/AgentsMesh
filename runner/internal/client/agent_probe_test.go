package client

import (
	"testing"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

func TestParseVersionFromOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"claude code format", "Claude Code v1.2.3", "1.2.3"},
		{"codex format", "codex 1.2.3", "1.2.3"},
		{"aider format", "aider v0.50.1", "0.50.1"},
		{"bare version", "1.2.3", "1.2.3"},
		{"v prefix", "v1.2.3", "1.2.3"},
		{"calver style", "0.1.2025042500", "0.1.2025042500"},
		{"four part version", "1.2.3.4", "1.2.3.4"},
		{"two part version", "1.2", "1.2"},
		{"multiline output", "tool v2.0.1\nsome other info\n", "2.0.1"},
		{"with trailing newline", "1.2.3\n", "1.2.3"},
		{"with carriage return", "1.2.3\r\n", "1.2.3"},
		{"empty input", "", ""},
		{"no version found", "some random text", "some random text"},
		{"version in sentence", "Version: v3.1.4 (stable)", "3.1.4"},
		{"gemini format", "Gemini CLI v1.0.0", "1.0.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseVersionFromOutput(tt.input)
			if result != tt.expected {
				t.Errorf("parseVersionFromOutput(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestAgentProbeProbeAll(t *testing.T) {
	probe := NewAgentProbe()

	// Probe with an agent that definitely exists (go) and one that doesn't
	agents := []*runnerv1.AgentInfo{
		{Slug: "test-go", Command: "go", Name: "Go"},
		{Slug: "nonexistent", Command: "this-command-does-not-exist-xyz", Name: "Nonexistent"},
		{Slug: "no-command", Command: "", Name: "No Command"},
	}

	available, versions := probe.ProbeAll(agents)

	// "go" should be detected, others should not
	if len(available) != 1 || available[0] != "test-go" {
		t.Errorf("expected [test-go], got %v", available)
	}
	if len(versions) != 1 || versions[0].Slug != "test-go" {
		t.Errorf("expected 1 version for test-go, got %d", len(versions))
	}
	if versions[0].Path == "" {
		t.Error("expected non-empty path for go")
	}

	// Verify cache
	cachedAgents := probe.GetAvailableAgents()
	if len(cachedAgents) != 1 {
		t.Errorf("expected 1 cached agent, got %d", len(cachedAgents))
	}
}

func TestAgentProbeProbeAndDiff_NoChanges(t *testing.T) {
	probe := NewAgentProbe()

	// Initial probe
	agents := []*runnerv1.AgentInfo{
		{Slug: "test-go", Command: "go", Name: "Go"},
	}
	probe.ProbeAll(agents)

	// Second probe should return nil (no changes)
	changes := probe.ProbeAndDiff()
	if changes != nil {
		t.Errorf("expected nil changes on second probe, got %v", changes)
	}
}

func TestAgentProbeProbeAndDiff_AgentRemoved(t *testing.T) {
	probe := NewAgentProbe()

	// Initial probe with "go" agent
	agents := []*runnerv1.AgentInfo{
		{Slug: "test-go", Command: "go", Name: "Go"},
	}
	probe.ProbeAll(agents)

	// Now change agents to have a nonexistent command for the same slug
	probe.mu.Lock()
	probe.agents = []*runnerv1.AgentInfo{
		{Slug: "test-go", Command: "this-command-does-not-exist-xyz", Name: "Go"},
	}
	probe.mu.Unlock()

	// Probe should detect removal
	changes := probe.ProbeAndDiff()
	if len(changes) != 1 {
		t.Fatalf("expected 1 change (removal), got %d", len(changes))
	}
	if changes[0].Slug != "test-go" || changes[0].Version != "" {
		t.Errorf("expected removal entry for test-go, got %+v", changes[0])
	}

	// Cache should be empty
	if agents := probe.GetAvailableAgents(); len(agents) != 0 {
		t.Errorf("expected 0 cached agents after removal, got %d", len(agents))
	}
}

func TestAgentProbeProbeAndDiff_NewAgent(t *testing.T) {
	probe := NewAgentProbe()

	// Initial probe with nonexistent agent
	agents := []*runnerv1.AgentInfo{
		{Slug: "nonexistent", Command: "this-command-does-not-exist-xyz", Name: "Nonexistent"},
	}
	probe.ProbeAll(agents)

	// Now add "go" to the agents
	probe.mu.Lock()
	probe.agents = []*runnerv1.AgentInfo{
		{Slug: "test-go", Command: "go", Name: "Go"},
	}
	probe.mu.Unlock()

	// Probe should detect new agent
	changes := probe.ProbeAndDiff()
	if len(changes) != 1 {
		t.Fatalf("expected 1 change (new agent), got %d", len(changes))
	}
	if changes[0].Slug != "test-go" {
		t.Errorf("expected new agent entry for test-go, got %+v", changes[0])
	}
}

func TestAgentProbeEmptyAgents(t *testing.T) {
	probe := NewAgentProbe()

	// ProbeAll with empty list
	available, versions := probe.ProbeAll(nil)
	if len(available) != 0 {
		t.Errorf("expected 0 available, got %d", len(available))
	}
	if len(versions) != 0 {
		t.Errorf("expected 0 versions, got %d", len(versions))
	}

	// ProbeAndDiff before any ProbeAll should return nil
	changes := probe.ProbeAndDiff()
	if changes != nil {
		t.Errorf("expected nil changes, got %v", changes)
	}
}
