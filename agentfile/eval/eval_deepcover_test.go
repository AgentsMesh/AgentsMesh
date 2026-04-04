package eval

import (
	"testing"

	"github.com/anthropics/agentsmesh/agentfile/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Cover evalIfStmt body error propagation
func TestEval_IfBodyError(t *testing.T) {
	prog, errs := parser.Parse(`
AGENT test
if true {
  x = undefined_func()
}
`)
	require.Empty(t, errs)
	ctx := NewContext(nil)
	assert.Error(t, Eval(prog, ctx))
}

// Cover evalIfStmt else body error
func TestEval_IfElseBodyError(t *testing.T) {
	prog, errs := parser.Parse(`
AGENT test
if false {
  arg "ok"
} else {
  x = undefined_func()
}
`)
	require.Empty(t, errs)
	ctx := NewContext(nil)
	assert.Error(t, Eval(prog, ctx))
}

// Cover evalForStmt body error
func TestEval_ForBodyError(t *testing.T) {
	prog, errs := parser.Parse(`
AGENT test
for item in items {
  x = undefined_func()
}
`)
	require.Empty(t, errs)
	ctx := NewContext(map[string]interface{}{"items": []interface{}{"a"}})
	assert.Error(t, Eval(prog, ctx))
}

// Cover evalForStmt with map body error
func TestEval_ForMapBodyError(t *testing.T) {
	prog, errs := parser.Parse(`
AGENT test
for k, v in items {
  x = undefined_func()
}
`)
	require.Empty(t, errs)
	ctx := NewContext(map[string]interface{}{"items": map[string]interface{}{"a": "b"}})
	assert.Error(t, Eval(prog, ctx))
}

// Cover evalBlock error propagation
func TestEval_BlockError(t *testing.T) {
	ctx := NewContext(nil)
	err := evalBlock(ctx, []parser.Statement{
		&parser.ArgStmt{Args: []parser.Expr{nil}},
	})
	assert.Error(t, err)
}

// Cover evalArgStmt arg eval error
func TestEval_ArgStmtArgError(t *testing.T) {
	ctx := NewContext(nil)
	err := evalArgStmt(ctx, &parser.ArgStmt{
		Args: []parser.Expr{nil},
	})
	assert.Error(t, err)
}

// Cover evalEnvDecl ValueExpr when error
func TestEval_EnvDeclExprWhenError(t *testing.T) {
	ctx := NewContext(nil)
	err := evalEnvDecl(ctx, &parser.EnvDecl{
		Name:      "K",
		ValueExpr: &parser.StringLit{Value: "v"},
		When:      nil, // nil when expr → not the error we want
	})
	assert.NoError(t, err) // nil when = unconditional

	// Error in when clause
	err = evalEnvDecl(ctx, &parser.EnvDecl{
		Name:      "K",
		ValueExpr: &parser.StringLit{Value: "v"},
		When:      &parser.CallExpr{Func: "nope", Args: nil},
	})
	assert.Error(t, err)
}

// Cover evalEnvDecl ValueExpr value error
func TestEval_EnvDeclExprValueError(t *testing.T) {
	ctx := NewContext(nil)
	err := evalEnvDecl(ctx, &parser.EnvDecl{Name: "K", ValueExpr: nil})
	// nil ValueExpr with no Source and no Value → no-op, not an error
	assert.NoError(t, err)

	// Actual value error: expression that fails
	err = evalEnvDecl(ctx, &parser.EnvDecl{
		Name:      "K",
		ValueExpr: &parser.CallExpr{Func: "nope", Args: nil},
	})
	assert.Error(t, err)
}

// Cover evalFileStmt when error
func TestEval_FileStmtWhenError(t *testing.T) {
	ctx := NewContext(nil)
	err := evalFileStmt(ctx, &parser.FileStmt{
		Path:    &parser.StringLit{Value: "/p"},
		Content: &parser.StringLit{Value: "c"},
		When:    &parser.CallExpr{Func: "nope", Args: nil},
	})
	assert.Error(t, err)
}

// Cover toString for unknown type
func TestToString_UnknownType(t *testing.T) {
	result := toString(struct{ X int }{42})
	assert.Contains(t, result, "42")
}

// Cover evalBinaryExpr left error
func TestEval_BinaryExprLeftError(t *testing.T) {
	ctx := NewContext(nil)
	_, err := evalBinaryExpr(ctx, &parser.BinaryExpr{
		Left: nil, Op: "+", Right: &parser.StringLit{Value: "b"},
	})
	assert.Error(t, err)
}

// Cover evalBinaryExpr right error
func TestEval_BinaryExprRightError(t *testing.T) {
	ctx := NewContext(nil)
	_, err := evalBinaryExpr(ctx, &parser.BinaryExpr{
		Left: &parser.StringLit{Value: "a"}, Op: "+", Right: nil,
	})
	assert.Error(t, err)
}

// Cover builtinLen for default (unknown type)
func TestBuiltinLen_UnknownType(t *testing.T) {
	r, err := builtinLen(42)
	require.NoError(t, err)
	assert.Equal(t, float64(0), r)
}

// Cover evalDecl for BranchDecl eval error
func TestEval_BranchDeclEvalError(t *testing.T) {
	ctx := NewContext(nil)
	err := evalDecl(ctx, &parser.BranchDecl{
		Value: nil, // nil causes eval error
	})
	assert.Error(t, err)
}

// Cover evalDecl for RepoDecl eval error
func TestEval_RepoDeclEvalError(t *testing.T) {
	ctx := NewContext(nil)
	err := evalDecl(ctx, &parser.RepoDecl{
		Value: nil,
	})
	assert.Error(t, err)
}
