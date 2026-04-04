package serialize

import (
	"fmt"
	"strings"

	"github.com/anthropics/agentsmesh/agentfile/parser"
)

func writeStmt(b *strings.Builder, stmt parser.Statement, indent int) {
	prefix := strings.Repeat("\t", indent)
	b.WriteString(prefix)

	switch s := stmt.(type) {
	case *parser.ArgStmt:
		writeArgStmt(b, s)
	case *parser.FileStmt:
		writeFileStmt(b, s)
	case *parser.MkdirStmt:
		fmt.Fprintf(b, "mkdir %s", serializeExpr(s.Path))
	case *parser.AssignStmt:
		fmt.Fprintf(b, "%s = %s", s.Name, serializeExpr(s.Value))
	case *parser.IfStmt:
		writeIfStmt(b, s, indent)
	case *parser.ForStmt:
		writeForStmt(b, s, indent)
	}
}

func writeArgStmt(b *strings.Builder, s *parser.ArgStmt) {
	b.WriteString("arg")
	for _, arg := range s.Args {
		b.WriteByte(' ')
		b.WriteString(serializeExpr(arg))
	}
	writeWhen(b, s.When)
}

func writeFileStmt(b *strings.Builder, s *parser.FileStmt) {
	fmt.Fprintf(b, "file %s %s", serializeExpr(s.Path), serializeExpr(s.Content))
	if s.Mode != 0 {
		fmt.Fprintf(b, " 0%o", s.Mode)
	}
	writeWhen(b, s.When)
}

func writeIfStmt(b *strings.Builder, s *parser.IfStmt, indent int) {
	fmt.Fprintf(b, "if %s", serializeExpr(s.Condition))
	writeBlock(b, s.Body, indent)
	if len(s.Else) > 0 {
		b.WriteString(" else")
		writeBlock(b, s.Else, indent)
	}
}

func writeForStmt(b *strings.Builder, s *parser.ForStmt, indent int) {
	b.WriteString("for ")
	b.WriteString(s.Key)
	if s.Value != "" {
		b.WriteString(", ")
		b.WriteString(s.Value)
	}
	fmt.Fprintf(b, " in %s", serializeExpr(s.Iter))
	writeBlock(b, s.Body, indent)
}

func writeBlock(b *strings.Builder, stmts []parser.Statement, indent int) {
	b.WriteString(" {\n")
	for _, s := range stmts {
		writeStmt(b, s, indent+1)
		b.WriteByte('\n')
	}
	b.WriteString(strings.Repeat("\t", indent))
	b.WriteByte('}')
}

func writeWhen(b *strings.Builder, when parser.Expr) {
	if when != nil {
		fmt.Fprintf(b, " when %s", serializeExpr(when))
	}
}
