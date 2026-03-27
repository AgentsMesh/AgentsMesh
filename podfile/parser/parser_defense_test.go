package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Cover parseProgram — comments handled by skipNewlines, verify no regression
func TestParse_CommentsEverywhere(t *testing.T) {
	prog, errs := Parse("# top\nAGENT test\n# mid\narg \"--a\"\n# end\n")
	require.Empty(t, errs)
	assert.Len(t, prog.Declarations, 1)
	assert.Len(t, prog.Statements, 1)
}

// Cover parseLiteralValue default error branch (parser.go:155-157)
// When CONFIG default is not a valid literal (e.g., a comma)
func TestParse_ConfigInvalidDefault(t *testing.T) {
	_, errs := Parse(`CONFIG model BOOL = ,`)
	assert.NotEmpty(t, errs)
}

// Cover parseLiteralValue number-as-string fallback (parser.go:151)
// A non-float number literal like octal
func TestParse_ConfigOctalDefault(t *testing.T) {
	prog, errs := Parse(`CONFIG mode NUMBER = 0644`)
	require.Empty(t, errs)
	cfg := prog.Declarations[0].(*ConfigDecl)
	// 0644 parses as float64(644) or similar
	assert.NotNil(t, cfg.Default)
}

// Cover objectLit inner RBRACE check after skipNewlines (parser_expr.go:134)
// Comma after entry creates a loop iteration where skipNewlines hits } on inner check
func TestParse_ObjectLitNewlineBeforeClose(t *testing.T) {
	prog, errs := Parse("AGENT test\nx = { a: \"1\",\n\n}\n")
	require.Empty(t, errs)
	assign := prog.Statements[0].(*AssignStmt)
	obj := assign.Value.(*ObjectLit)
	assert.Len(t, obj.Fields, 1)
}

// Cover objectLit atEnd guard — truncated mid-object
func TestParse_UnclosedObjectLit(t *testing.T) {
	_, errs := Parse("AGENT test\nx = { a: \"1\"")
	assert.NotEmpty(t, errs)
}

// Cover listLit inner RBRACKET check after skipNewlines (parser_expr.go:156)
func TestParse_ListLitNewlineBeforeClose(t *testing.T) {
	prog, errs := Parse("AGENT test\nx = [\"a\",\n\n]\n")
	require.Empty(t, errs)
	assign := prog.Statements[0].(*AssignStmt)
	list := assign.Value.(*ListLit)
	assert.Len(t, list.Elements, 1)
}

// Cover listLit atEnd guard — truncated mid-list
func TestParse_UnclosedListLit(t *testing.T) {
	_, errs := Parse("AGENT test\nx = [\"a\"")
	assert.NotEmpty(t, errs)
}

// Cover parseBlock with comment right before closing brace (parser_stmt.go:167)
func TestParse_BlockCommentBeforeClose(t *testing.T) {
	prog, errs := Parse("AGENT test\nif true {\narg \"--x\"\n# comment before close\n}\n")
	require.Empty(t, errs)
	ifStmt := prog.Statements[0].(*IfStmt)
	assert.Len(t, ifStmt.Body, 1) // only arg, comment skipped
}

// Cover parseBlock atEnd guard — unclosed block
func TestParse_UnclosedBlock(t *testing.T) {
	_, errs := Parse("AGENT test\nif true {\narg \"--x\"\n")
	assert.NotEmpty(t, errs)
}
