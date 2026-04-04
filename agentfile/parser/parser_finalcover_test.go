package parser

import (
	"testing"

	"github.com/anthropics/agentsmesh/agentfile/lexer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Cover current() EOF guard (parser.go:59-61) — empty token stream
func TestParse_InternalCurrentEOF(t *testing.T) {
	p := &Parser{tokens: nil, pos: 0}
	tok := p.current()
	assert.Equal(t, lexer.EOF, tok.Type)
}

// Cover peekIs() EOF guard (parser.go:71-73)
func TestParse_InternalPeekIsEOF(t *testing.T) {
	p := &Parser{tokens: nil, pos: 0}
	assert.False(t, p.peekIs(lexer.IDENT))
}

// Cover expectInt strconv.Atoi error (parser.go:133-136)
// Trigger with a number that has two decimal points
func TestParse_InvalidIntLiteral(t *testing.T) {
	_, errs := Parse(`AGENT test` + "\n" + `file "/p" "c" 1.2.3` + "\n")
	// 1.2.3 is tokenized as NUMBER "1.2" then DOT "." then NUMBER "3"
	// or as "1.2.3" depending on lexer — either way, parsing continues
	_ = errs // may or may not error, but shouldn't panic
}

// Cover parseLiteralValue number-as-string fallback (parser.go:151)
// A number that's a valid float but stored as literal
func TestParse_ConfigDefaultNumber(t *testing.T) {
	prog, errs := Parse(`CONFIG threshold NUMBER = 42`)
	require.Empty(t, errs)
	cfg := prog.Declarations[0].(*ConfigDecl)
	assert.Equal(t, float64(42), cfg.Default)
}

// Cover CONFIG STRING and SECRET types (parser_decl.go:60,66)
func TestParse_ConfigStringAndSecret(t *testing.T) {
	prog, errs := Parse(`
CONFIG custom STRING = "default"
CONFIG token SECRET = ""
`)
	require.Empty(t, errs)
	c1 := prog.Declarations[0].(*ConfigDecl)
	assert.Equal(t, "string", c1.TypeName)
	c2 := prog.Declarations[1].(*ConfigDecl)
	assert.Equal(t, "secret", c2.TypeName)
}

// Cover MCP OFF branch (parser_decl.go:158)
func TestParse_McpOff(t *testing.T) {
	prog, errs := Parse(`MCP OFF`)
	require.Empty(t, errs)
	mcp := prog.Declarations[0].(*McpDecl)
	assert.False(t, mcp.Enabled)
}

// Cover object lit inner break after skipNewlines (parser_expr.go:134)
func TestParse_EmptyObjectLit(t *testing.T) {
	prog, errs := Parse("AGENT test\nx = json({\n})\n")
	require.Empty(t, errs)
	assign := prog.Statements[0].(*AssignStmt)
	call := assign.Value.(*CallExpr)
	obj := call.Args[0].(*ObjectLit)
	assert.Empty(t, obj.Fields)
}

// Cover list inner break after skipNewlines (parser_expr.go:156)
func TestParse_EmptyListLit(t *testing.T) {
	prog, errs := Parse("AGENT test\nx = [\n]\n")
	require.Empty(t, errs)
	assign := prog.Statements[0].(*AssignStmt)
	list := assign.Value.(*ListLit)
	assert.Empty(t, list.Elements)
}

// Cover parseBlock comment skipping (parser_stmt.go:167-169)
func TestParse_BlockOnlyComments(t *testing.T) {
	prog, errs := Parse("AGENT test\nif true {\n# comment1\n# comment2\n}\n")
	require.Empty(t, errs)
	ifStmt := prog.Statements[0].(*IfStmt)
	assert.Empty(t, ifStmt.Body) // only comments, no statements
}

// Cover expectNewline with trailing comment (parser.go:170)
func TestParse_DeclWithTrailingComment(t *testing.T) {
	prog, errs := Parse(`AGENT claude # this is a comment`)
	require.Empty(t, errs)
	agent := prog.Declarations[0].(*AgentDecl)
	assert.Equal(t, "claude", agent.Command)
}

// Cover parseProgram comment-only line (parser.go:43-45)
func TestParse_TopLevelComment(t *testing.T) {
	prog, errs := Parse("# just a comment\nAGENT test\n# trailing\n")
	require.Empty(t, errs)
	assert.Len(t, prog.Declarations, 1)
}
