package serialize

import (
	"fmt"
	"strings"

	"github.com/anthropics/agentsmesh/agentfile/parser"
)

// serializeExpr converts an expression AST node to its source representation.
func serializeExpr(expr parser.Expr) string {
	switch e := expr.(type) {
	case *parser.StringLit:
		return QuoteString(e.Value)
	case *parser.NumberLit:
		return e.Value
	case *parser.BoolLit:
		if e.Value {
			return "true"
		}
		return "false"
	case *parser.Ident:
		return e.Name
	case *parser.DotExpr:
		return serializeDotExpr(e)
	case *parser.BinaryExpr:
		return serializeBinaryExpr(e)
	case *parser.UnaryExpr:
		return fmt.Sprintf("not %s", serializeExpr(e.Operand))
	case *parser.CallExpr:
		return serializeCallExpr(e)
	case *parser.ObjectLit:
		return serializeObjectLit(e)
	case *parser.HeredocLit:
		return serializeHeredoc(e)
	case *parser.ListLit:
		return serializeListLit(e)
	default:
		return ""
	}
}

func serializeDotExpr(e *parser.DotExpr) string {
	return fmt.Sprintf("%s.%s", serializeExpr(e.Left), e.Field)
}

func serializeBinaryExpr(e *parser.BinaryExpr) string {
	left := serializeExpr(e.Left)
	right := serializeExpr(e.Right)
	return fmt.Sprintf("%s %s %s", left, e.Op, right)
}

func serializeCallExpr(e *parser.CallExpr) string {
	args := make([]string, len(e.Args))
	for i, arg := range e.Args {
		args[i] = serializeExpr(arg)
	}
	return fmt.Sprintf("%s(%s)", e.Func, strings.Join(args, ", "))
}

func serializeObjectLit(e *parser.ObjectLit) string {
	if len(e.Fields) == 0 {
		return "{}"
	}
	fields := make([]string, len(e.Fields))
	for i, f := range e.Fields {
		fields[i] = fmt.Sprintf("%s: %s", f.Key, serializeExpr(f.Value))
	}
	return fmt.Sprintf("{ %s }", strings.Join(fields, ", "))
}

func serializeHeredoc(e *parser.HeredocLit) string {
	marker := chooseHeredocMarker(e.Content)
	return fmt.Sprintf("<<%s\n%s\n%s", marker, e.Content, marker)
}

func serializeListLit(e *parser.ListLit) string {
	if len(e.Elements) == 0 {
		return "[]"
	}
	elems := make([]string, len(e.Elements))
	for i, el := range e.Elements {
		elems[i] = serializeExpr(el)
	}
	return fmt.Sprintf("[%s]", strings.Join(elems, ", "))
}

// QuoteString produces a double-quoted AgentFile string with proper escaping.
// Handles: \ → \\, " → \", \n → \n, \t → \t.
func QuoteString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, ch := range s {
		switch ch {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\t':
			b.WriteString(`\t`)
		default:
			b.WriteRune(ch)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// quoteIfNeeded quotes a string only if it contains special characters.
// Simple identifiers are returned as-is.
func quoteIfNeeded(s string) string {
	if s == "" {
		return `""`
	}
	for _, ch := range s {
		if !isIdentChar(ch) {
			return QuoteString(s)
		}
	}
	return s
}

func isIdentChar(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9') || ch == '_' || ch == '-'
}

// chooseHeredocMarker picks a heredoc marker that doesn't appear in content.
func chooseHeredocMarker(content string) string {
	candidates := []string{"EOF", "HEREDOC", "END", "SCRIPT", "DOC"}
	for _, m := range candidates {
		if !strings.Contains(content, m) {
			return m
		}
	}
	// Fallback: append a number
	for i := 0; ; i++ {
		m := fmt.Sprintf("EOF%d", i)
		if !strings.Contains(content, m) {
			return m
		}
	}
}
