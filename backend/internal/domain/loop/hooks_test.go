package loop

import "testing"

func TestLoop_ValidateIdentifiers_RejectsInvalidSlug(t *testing.T) {
	l := &Loop{Slug: "Loop.Bad", AgentSlug: ""}
	if err := l.ValidateIdentifiers(); err == nil {
		t.Fatal("validator should reject slug with dot")
	}
}

func TestLoop_ValidateIdentifiers_RejectsInvalidAgentSlug(t *testing.T) {
	l := &Loop{Slug: "ok-loop", AgentSlug: "Agent.Bad"}
	if err := l.ValidateIdentifiers(); err == nil {
		t.Fatal("validator should reject agent_slug with dot")
	}
}

func TestLoop_ValidateIdentifiers_AcceptsValidSlugs(t *testing.T) {
	l := &Loop{Slug: "my-loop", AgentSlug: "claude-code"}
	if err := l.ValidateIdentifiers(); err != nil {
		t.Errorf("validator rejected valid slugs: %v", err)
	}
}

func TestLoop_ValidateIdentifiers_EmptyAgentSlugAllowed(t *testing.T) {
	l := &Loop{Slug: "my-loop", AgentSlug: ""}
	if err := l.ValidateIdentifiers(); err != nil {
		t.Errorf("empty agent_slug (optional reference) should pass: %v", err)
	}
}
