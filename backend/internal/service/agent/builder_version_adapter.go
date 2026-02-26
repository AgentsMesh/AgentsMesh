package agent

import (
	"strconv"
	"strings"
)

// ArgTransform defines a single CLI argument transformation rule.
// Transformation direction: NewFlag → OldFlag (downgrade from latest to legacy syntax).
// When args contain NewFlag (from DB template, latest syntax), it is replaced with OldFlag (legacy syntax).
type ArgTransform struct {
	// OldFlag is the legacy flag name used by older CLI versions (replacement target).
	OldFlag string
	// NewFlag is the current flag name used in DB template / latest CLI (matched source).
	NewFlag string
	// ValueMap maps current values to legacy values (optional).
	// If nil, the original value is kept unchanged.
	ValueMap map[string]string
}

// VersionArgRule defines arg transformations for a specific version range.
type VersionArgRule struct {
	// VersionBelow: apply this rule if agent version < this value.
	// Uses semantic version comparison.
	VersionBelow string
	// Transforms are the arg transformations to apply.
	Transforms []ArgTransform
}

// AdaptArgsForVersion applies version-specific arg transformations.
// If version is empty (Runner didn't report), returns args unchanged.
// Rules are evaluated in order; the first matching rule wins.
func AdaptArgsForVersion(args []string, version string, rules []VersionArgRule) []string {
	if version == "" || len(rules) == 0 {
		return args
	}

	// Find the first matching rule
	var matchedRule *VersionArgRule
	for i := range rules {
		if CompareVersions(version, rules[i].VersionBelow) < 0 {
			matchedRule = &rules[i]
			break
		}
	}

	if matchedRule == nil {
		return args // Version is >= all thresholds, no adaptation needed
	}

	// Apply transforms
	result := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		transformed := false
		for _, t := range matchedRule.Transforms {
			if args[i] == t.NewFlag {
				// Replace flag name
				result = append(result, t.OldFlag)
				// If next arg exists and is a value (not another flag), apply value mapping
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					i++
					if t.ValueMap != nil {
						if mapped, ok := t.ValueMap[args[i]]; ok {
							result = append(result, mapped)
						} else {
							result = append(result, args[i])
						}
					} else {
						result = append(result, args[i])
					}
				}
				transformed = true
				break
			}
		}
		if !transformed {
			result = append(result, args[i])
		}
	}
	return result
}

// CompareVersions compares two version strings.
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
// Supports formats: "1.2.3", "0.1.2025042500", "v1.2.3".
// Non-numeric segments are compared lexicographically.
func CompareVersions(a, b string) int {
	a = strings.TrimPrefix(a, "v")
	b = strings.TrimPrefix(b, "v")

	partsA := strings.Split(a, ".")
	partsB := strings.Split(b, ".")

	maxLen := len(partsA)
	if len(partsB) > maxLen {
		maxLen = len(partsB)
	}

	for i := 0; i < maxLen; i++ {
		var pa, pb string
		if i < len(partsA) {
			pa = partsA[i]
		}
		if i < len(partsB) {
			pb = partsB[i]
		}

		// Try numeric comparison first
		na, errA := strconv.ParseInt(pa, 10, 64)
		nb, errB := strconv.ParseInt(pb, 10, 64)

		if errA == nil && errB == nil {
			if na < nb {
				return -1
			}
			if na > nb {
				return 1
			}
		} else {
			// Fallback to lexicographic comparison
			if pa < pb {
				return -1
			}
			if pa > pb {
				return 1
			}
		}
	}

	return 0
}
