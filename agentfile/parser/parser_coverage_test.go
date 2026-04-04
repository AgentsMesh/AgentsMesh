package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Cover parseOr (multi-or chain)
func TestParse_OrChain(t *testing.T) {
	prog, errs := Parse(`
AGENT test
arg "--x" when a or b or c
`)
	require.Empty(t, errs)
	argStmt := prog.Statements[0].(*ArgStmt)
	// a or b or c → BinaryExpr(BinaryExpr(a, or, b), or, c)
	outer := argStmt.When.(*BinaryExpr)
	assert.Equal(t, "or", outer.Op)
	inner := outer.Left.(*BinaryExpr)
	assert.Equal(t, "or", inner.Op)
}

// Cover parseAnd (multi-and chain)
func TestParse_AndChain(t *testing.T) {
	prog, errs := Parse(`
AGENT test
arg "--x" when a and b and c
`)
	require.Empty(t, errs)
	argStmt := prog.Statements[0].(*ArgStmt)
	outer := argStmt.When.(*BinaryExpr)
	assert.Equal(t, "and", outer.Op)
}

// Cover != operator
func TestParse_NeqOperator(t *testing.T) {
	prog, errs := Parse(`
AGENT test
arg "--x" when config.val != "bad"
`)
	require.Empty(t, errs)
	argStmt := prog.Statements[0].(*ArgStmt)
	bin := argStmt.When.(*BinaryExpr)
	assert.Equal(t, "!=", bin.Op)
}

// Cover BoolLit as standalone expression
func TestParse_BoolLitExpr(t *testing.T) {
	prog, errs := Parse(`
AGENT test
x = true
y = false
`)
	require.Empty(t, errs)
	a1 := prog.Statements[0].(*AssignStmt)
	assert.True(t, a1.Value.(*BoolLit).Value)
	a2 := prog.Statements[1].(*AssignStmt)
	assert.False(t, a2.Value.(*BoolLit).Value)
}

// Cover block with comments inside
func TestParse_BlockWithComments(t *testing.T) {
	prog, errs := Parse(`
AGENT test
if true {
  # this is a comment
  arg "--flag"
  # another comment
}
`)
	require.Empty(t, errs)
	ifStmt := prog.Statements[0].(*IfStmt)
	assert.Len(t, ifStmt.Body, 1)
}

// Cover REMOVE decl error path — invalid target
func TestParse_RemoveDecl_InvalidTarget(t *testing.T) {
	_, errs := Parse(`
AGENT test
REMOVE mkdir "/path"
`)
	assert.NotEmpty(t, errs)
}

// Cover parseBlock error — unexpected token in block
func TestParse_BlockUnexpectedToken(t *testing.T) {
	_, errs := Parse(`
AGENT test
if true {
  AGENT inside_block
}
`)
	assert.NotEmpty(t, errs)
}

// Cover parseLiteralValue — number default
func TestParse_ConfigNumberDefault(t *testing.T) {
	prog, errs := Parse(`CONFIG temp NUMBER = 0.7`)
	require.Empty(t, errs)
	cfg := prog.Declarations[0].(*ConfigDecl)
	assert.Equal(t, 0.7, cfg.Default)
}

// Cover file stmt with when clause
func TestParse_FileWithWhen(t *testing.T) {
	prog, errs := Parse(`
AGENT test
file "/path" "content" 0644 when config.enabled
`)
	require.Empty(t, errs)
	fs := prog.Statements[0].(*FileStmt)
	assert.NotNil(t, fs.When)
	assert.Equal(t, 0644, fs.Mode)
}

// Cover object literal with comma in parser
func TestParse_ObjectComma(t *testing.T) {
	prog, errs := Parse(`
AGENT test
x = json({ a: "1", b: "2", c: "3" })
`)
	require.Empty(t, errs)
	assign := prog.Statements[0].(*AssignStmt)
	call := assign.Value.(*CallExpr)
	obj := call.Args[0].(*ObjectLit)
	assert.Len(t, obj.Fields, 3)
}

// Cover list with trailing comma
func TestParse_ListTrailingComma(t *testing.T) {
	prog, errs := Parse(`
AGENT test
x = ["a", "b",]
`)
	require.Empty(t, errs)
	assign := prog.Statements[0].(*AssignStmt)
	list := assign.Value.(*ListLit)
	assert.Len(t, list.Elements, 2)
}
