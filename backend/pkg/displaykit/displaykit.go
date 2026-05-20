// Package displaykit guards display-name fields (channel.name, ticket.title,
// pod.alias, api_key.name) against unicode anomalies that bite downstream:
// zero-width chars, RTL/LTR overrides, ASCII control bytes, and unbounded
// whitespace abuse. These fields hold free-form Unicode (unlike slugkit
// identifiers), so the rules are deliberately lax — just remove stuff that
// no honest user types but attackers exploit.
package displaykit

import (
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	ErrEmpty      = errors.New("display: empty after sanitize")
	ErrTooLong    = errors.New("display: exceeds max length")
	ErrInvalidUTF = errors.New("display: invalid UTF-8")
)

// isDangerousRune strips:
//   - C0/C1 control chars (except \t, \n, \r which are whitespace and get
//     collapsed to a single space below)
//   - Unicode category Cf "Format": ZWSP, ZWJ, ZWNJ, LRM, RLM, LRE, RLE,
//     PDF, LRO, RLO, BOM/ZWNBSP, word joiners. None are typed by honest
//     users; attackers use them to spoof identifiers and hide content.
func isDangerousRune(r rune) bool {
	if r == '\t' || r == '\n' || r == '\r' {
		return false
	}
	if unicode.IsControl(r) {
		return true
	}
	if unicode.In(r, unicode.Cf) {
		return true
	}
	return false
}

// Sanitize strips dangerous runes, collapses runs of whitespace (incl.
// newlines and tabs), and trims outer whitespace. The fields this targets
// are single-line by convention.
func Sanitize(raw string) string {
	if !utf8.ValidString(raw) {
		raw = strings.ToValidUTF8(raw, "")
	}
	var b strings.Builder
	prevSpace := true
	for _, r := range raw {
		if isDangerousRune(r) {
			continue
		}
		if unicode.IsSpace(r) {
			if !prevSpace {
				b.WriteRune(' ')
				prevSpace = true
			}
			continue
		}
		b.WriteRune(r)
		prevSpace = false
	}
	return strings.TrimSpace(b.String())
}

// SanitizeAndValidate sanitizes raw, then enforces length bounds in runes
// (not bytes — Chinese/emoji should be counted as 1 each).
func SanitizeAndValidate(raw string, minLen, maxLen int) (string, error) {
	s := Sanitize(raw)
	if utf8.RuneCountInString(s) < minLen {
		return "", ErrEmpty
	}
	if utf8.RuneCountInString(s) > maxLen {
		return "", ErrTooLong
	}
	return s, nil
}
