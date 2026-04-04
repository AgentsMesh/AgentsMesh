// Package serialize converts an AgentFile AST back to valid AgentFile source code.
// This enables AST-level merge followed by serialization, replacing fragile
// string concatenation approaches.
package serialize

import (
	"fmt"
	"strings"

	"github.com/anthropics/agentsmesh/agentfile/parser"
)

// Serialize converts a parsed Program AST back to AgentFile source code.
// The output is round-trip safe: Parse(Serialize(prog)) produces an
// equivalent AST.
func Serialize(prog *parser.Program) string {
	var b strings.Builder
	for i, decl := range prog.Declarations {
		if i > 0 {
			b.WriteByte('\n')
		}
		writeDecl(&b, decl)
	}
	if len(prog.Declarations) > 0 && len(prog.Statements) > 0 {
		b.WriteByte('\n')
	}
	for i, stmt := range prog.Statements {
		if i > 0 {
			b.WriteByte('\n')
		}
		writeStmt(&b, stmt, 0)
	}
	if b.Len() > 0 {
		b.WriteByte('\n')
	}
	return b.String()
}

func writeDecl(b *strings.Builder, decl parser.Declaration) {
	switch d := decl.(type) {
	case *parser.AgentDecl:
		writeAgentDecl(b, d)
	case *parser.ExecutableDecl:
		fmt.Fprintf(b, "EXECUTABLE %s", quoteIfNeeded(d.Name))
	case *parser.ConfigDecl:
		writeConfigDecl(b, d)
	case *parser.EnvDecl:
		writeEnvDecl(b, d)
	case *parser.RepoDecl:
		fmt.Fprintf(b, "REPO %s", serializeExpr(d.Value))
	case *parser.BranchDecl:
		fmt.Fprintf(b, "BRANCH %s", serializeExpr(d.Value))
	case *parser.GitCredentialDecl:
		fmt.Fprintf(b, "GIT_CREDENTIAL %s", d.Type)
	case *parser.McpDecl:
		writeMcpDecl(b, d)
	case *parser.SkillsDecl:
		writeSkillsDecl(b, d)
	case *parser.SetupDecl:
		writeSetupDecl(b, d)
	case *parser.RemoveDecl:
		fmt.Fprintf(b, "REMOVE %s %s", d.Target, QuoteString(d.Name))
	case *parser.ModeDecl:
		fmt.Fprintf(b, "MODE %s", d.Mode)
	case *parser.ModeArgsDecl:
		fmt.Fprintf(b, "MODE %s", d.Mode)
		for _, arg := range d.Args {
			fmt.Fprintf(b, " %s", QuoteString(arg))
		}
	case *parser.CredentialDecl:
		fmt.Fprintf(b, "CREDENTIAL %s", quoteIfNeeded(d.ProfileName))
	case *parser.PromptDecl:
		fmt.Fprintf(b, "PROMPT %q", d.Content)
	case *parser.PromptPositionDecl:
		fmt.Fprintf(b, "PROMPT_POSITION %s", d.Mode)
	}
}

func writeAgentDecl(b *strings.Builder, d *parser.AgentDecl) {
	fmt.Fprintf(b, "AGENT %s", quoteIfNeeded(d.Command))
}

func writeConfigDecl(b *strings.Builder, d *parser.ConfigDecl) {
	b.WriteString("CONFIG ")
	b.WriteString(d.Name)
	if d.TypeName != "" {
		b.WriteByte(' ')
		b.WriteString(configTypeKeyword(d.TypeName))
		if d.TypeName == "select" && len(d.Options) > 0 {
			b.WriteByte('(')
			for i, opt := range d.Options {
				if i > 0 {
					b.WriteString(", ")
				}
				b.WriteString(QuoteString(opt))
			}
			b.WriteByte(')')
		}
	}
	if d.Default != nil {
		b.WriteString(" = ")
		b.WriteString(FormatValue(d.Default))
	}
}

func writeEnvDecl(b *strings.Builder, d *parser.EnvDecl) {
	b.WriteString("ENV ")
	b.WriteString(d.Name)
	if d.Source != "" {
		b.WriteByte(' ')
		b.WriteString(strings.ToUpper(d.Source))
		if d.Optional {
			b.WriteString(" OPTIONAL")
		}
	} else if d.ValueExpr != nil {
		fmt.Fprintf(b, " = %s", serializeExpr(d.ValueExpr))
		writeWhen(b, d.When)
	} else {
		fmt.Fprintf(b, " = %s", QuoteString(d.Value))
	}
}

func writeMcpDecl(b *strings.Builder, d *parser.McpDecl) {
	if d.Enabled {
		b.WriteString("MCP ON")
		if d.Format != "" {
			fmt.Fprintf(b, " FORMAT %s", d.Format)
		}
	} else {
		b.WriteString("MCP OFF")
	}
}

func writeSkillsDecl(b *strings.Builder, d *parser.SkillsDecl) {
	b.WriteString("SKILLS ")
	b.WriteString(strings.Join(d.Slugs, ", "))
}

func writeSetupDecl(b *strings.Builder, d *parser.SetupDecl) {
	b.WriteString("SETUP")
	if d.Timeout != 300 {
		fmt.Fprintf(b, " timeout=%d", d.Timeout)
	}
	marker := chooseHeredocMarker(d.Script)
	fmt.Fprintf(b, " <<%s\n", marker)
	b.WriteString(d.Script)
	fmt.Fprintf(b, "\n%s", marker)
}

// configTypeKeyword maps internal type names back to AgentFile keywords.
func configTypeKeyword(typeName string) string {
	switch typeName {
	case "boolean":
		return "BOOL"
	case "string":
		return "STRING"
	case "number":
		return "NUMBER"
	case "secret":
		return "SECRET"
	case "select":
		return "SELECT"
	default:
		return strings.ToUpper(typeName)
	}
}

// FormatValue formats a Go value to its AgentFile literal representation.
// Strings are double-quoted with proper escaping; bools are true/false;
// float64 integers are rendered without decimals.
func FormatValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		return QuoteString(val)
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
		return fmt.Sprintf("%q", fmt.Sprintf("%v", val))
	}
}
