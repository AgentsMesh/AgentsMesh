package codex

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/agents/codex/testsupport"
	"github.com/anthropics/agentsmesh/runner/internal/tokenusage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type expectedModel struct {
	Input         int64
	Output        int64
	CacheCreation int64
	CacheRead     int64
}

// Codex emits multiple JSONL shapes — pinning each shape to a fixture file
// makes any future refactor that drops a branch fail loudly with a numeric
// mismatch instead of silently regressing token tracking.
//
// Fixtures live in the testsupport sub-package so the production codex
// library does not embed test data; this test materializes them via a
// temp file before invoking parseCodexJSONLFile.
func TestCodexParser_AllKnownFormats(t *testing.T) {
	fixtures := testsupport.Fixtures()

	cases := []struct {
		fixture string
		want    map[string]expectedModel
	}{
		{
			fixture: "openai_response.jsonl",
			want: map[string]expectedModel{
				"o3-mini": {Input: 1230, Output: 507},
				"gpt-4.1": {Input: 3740, Output: 1980},
			},
		},
		{
			fixture: "anthropic_message.jsonl",
			want: map[string]expectedModel{
				"claude-3-5-sonnet": {Input: 800, Output: 390, CacheCreation: 50, CacheRead: 550},
			},
		},
		{
			fixture: "mixed_with_noise.jsonl",
			want: map[string]expectedModel{
				"o3-mini": {Input: 500, Output: 250},
				"gpt-4o":  {Input: 175, Output: 75},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.fixture, func(t *testing.T) {
			data, ok := fixtures[tc.fixture]
			require.Truef(t, ok && !bytes.Equal(data, nil), "fixture %s missing from testsupport.Fixtures()", tc.fixture)

			path := filepath.Join(t.TempDir(), tc.fixture)
			require.NoError(t, os.WriteFile(path, data, 0o644))

			usage := tokenusage.NewTokenUsage()
			require.NoError(t, parseCodexJSONLFile(path, usage), "fixture %s must parse cleanly", tc.fixture)

			assert.Len(t, usage.Models, len(tc.want), "model count mismatch for %s", tc.fixture)

			for model, want := range tc.want {
				m := usage.Models[model]
				require.NotNilf(t, m, "fixture %s missing model %s", tc.fixture, model)
				assert.Equalf(t, want.Input, m.InputTokens, "%s/%s input", tc.fixture, model)
				assert.Equalf(t, want.Output, m.OutputTokens, "%s/%s output", tc.fixture, model)
				assert.Equalf(t, want.CacheCreation, m.CacheCreationTokens, "%s/%s cache_creation", tc.fixture, model)
				assert.Equalf(t, want.CacheRead, m.CacheReadTokens, "%s/%s cache_read", tc.fixture, model)
			}
		})
	}
}
