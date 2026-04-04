package eval

import (
	"testing"

	"github.com/anthropics/agentsmesh/agentfile/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Cover evalBinaryExpr unknown operator error path
func TestEval_BinaryExprUnknownOp(t *testing.T) {
	ctx := NewContext(nil)
	_, err := evalBinaryExpr(ctx, &parser.BinaryExpr{
		Left: &parser.StringLit{Value: "a"}, Op: "**", Right: &parser.StringLit{Value: "b"},
	})
	assert.Error(t, err)
}

// Cover evalUnaryExpr unknown operator error path
func TestEval_UnaryExprUnknownOp(t *testing.T) {
	ctx := NewContext(nil)
	_, err := evalUnaryExpr(ctx, &parser.UnaryExpr{Op: "~", Operand: &parser.BoolLit{Value: true}})
	assert.Error(t, err)
}

// Cover evalExpr unknown type error path
func TestEval_ExprUnknownType(t *testing.T) {
	ctx := NewContext(nil)
	_, err := evalExpr(ctx, nil) // nil expr not in switch
	assert.Error(t, err)
}

// Cover evalStmt unknown type error path
func TestEval_StmtUnknownType(t *testing.T) {
	ctx := NewContext(nil)
	err := evalStmt(ctx, nil) // nil stmt not in switch
	assert.Error(t, err)
}

// Cover Eval error propagation from declarations
func TestEval_DeclErrorPropagation(t *testing.T) {
	// REPO with expression that errors during eval
	prog, errs := parser.Parse(`
AGENT test
REPO input.url
`)
	require.Empty(t, errs)
	ctx := NewContext(nil) // input not injected → nil
	// This should succeed (nil resolves to "")
	require.NoError(t, Eval(prog, ctx))
}

// Cover evalCallExpr error propagation from arg eval
func TestEval_CallExprArgError(t *testing.T) {
	ctx := NewContext(nil)
	// Call with arg that references unknown type
	_, err := evalCallExpr(ctx, &parser.CallExpr{
		Func: "json",
		Args: []parser.Expr{nil}, // nil causes error
	})
	assert.Error(t, err)
}

// Cover evalObjectLit error propagation
func TestEval_ObjectLitError(t *testing.T) {
	ctx := NewContext(nil)
	_, err := evalObjectLit(ctx, &parser.ObjectLit{
		Fields: []parser.ObjectField{{Key: "k", Value: nil}},
	})
	assert.Error(t, err)
}

// Cover evalListLit error propagation
func TestEval_ListLitError(t *testing.T) {
	ctx := NewContext(nil)
	_, err := evalListLit(ctx, &parser.ListLit{Elements: []parser.Expr{nil}})
	assert.Error(t, err)
}

// Cover builtinJSON marshal error
func TestEval_BuiltinJSON_MarshalError(t *testing.T) {
	// func values can't be marshaled
	_, err := builtinJSON(func() {})
	assert.Error(t, err)
}

// Cover evalDotExpr error propagation
func TestEval_DotExprLeftError(t *testing.T) {
	ctx := NewContext(nil)
	_, err := evalDotExpr(ctx, &parser.DotExpr{Left: nil, Field: "x"})
	assert.Error(t, err)
}

// Cover evalArgStmt when clause error
func TestEval_ArgStmtWhenError(t *testing.T) {
	prog, errs := parser.Parse(`
AGENT test
arg "--flag" when undefined_func()
`)
	require.Empty(t, errs)
	ctx := NewContext(nil)
	err := Eval(prog, ctx)
	assert.Error(t, err)
}

// Cover evalFileStmt content error
func TestEval_FileStmtContentError(t *testing.T) {
	ctx := NewContext(nil)
	err := evalFileStmt(ctx, &parser.FileStmt{
		Path:    &parser.StringLit{Value: "/path"},
		Content: nil, // nil causes error
	})
	assert.Error(t, err)
}

// Cover evalFileStmt path error
func TestEval_FileStmtPathError(t *testing.T) {
	ctx := NewContext(nil)
	err := evalFileStmt(ctx, &parser.FileStmt{
		Path:    nil,
		Content: &parser.StringLit{Value: "content"},
	})
	assert.Error(t, err)
}

// Cover evalMkdirStmt error
func TestEval_MkdirStmtError(t *testing.T) {
	ctx := NewContext(nil)
	err := evalMkdirStmt(ctx, &parser.MkdirStmt{Path: nil})
	assert.Error(t, err)
}

// Cover evalAssignStmt error
func TestEval_AssignStmtError(t *testing.T) {
	ctx := NewContext(nil)
	err := evalAssignStmt(ctx, &parser.AssignStmt{Name: "x", Value: nil})
	assert.Error(t, err)
}

// Cover evalIfStmt condition error
func TestEval_IfStmtCondError(t *testing.T) {
	ctx := NewContext(nil)
	err := evalIfStmt(ctx, &parser.IfStmt{
		Condition: nil,
		Body:      nil,
	})
	assert.Error(t, err)
}

// Cover evalRemoveDecl with arg/file targets
func TestEval_RemoveDeclArgFile(t *testing.T) {
	ctx := NewContext(nil)
	err := evalRemoveDecl(ctx, &parser.RemoveDecl{Target: "arg", Name: "--verbose"})
	require.NoError(t, err)
	assert.Equal(t, []string{"--verbose"}, ctx.Result.RemoveArgs)

	err = evalRemoveDecl(ctx, &parser.RemoveDecl{Target: "file", Name: "/tmp/f"})
	require.NoError(t, err)
	assert.Equal(t, []string{"/tmp/f"}, ctx.Result.RemoveFiles)
}

// Cover evalForStmt iter error
func TestEval_ForStmtIterError(t *testing.T) {
	ctx := NewContext(nil)
	err := evalForStmt(ctx, &parser.ForStmt{Key: "k", Iter: nil, Body: nil})
	assert.Error(t, err)
}
