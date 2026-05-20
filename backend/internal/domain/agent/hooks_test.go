package agent

import "testing"

func TestAgent_ValidateIdentifiers_RejectsInvalidSlug(t *testing.T) {
	a := &Agent{Slug: "Bad.Slug"}
	if err := a.ValidateIdentifiers(); err == nil {
		t.Fatal("validator should reject slug with dot")
	}
}

func TestAgent_ValidateIdentifiers_AcceptsValidSlug(t *testing.T) {
	a := &Agent{Slug: "claude-code"}
	if err := a.ValidateIdentifiers(); err != nil {
		t.Errorf("validator rejected valid slug: %v", err)
	}
}
