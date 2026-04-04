package agentfile

import (
	"fmt"
	"strings"
)

// AgentFile String Literal Specification
//
// This is the CANONICAL reference for AgentFile string escaping rules.
// All AgentFile producers MUST follow these rules:
//
//   STRING  = '"' CHAR* '"'
//   CHAR    = ESCAPE | REGULAR
//   ESCAPE  = '\' ( '\' | '"' | 'n' | 't' )
//   REGULAR = <any Unicode char except '"', '\', newline, tab>
//
// Escape order (backslash MUST be first to prevent double-escaping):
//   1. '\' → '\\'
//   2. '"' → '\"'
//   3. newline (0x0A) → '\n'
//   4. tab (0x09) → '\t'
//
// Implementations:
//   - Go escape: agentfile/format.go FormatStringLiteral (this file)
//   - Go unescape: agentfile/lexer/lexer_readers.go readString
//   - Go serialize: agentfile/serialize/serialize_expr.go quoteString
//   - TypeScript: web/src/lib/agentfile-layer.ts escapeAgentfileString
//   - Loop builder: backend/internal/service/loop/loop_orchestrator_start.go

// FormatStringLiteral escapes and quotes a string for AgentFile syntax.
// See AgentFile String Literal Specification above.
func FormatStringLiteral(s string) string {
	escaped := strings.ReplaceAll(s, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)
	escaped = strings.ReplaceAll(escaped, "\n", `\n`)
	escaped = strings.ReplaceAll(escaped, "\t", `\t`)
	return fmt.Sprintf(`"%s"`, escaped)
}

// FormatValue formats any Go value for AgentFile CONFIG syntax.
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
