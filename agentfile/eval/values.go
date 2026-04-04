package eval

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// interpolatePattern matches ${var.path} for string interpolation.
var interpolatePattern = regexp.MustCompile(`\$\{([a-zA-Z_][a-zA-Z0-9_.]*)\}`)

// isTruthy determines the truthiness of an AgentFile value.
func isTruthy(val interface{}) bool {
	if val == nil {
		return false
	}
	switch v := val.(type) {
	case bool:
		return v
	case string:
		return v != ""
	case float64:
		return v != 0
	case int:
		return v != 0
	case map[string]interface{}:
		return len(v) > 0
	case []interface{}:
		return len(v) > 0
	default:
		return true
	}
}

// isEqual compares two AgentFile values.
func isEqual(a, b interface{}) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	// Use DeepEqual for maps and slices (fmt.Sprintf is order-dependent for maps)
	return reflect.DeepEqual(a, b)
}

// toString converts an AgentFile value to string.
func toString(val interface{}) string {
	if val == nil {
		return ""
	}
	switch v := val.(type) {
	case string:
		return v
	case bool:
		if v {
			return "true"
		}
		return "false"
	case float64:
		if v == float64(int64(v)) {
			return strconv.FormatInt(int64(v), 10)
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	case int:
		return strconv.Itoa(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// interpolate replaces ${var.path} references in a string with context values.
// E.g., "http://127.0.0.1:${mcp.port}/mcp" → "http://127.0.0.1:19000/mcp"
func interpolate(ctx *Context, s string) string {
	if !strings.Contains(s, "${") {
		return s
	}
	return interpolatePattern.ReplaceAllStringFunc(s, func(match string) string {
		sub := interpolatePattern.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		val := resolveVarPath(ctx, sub[1])
		if val == nil {
			return match // leave unreplaced if not found
		}
		return toString(val)
	})
}

// resolveVarPath resolves a dot-separated variable path like "mcp.port".
func resolveVarPath(ctx *Context, path string) interface{} {
	parts := strings.SplitN(path, ".", 2)
	root, ok := ctx.Get(parts[0])
	if !ok {
		return nil
	}
	if len(parts) == 1 {
		return root
	}
	remaining := parts[1]
	current := root
	for remaining != "" {
		parts = strings.SplitN(remaining, ".", 2)
		nested, found := GetNested(current, parts[0])
		if !found {
			return nil
		}
		current = nested
		if len(parts) == 1 {
			break
		}
		remaining = parts[1]
	}
	return current
}
