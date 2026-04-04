package eval

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuiltinJSONParse(t *testing.T) {
	result, err := builtinJSONParse(`{"key":"val"}`)
	require.NoError(t, err)
	m := result.(map[string]interface{})
	assert.Equal(t, "val", m["key"])
}

func TestBuiltinJSONParse_Invalid(t *testing.T) {
	_, err := builtinJSONParse("not json")
	assert.Error(t, err)
}

func TestBuiltinStrJoin_List(t *testing.T) {
	result, err := builtinStrJoin([]interface{}{"a", "b", "c"}, ",")
	require.NoError(t, err)
	assert.Equal(t, "a,b,c", result)
}

func TestBuiltinStrJoin_Map(t *testing.T) {
	result, err := builtinStrJoin(map[string]interface{}{"key": "val"}, ",")
	require.NoError(t, err)
	assert.Equal(t, "key", result)
}

func TestBuiltinStrJoin_InvalidType(t *testing.T) {
	_, err := builtinStrJoin("not a list", ",")
	assert.Error(t, err)
}

func TestBuiltinLen(t *testing.T) {
	// String
	r, err := builtinLen("hello")
	require.NoError(t, err)
	assert.Equal(t, float64(5), r)

	// Map
	r, err = builtinLen(map[string]interface{}{"a": 1, "b": 2})
	require.NoError(t, err)
	assert.Equal(t, float64(2), r)

	// List
	r, err = builtinLen([]interface{}{1, 2, 3})
	require.NoError(t, err)
	assert.Equal(t, float64(3), r)

	// Nil
	r, err = builtinLen(nil)
	require.NoError(t, err)
	assert.Equal(t, float64(0), r)
}

func TestBuiltinPrint(t *testing.T) {
	result, err := builtinPrint("hello", "world")
	require.NoError(t, err)
	assert.Equal(t, "hello world", result)
}

func TestBuiltinMCPTransform_OpenCode(t *testing.T) {
	servers := map[string]interface{}{
		"agentsmesh": map[string]interface{}{
			"type": "http",
			"url":  "http://localhost:19000/mcp",
		},
	}
	result, err := builtinMCPTransform(servers, "opencode")
	require.NoError(t, err)
	m := result.(map[string]interface{})
	srv := m["agentsmesh"].(map[string]interface{})
	assert.Equal(t, true, srv["enabled"])
	assert.Equal(t, "local", srv["type"])
	assert.Equal(t, []interface{}{"npx", "-y", "mcp-remote", "http://localhost:19000/mcp"}, srv["command"])
	assert.Nil(t, srv["url"]) // url removed, converted to command
}

func TestBuiltinMCPTransform_Codex(t *testing.T) {
	servers := map[string]interface{}{
		"agentsmesh": map[string]interface{}{
			"type": "http",
			"url":  "http://localhost:19000/mcp",
		},
	}
	result, err := builtinMCPTransform(servers, "codex")
	require.NoError(t, err)
	m := result.(map[string]interface{})
	srv := m["agentsmesh"].(map[string]interface{})
	// Codex: no transformation, fields preserved as-is
	assert.Equal(t, "http", srv["type"])
	assert.Equal(t, "http://localhost:19000/mcp", srv["url"])
}

func TestBuiltinJSON_Error(t *testing.T) {
	_, err := builtinJSON() // no args
	assert.Error(t, err)
}

func TestBuiltinJSONMerge_MinArgs(t *testing.T) {
	_, err := builtinJSONMerge(map[string]interface{}{}) // only 1 arg
	assert.Error(t, err)
}

func TestBuiltinJSONMerge_NonMapArgs(t *testing.T) {
	// Non-map args should be skipped
	result, err := builtinJSONMerge("not a map", map[string]interface{}{"k": "v"})
	require.NoError(t, err)
	m := result.(map[string]interface{})
	assert.Equal(t, "v", m["k"])
}

func TestBuiltinStrReplace_ArgCount(t *testing.T) {
	_, err := builtinStrReplace("a", "b") // needs 3
	assert.Error(t, err)
}

func TestBuiltinStrContains_ArgCount(t *testing.T) {
	_, err := builtinStrContains("a") // needs 2
	assert.Error(t, err)
}

func TestBuiltinStrJoin_ArgCount(t *testing.T) {
	_, err := builtinStrJoin([]interface{}{"a"}) // needs 2
	assert.Error(t, err)
}

func TestBuiltinLen_ArgCount(t *testing.T) {
	_, err := builtinLen() // needs 1
	assert.Error(t, err)
}

func TestBuiltinMCPTransform_ArgCount(t *testing.T) {
	_, err := builtinMCPTransform(map[string]interface{}{}) // needs 2
	assert.Error(t, err)
}

func TestBuiltinMCPTransform_NonMap(t *testing.T) {
	// Non-map input returned as-is
	result, err := builtinMCPTransform("not a map", "claude")
	require.NoError(t, err)
	assert.Equal(t, "not a map", result)
}

func TestBuiltinJSONParse_ArgCount(t *testing.T) {
	_, err := builtinJSONParse() // needs 1
	assert.Error(t, err)
}

func TestBuiltinPrint_Single(t *testing.T) {
	result, err := builtinPrint("hello")
	require.NoError(t, err)
	assert.Equal(t, "hello", result)
}
