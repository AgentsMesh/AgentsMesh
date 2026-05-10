// Package testsupport provides testing-only fixture helpers for the codex
// agent. It is intentionally a separate package so the production codex
// library does not link the testing package or embed fixture bytes.
package testsupport

import (
	_ "embed"
	"os"
	"path/filepath"
	"testing"
)

//go:embed testdata/openai_response.jsonl
var fixtureOpenAIResponse []byte

//go:embed testdata/anthropic_message.jsonl
var fixtureAnthropicMessage []byte

//go:embed testdata/mixed_with_noise.jsonl
var fixtureMixedNoise []byte

// Fixtures returns the embedded codex JSONL fixtures by basename so the
// codex package's table-driven test can iterate over them without depending
// on Bazel runfile layout.
func Fixtures() map[string][]byte {
	return map[string][]byte{
		"openai_response.jsonl":   fixtureOpenAIResponse,
		"anthropic_message.jsonl": fixtureAnthropicMessage,
		"mixed_with_noise.jsonl":  fixtureMixedNoise,
	}
}

// BuildFixtureSandbox plants the OpenAI-format fixture in a layout the codex
// parser will scan. Caller (tokenusage contract test) passes the resulting
// sandbox path into parser.Parse(...).
func BuildFixtureSandbox(t *testing.T) string {
	t.Helper()
	sandbox := t.TempDir()
	dir := filepath.Join(sandbox, "codex-home", "sessions", "2026", "04", "15")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("codex fixture: mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "rollout-fixture.jsonl"), fixtureOpenAIResponse, 0o644); err != nil {
		t.Fatalf("codex fixture: write: %v", err)
	}
	return sandbox
}
