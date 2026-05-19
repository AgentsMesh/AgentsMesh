package slugkit

import (
	"regexp"
	"strings"
)

var (
	nonAlnum     = regexp.MustCompile(`[^a-z0-9]+`)
	trailingDash = regexp.MustCompile(`-+$`)
)

func Sanitize(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	s = nonAlnum.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if len(s) > MaxLen {
		s = trailingDash.ReplaceAllString(s[:MaxLen], "")
	}
	return s
}

// SanitizeAndValidate sanitizes raw input and runs Validate.
// Returns the sanitized slug on success; empty string + error otherwise.
func SanitizeAndValidate(raw string) (string, error) {
	s := Sanitize(raw)
	if err := Validate(s); err != nil {
		return "", err
	}
	return s, nil
}
