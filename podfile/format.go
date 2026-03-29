package podfile

import (
	"fmt"
	"strings"
)

// FormatStringLiteral escapes and quotes a string for PodFile syntax.
// Handles backslashes, double quotes, newlines, and tabs.
func FormatStringLiteral(s string) string {
	escaped := strings.ReplaceAll(s, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)
	escaped = strings.ReplaceAll(escaped, "\n", `\n`)
	escaped = strings.ReplaceAll(escaped, "\t", `\t`)
	return fmt.Sprintf(`"%s"`, escaped)
}

// FormatValue formats any Go value for PodFile CONFIG syntax.
// Strings are escaped+quoted, bools are true/false, numbers are formatted.
func FormatValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		return FormatStringLiteral(val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%g", val)
	default:
		return FormatStringLiteral(fmt.Sprintf("%v", val))
	}
}
