package mockagent

import (
	"testing"
)

func TestFilterEnvByPrefix(t *testing.T) {
	env := []string{
		"E2E_TEST_FOO=1",
		"ANTHROPIC_API_KEY=secret",
		"CLAUDE_MODEL=sonnet",
		"PATH=/usr/bin",
		"HOME=/root",
		"E2E_TEST_BAR=2",
	}
	got := filterEnvByPrefix(env, envDumpPrefixes)

	wantSet := map[string]bool{
		"E2E_TEST_FOO=1":         true,
		"ANTHROPIC_API_KEY=secret": true,
		"CLAUDE_MODEL=sonnet":    true,
		"E2E_TEST_BAR=2":         true,
	}
	if len(got) != len(wantSet) {
		t.Fatalf("got %d entries, want %d: %v", len(got), len(wantSet), got)
	}
	for _, kv := range got {
		if !wantSet[kv] {
			t.Errorf("unexpected entry %q", kv)
		}
	}
}

func TestFilterEnvByPrefix_NoMatch(t *testing.T) {
	got := filterEnvByPrefix([]string{"PATH=/bin", "HOME=/x"}, envDumpPrefixes)
	if len(got) != 0 {
		t.Errorf("expected no matches, got %v", got)
	}
}

func TestFilterEnvByPrefix_EmptyInput(t *testing.T) {
	got := filterEnvByPrefix(nil, envDumpPrefixes)
	if len(got) != 0 {
		t.Errorf("expected nil/empty result, got %v", got)
	}
}
