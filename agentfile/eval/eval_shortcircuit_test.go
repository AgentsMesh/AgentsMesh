package eval

import (
	"testing"

	"github.com/anthropics/agentsmesh/agentfile/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShortCircuit_AndFalseSkipsRight(t *testing.T) {
	// "and" with falsy left should NOT evaluate right side.
	// json_parse("<<<invalid") would error if evaluated.
	src := `
if flag and json_parse("<<<invalid") {
  arg "--never"
}
`
	prog, errs := parser.Parse(src)
	require.Empty(t, errs)
	ctx := NewContext(nil)
	ctx.Set("flag", false)
	require.NoError(t, Eval(prog, ctx))
	assert.Empty(t, ctx.Result.LaunchArgs)
}

func TestShortCircuit_OrTrueSkipsRight(t *testing.T) {
	// "or" with truthy left should NOT evaluate right side.
	src := `
if flag or json_parse("<<<invalid") {
  arg "--yes"
}
`
	prog, errs := parser.Parse(src)
	require.Empty(t, errs)
	ctx := NewContext(nil)
	ctx.Set("flag", true)
	require.NoError(t, Eval(prog, ctx))
	assert.Contains(t, ctx.Result.LaunchArgs, "--yes")
}

func TestShortCircuit_AndTrueEvaluatesRight(t *testing.T) {
	src := `
if a and b {
  arg "--both"
}
`
	prog, errs := parser.Parse(src)
	require.Empty(t, errs)
	ctx := NewContext(nil)
	ctx.Set("a", true)
	ctx.Set("b", true)
	require.NoError(t, Eval(prog, ctx))
	assert.Contains(t, ctx.Result.LaunchArgs, "--both")
}

func TestShortCircuit_OrFalseEvaluatesRight(t *testing.T) {
	src := `
if a or b {
  arg "--either"
}
`
	prog, errs := parser.Parse(src)
	require.Empty(t, errs)
	ctx := NewContext(nil)
	ctx.Set("a", false)
	ctx.Set("b", true)
	require.NoError(t, Eval(prog, ctx))
	assert.Contains(t, ctx.Result.LaunchArgs, "--either")
}

func TestForLoop_ExceedsLimit(t *testing.T) {
	ctx := NewContext(nil)

	bigList := make([]interface{}, maxForIterations+1)
	for i := range bigList {
		bigList[i] = float64(i)
	}
	ctx.Set("items", bigList)

	forStmt := &parser.ForStmt{
		Key:  "item",
		Iter: &parser.Ident{Name: "items"},
		Body: []parser.Statement{},
	}
	err := evalForStmt(ctx, forStmt)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds limit")
}
