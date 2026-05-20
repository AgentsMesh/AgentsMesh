package agentpod

import "testing"

func TestPod_ValidateIdentifiers_RejectsInvalidPodKey(t *testing.T) {
	p := &Pod{PodKey: "bad.key"}
	if err := p.ValidateIdentifiers(); err == nil {
		t.Fatal("validator should reject pod_key with dot")
	}
}

func TestPod_ValidateIdentifiers_AcceptsValidPodKey(t *testing.T) {
	p := &Pod{PodKey: "1-standalone-abc12345", AgentSlug: "claude-code"}
	if err := p.ValidateIdentifiers(); err != nil {
		t.Errorf("validator rejected valid pod_key: %v", err)
	}
}

func TestPod_ValidateIdentifiers_RejectsInvalidAgentSlug(t *testing.T) {
	p := &Pod{PodKey: "1-standalone-abc12345", AgentSlug: "Agent.Bad"}
	if err := p.ValidateIdentifiers(); err == nil {
		t.Fatal("validator should reject agent_slug with dot")
	}
}
