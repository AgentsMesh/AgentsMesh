package eval

import (
	"testing"

	"github.com/anthropics/agentsmesh/agentfile/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Cover evalRemoveDecl (0% → package-local test)
func TestEval_RemoveDecl(t *testing.T) {
	prog, errs := parser.Parse(`
AGENT test
ENV KEY1 SECRET
ENV KEY2 SECRET
SKILLS s1, s2
REMOVE ENV KEY2
REMOVE SKILLS s1
`)
	require.Empty(t, errs)

	ctx := NewContext(nil)
	ctx.Credentials = map[string]string{"KEY1": "v1", "KEY2": "v2"}
	require.NoError(t, Eval(prog, ctx))
	ApplyRemoves(ctx.Result)

	assert.Equal(t, "v1", ctx.Result.EnvVars["KEY1"])
	_, hasKey2 := ctx.Result.EnvVars["KEY2"]
	assert.False(t, hasKey2)
	assert.Equal(t, []string{"s2"}, ctx.Result.Skills)
}

// Cover evalRemoveDecl for arg/file targets (0% → package-local test)
func TestEval_RemoveArgFileDecl(t *testing.T) {
	prog, errs := parser.Parse(`
AGENT test
arg "--verbose"
arg "--model" "opus"
file "/a/b.json" "content"
REMOVE arg "--verbose"
REMOVE file "/a/b.json"
`)
	require.Empty(t, errs)

	ctx := NewContext(nil)
	require.NoError(t, Eval(prog, ctx))
	ApplyRemoves(ctx.Result)

	assert.Equal(t, []string{"--model", "opus"}, ctx.Result.LaunchArgs)
	assert.Empty(t, ctx.Result.FilesToCreate)
}

// Cover evalUnaryExpr (not)
func TestEval_NotExpr(t *testing.T) {
	prog, errs := parser.Parse(`
AGENT test
arg "--disabled" when not config.enabled
arg "--enabled" when config.enabled
`)
	require.Empty(t, errs)

	ctx := NewContext(map[string]interface{}{
		"config": map[string]interface{}{"enabled": false},
	})
	require.NoError(t, Eval(prog, ctx))
	assert.Equal(t, []string{"--disabled"}, ctx.Result.LaunchArgs)
}

// Cover evalEnvDecl with ValueExpr (unified from old env stmt)
func TestEval_EnvDeclWithExpr(t *testing.T) {
	prog, errs := parser.Parse(`
AGENT test
ENV DYNAMIC_KEY = config.value when config.value != ""
ENV STATIC_KEY = "always"
`)
	require.Empty(t, errs)

	ctx := NewContext(map[string]interface{}{
		"config": map[string]interface{}{"value": "hello"},
	})
	require.NoError(t, Eval(prog, ctx))
	assert.Equal(t, "hello", ctx.Result.EnvVars["DYNAMIC_KEY"])
	assert.Equal(t, "always", ctx.Result.EnvVars["STATIC_KEY"])
}

// Cover evalEnvDecl when=false
func TestEval_EnvDeclWhenFalse(t *testing.T) {
	prog, errs := parser.Parse(`
AGENT test
ENV KEY = "val" when config.enabled
`)
	require.Empty(t, errs)
	ctx := NewContext(map[string]interface{}{
		"config": map[string]interface{}{"enabled": false},
	})
	require.NoError(t, Eval(prog, ctx))
	_, has := ctx.Result.EnvVars["KEY"]
	assert.False(t, has)
}

// Cover evalFileStmt when=false branch
func TestEval_FileStmtWhenFalse(t *testing.T) {
	prog, errs := parser.Parse(`
AGENT test
file "/path" "content" when config.enabled
`)
	require.Empty(t, errs)
	ctx := NewContext(map[string]interface{}{
		"config": map[string]interface{}{"enabled": false},
	})
	require.NoError(t, Eval(prog, ctx))
	assert.Empty(t, ctx.Result.FilesToCreate)
}

// Cover evalFileStmt with mode
func TestEval_FileStmtWithMode(t *testing.T) {
	prog, errs := parser.Parse(`
AGENT test
file "/path" "content" 0755
`)
	require.Empty(t, errs)
	ctx := NewContext(nil)
	require.NoError(t, Eval(prog, ctx))
	require.Len(t, ctx.Result.FilesToCreate, 1)
	assert.Equal(t, 0755, ctx.Result.FilesToCreate[0].Mode)
}

// Cover or expression
func TestEval_OrExpression(t *testing.T) {
	prog, errs := parser.Parse(`
AGENT test
arg "--match" when config.a or config.b
`)
	require.Empty(t, errs)

	ctx := NewContext(map[string]interface{}{
		"config": map[string]interface{}{"a": false, "b": true},
	})
	require.NoError(t, Eval(prog, ctx))
	assert.Equal(t, []string{"--match"}, ctx.Result.LaunchArgs)
}

// Cover and expression (both true)
func TestEval_AndExpression(t *testing.T) {
	prog, errs := parser.Parse(`
AGENT test
arg "--both" when config.a and config.b
`)
	require.Empty(t, errs)

	ctx := NewContext(map[string]interface{}{
		"config": map[string]interface{}{"a": true, "b": true},
	})
	require.NoError(t, Eval(prog, ctx))
	assert.Equal(t, []string{"--both"}, ctx.Result.LaunchArgs)
}

// Cover for loop error (non-iterable)
func TestEval_ForLoopError(t *testing.T) {
	prog, errs := parser.Parse(`
AGENT test
for x in items {
  arg x
}
`)
	require.Empty(t, errs)

	ctx := NewContext(map[string]interface{}{"items": "not iterable"})
	err := Eval(prog, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected map or list")
}

// Cover undefined function call error
func TestEval_UndefinedFunction(t *testing.T) {
	prog, errs := parser.Parse(`
AGENT test
x = nonexistent()
`)
	require.Empty(t, errs)
	ctx := NewContext(nil)
	err := Eval(prog, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "undefined function")
}

// Cover credential empty string not injected
func TestEval_CredentialEmptyString(t *testing.T) {
	prog, errs := parser.Parse(`
AGENT test
ENV MY_KEY SECRET
`)
	require.Empty(t, errs)
	ctx := NewContext(nil)
	ctx.Credentials = map[string]string{"MY_KEY": ""}
	require.NoError(t, Eval(prog, ctx))
	_, has := ctx.Result.EnvVars["MY_KEY"]
	assert.False(t, has) // empty string not injected
}

// Cover dot access on nil
func TestEval_DotAccessOnNil(t *testing.T) {
	prog, errs := parser.Parse(`
AGENT test
x = missing.field
arg "val"
`)
	require.Empty(t, errs)
	ctx := NewContext(nil)
	require.NoError(t, Eval(prog, ctx))
	// missing.field is nil, no error
	val, _ := ctx.Get("x")
	assert.Nil(t, val)
}

// Cover GetNested on non-map
func TestGetNested_NonMap(t *testing.T) {
	val, ok := GetNested("not a map", "key")
	assert.Nil(t, val)
	assert.False(t, ok)
}
