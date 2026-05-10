package tokenusage_test

// NOTE: subtests below mutate process-global HOME via t.Setenv (claude /
// opencode parsers read os.UserHomeDir). Do NOT call t.Parallel() in any
// case in this file — sibling tests would race. The same constraint
// applies to parser_agents_test.go in this package.

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	aiderfixture "github.com/anthropics/agentsmesh/runner/internal/agents/aider/testsupport"
	claudefixture "github.com/anthropics/agentsmesh/runner/internal/agents/claude/testsupport"
	codexfixture "github.com/anthropics/agentsmesh/runner/internal/agents/codex/testsupport"
	_ "github.com/anthropics/agentsmesh/runner/internal/agents/loopal"
	opencodefixture "github.com/anthropics/agentsmesh/runner/internal/agents/opencode/testsupport"
	"github.com/anthropics/agentsmesh/runner/internal/tokenusage"
)

// fixtureCase wires one canonical slug to its fixture builder + expected
// model attribution. Aliased slugs (e.g. "codex-cli", "claude-code") share
// the same underlying parser instance — the registry sentinel below maps
// them back via instance identity, so they don't need their own entries.
type fixtureCase struct {
	buildFixture   func(t *testing.T) string
	wantModelNames []string
}

var fixtureCases = map[string]fixtureCase{
	"codex":    {buildFixture: codexfixture.BuildFixtureSandbox, wantModelNames: []string{"o3-mini", "gpt-4.1"}},
	"claude":   {buildFixture: claudefixture.BuildFixtureSandbox, wantModelNames: []string{"claude-sonnet-4-20250514"}},
	"aider":    {buildFixture: aiderfixture.BuildFixtureSandbox, wantModelNames: []string{"aider-unknown"}},
	"opencode": {buildFixture: opencodefixture.BuildFixtureSandbox, wantModelNames: []string{"claude-sonnet-4-20250514"}},
}

// Each fixture case must drive its parser through to non-zero token counts
// attributed to specific models. Pinning model names guards against
// regressions where totals stay correct but model attribution silently
// breaks. Iteration is driven from the parser registry (not the cases map)
// so aliased slugs (codex / codex-cli; claude / claude-code) each get their
// own subtest — failures are localized to the offending alias.
func TestRegisteredParsers_HaveFixtureProducingNonZeroTokens(t *testing.T) {
	slugs := tokenusage.RegisteredParserSlugs()

	for _, slug := range slugs {
		if tokenusage.IsParserOptOut(slug) {
			continue
		}
		t.Run(slug, func(t *testing.T) {
			parser := tokenusage.GetParser(slug)
			require.NotNil(t, parser, "agent %q must register a parser (or call RegisterParserOptOut)", slug)

			tc := lookupFixtureCase(slug)
			require.NotNilf(t, tc, "agent slug %q has parser registered but no fixture case covers its parser type — add it to fixtureCases in parser_contract_test.go", slug)

			sandbox := tc.buildFixture(t)
			usage, err := parser.Parse(sandbox, time.Unix(0, 0))
			require.NoError(t, err, "parser %q errored on its own fixture", slug)
			require.NotNil(t, usage, "parser %q produced nil usage from its own fixture", slug)
			require.False(t, usage.IsEmpty(), "parser %q produced empty usage from its own fixture", slug)

			for _, want := range tc.wantModelNames {
				m := usage.Models[want]
				require.NotNilf(t, m, "parser %q fixture must produce model %q (got models: %v)", slug, want, modelKeys(usage))
				assert.Greaterf(t, m.InputTokens, int64(0), "%s/%s input tokens must be > 0", slug, want)
				assert.Greaterf(t, m.OutputTokens, int64(0), "%s/%s output tokens must be > 0", slug, want)
			}
		})
	}
}

// lookupFixtureCase resolves a registered slug to its fixture case via
// parser instance identity, so aliased slugs (e.g. "codex-cli" mapping to
// the same parser as "codex") share one fixtureCases entry.
func lookupFixtureCase(slug string) *fixtureCase {
	target := tokenusage.GetParser(slug)
	if target == nil {
		return nil
	}
	for caseSlug, tc := range fixtureCases {
		if tokenusage.GetParser(caseSlug) == target {
			tc := tc
			return &tc
		}
	}
	return nil
}

// Sentinel kept for clarity even though the main test now drives off the
// registry directly: failure here surfaces "registered parser, no fixture"
// independently of the per-slug subtest noise.
func TestRegistryCoverage_EveryNonOptOutParserHasFixture(t *testing.T) {
	for _, slug := range tokenusage.RegisteredParserSlugs() {
		if tokenusage.IsParserOptOut(slug) {
			continue
		}
		assert.NotNilf(t, lookupFixtureCase(slug),
			"agent slug %q has parser registered but no fixtureCases entry covers its parser type — "+
				"add %q (or one of its aliases) to fixtureCases in parser_contract_test.go", slug, slug)
	}
}

func modelKeys(u *tokenusage.TokenUsage) []string {
	keys := make([]string, 0, len(u.Models))
	for k := range u.Models {
		keys = append(keys, k)
	}
	return keys
}

// Opt-out marker is mutually exclusive with parser registration: an agent
// either ships a parser+fixture, or formally opts out — never both.
// Driven off OptOutSlugs() so adding a new opt-out auto-extends coverage.
func TestOptOutAgents_DoNotAlsoRegisterParser(t *testing.T) {
	slugs := tokenusage.OptOutSlugs()
	require.NotEmpty(t, slugs, "expected at least one opt-out registration (e.g. loopal)")
	for _, slug := range slugs {
		assert.Truef(t, tokenusage.IsParserOptOut(slug), "agent %q expected to be opt-out", slug)
		assert.Nilf(t, tokenusage.GetParser(slug), "opt-out agent %q must not register a parser", slug)
	}
}
