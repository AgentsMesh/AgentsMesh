package eval

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsTruthy(t *testing.T) {
	tests := []struct {
		val  interface{}
		want bool
	}{
		{nil, false},
		{true, true},
		{false, false},
		{"hello", true},
		{"", false},
		{float64(1), true},
		{float64(0), false},
		{42, true},
		{0, false},
		{map[string]interface{}{"k": "v"}, true},
		{map[string]interface{}{}, false},
		{[]interface{}{"a"}, true},
		{[]interface{}{}, false},
		{struct{}{}, true}, // unknown type → true
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, isTruthy(tt.val), "isTruthy(%v)", tt.val)
	}
}

func TestIsEqual(t *testing.T) {
	assert.True(t, isEqual(nil, nil))
	assert.False(t, isEqual(nil, "x"))
	assert.False(t, isEqual("x", nil))
	assert.True(t, isEqual("hello", "hello"))
	assert.False(t, isEqual("a", "b"))
	assert.True(t, isEqual(float64(42), float64(42)))
	assert.True(t, isEqual(true, true))
	assert.False(t, isEqual(true, false))

	// Map comparison uses reflect.DeepEqual
	m1 := map[string]interface{}{"a": "1", "b": "2"}
	m2 := map[string]interface{}{"a": "1", "b": "2"}
	m3 := map[string]interface{}{"a": "1", "b": "3"}
	assert.True(t, isEqual(m1, m2))
	assert.False(t, isEqual(m1, m3))
}

func TestToString(t *testing.T) {
	assert.Equal(t, "", toString(nil))
	assert.Equal(t, "hello", toString("hello"))
	assert.Equal(t, "true", toString(true))
	assert.Equal(t, "false", toString(false))
	assert.Equal(t, "42", toString(float64(42)))
	assert.Equal(t, "3.14", toString(float64(3.14)))
	assert.Equal(t, "7", toString(7))
}

func TestInterpolate(t *testing.T) {
	ctx := NewContext(map[string]interface{}{
		"host":    "localhost",
		"port":    "8080",
		"nested":  map[string]interface{}{"key": "val"},
		"missing": nil,
	})

	// Basic interpolation
	assert.Equal(t, "http://localhost:8080", interpolate(ctx, "http://${host}:${port}"))

	// Nested path
	assert.Equal(t, "val", interpolate(ctx, "${nested.key}"))

	// Missing variable — preserved as-is
	assert.Equal(t, "${undefined}", interpolate(ctx, "${undefined}"))

	// No interpolation markers — short circuit
	assert.Equal(t, "plain string", interpolate(ctx, "plain string"))

	// Empty variable path
	assert.Equal(t, "${}", interpolate(ctx, "${}"))
}

func TestResolveVarPath(t *testing.T) {
	ctx := NewContext(map[string]interface{}{
		"config": map[string]interface{}{
			"model": "opus",
			"nested": map[string]interface{}{
				"deep": "value",
			},
		},
	})

	assert.Equal(t, "opus", resolveVarPath(ctx, "config.model"))
	assert.Equal(t, "value", resolveVarPath(ctx, "config.nested.deep"))
	assert.Nil(t, resolveVarPath(ctx, "undefined"))
	assert.Nil(t, resolveVarPath(ctx, "config.nonexistent"))

	// Single segment
	val := resolveVarPath(ctx, "config")
	assert.NotNil(t, val)
}
