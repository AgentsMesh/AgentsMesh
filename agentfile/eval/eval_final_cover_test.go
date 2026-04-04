package eval

import (
	"testing"

	"github.com/anthropics/agentsmesh/agentfile/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Cover Eval decl error propagation (evaluator.go:14-16)
func TestEval_DeclError(t *testing.T) {
	ctx := NewContext(nil)
	// Create program with a decl that errors: RepoDecl with nil Value
	prog := &parser.Program{
		Declarations: []parser.Declaration{
			&parser.RepoDecl{Value: nil, Position: parser.Position{Line: 1}},
		},
	}
	err := Eval(prog, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "line 1")
}

// Cover evalForStmt map single-var (evaluator.go:167-169)
func TestEval_ForMapSingleVar(t *testing.T) {
	prog, errs := parser.Parse(`
AGENT test
for key in items {
  arg key
}
`)
	require.Empty(t, errs)
	ctx := NewContext(map[string]interface{}{
		"items": map[string]interface{}{"a": "1", "b": "2"},
	})
	require.NoError(t, Eval(prog, ctx))
	assert.Len(t, ctx.Result.LaunchArgs, 2)
}

// Cover interpolate regex submatch < 2 branch (values.go:82-84)
// This is nearly impossible to trigger with real input since the regex
// always captures group 1, but we verify the short-circuit path
func TestInterpolate_NoMatch(t *testing.T) {
	ctx := NewContext(nil)
	// String with ${ but not matching the pattern
	assert.Equal(t, "${ }", interpolate(ctx, "${ }"))
}

// Cover builtinMCPTransform non-map server entry (builtins.go:82-84)
func TestBuiltinMCPTransform_NonMapServer(t *testing.T) {
	servers := map[string]interface{}{
		"good":    map[string]interface{}{"url": "http://localhost"},
		"bad":     "not a map",
	}
	result, err := builtinMCPTransform(servers, "claude")
	require.NoError(t, err)
	m := result.(map[string]interface{})
	assert.Equal(t, "not a map", m["bad"]) // preserved as-is
}

// Cover evalUnaryExpr operand error (eval_expr.go:98-100)
func TestEval_UnaryExprOperandError(t *testing.T) {
	ctx := NewContext(nil)
	_, err := evalUnaryExpr(ctx, &parser.UnaryExpr{
		Op:      "not",
		Operand: nil, // nil causes error
	})
	assert.Error(t, err)
}

// Cover evalRemoveDecl CONFIG branch (eval_decl.go:49)
func TestEval_RemoveDeclConfig(t *testing.T) {
	prog, errs := parser.Parse(`
AGENT test
REMOVE CONFIG model
`)
	require.Empty(t, errs)
	ctx := NewContext(nil)
	require.NoError(t, Eval(prog, ctx))
	// CONFIG removal is metadata only — no error, no build effect
}

// Cover NumberLit fallback (eval_expr.go:20) — non-parseable number
func TestEval_NumberLitFallback(t *testing.T) {
	ctx := NewContext(nil)
	// A number that doesn't parse as float (shouldn't happen normally)
	val, err := evalExpr(ctx, &parser.NumberLit{Value: "42"})
	require.NoError(t, err)
	assert.Equal(t, float64(42), val)

	// A string that somehow got into NumberLit
	val2, err := evalExpr(ctx, &parser.NumberLit{Value: "not_a_number"})
	require.NoError(t, err)
	assert.Equal(t, "not_a_number", val2) // fallback returns string
}
