package tokenusage

import (
	"sort"
	"strings"
)

// optOutSet records agent slugs that have token-usage semantics but cannot
// produce token counts via the on-disk parser path (e.g. ephemeral / in-memory
// agents like loopal). The cross-agent contract test in
// parser_contract_test.go skips these slugs instead of demanding a fixture.
var optOutSet = map[string]struct{}{}

// RegisterParserOptOut marks one or more agent slugs as intentionally lacking
// a fixture-driven parser. Use only when the agent has no on-disk session
// file format to parse — never as an escape hatch to skip writing tests.
// Panics if a slug is already opted out OR has a parser registered (mutually
// exclusive with RegisterParser).
func RegisterParserOptOut(slugs []string) {
	for _, s := range slugs {
		key := strings.ToLower(s)
		if _, exists := optOutSet[key]; exists {
			panic("tokenusage: duplicate opt-out registration: " + key)
		}
		if _, exists := parserRegistry[key]; exists {
			panic("tokenusage: " + key + " has a parser; cannot mark opt-out")
		}
		optOutSet[key] = struct{}{}
	}
}

// IsParserOptOut reports whether a slug has been marked opt-out.
func IsParserOptOut(slug string) bool {
	_, ok := optOutSet[strings.ToLower(slug)]
	return ok
}

// OptOutSlugs returns every slug currently marked opt-out, sorted for
// deterministic iteration. Exported so the contract test can assert
// invariants over all opt-outs without hardcoding the list.
func OptOutSlugs() []string {
	slugs := make([]string, 0, len(optOutSet))
	for k := range optOutSet {
		slugs = append(slugs, k)
	}
	sort.Strings(slugs)
	return slugs
}
