package eval

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"unicode"
)

// RegisterBuiltins registers all built-in functions into the context.
func RegisterBuiltins(ctx *Context) {
	ctx.Builtins["json"] = builtinJSON
	ctx.Builtins["json_parse"] = builtinJSONParse
	ctx.Builtins["json_merge"] = builtinJSONMerge
	ctx.Builtins["mcp_transform"] = builtinMCPTransform
	ctx.Builtins["codex_mcp_toml"] = builtinCodexMCPTOML
	ctx.Builtins["str_replace"] = builtinStrReplace
	ctx.Builtins["str_contains"] = builtinStrContains
	ctx.Builtins["str_join"] = builtinStrJoin
	ctx.Builtins["len"] = builtinLen
	ctx.Builtins["print"] = builtinPrint
}

// json(obj) — serialize a map/value to JSON string
func builtinJSON(args ...interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("json: expected 1 argument, got %d", len(args))
	}
	b, err := json.Marshal(args[0])
	if err != nil {
		return nil, fmt.Errorf("json: %w", err)
	}
	return string(b), nil
}

// json_parse(str) — parse JSON string into a map
func builtinJSONParse(args ...interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("json_parse: expected 1 argument, got %d", len(args))
	}
	s := toString(args[0])
	var result interface{}
	if err := json.Unmarshal([]byte(s), &result); err != nil {
		return nil, fmt.Errorf("json_parse: %w", err)
	}
	return result, nil
}

// json_merge(a, b, ...) — shallow merge multiple maps (later keys override earlier).
// Intentionally shallow: nested objects like MCP server configs are replaced whole,
// preserving agent-specific fields that differ between formats.
func builtinJSONMerge(args ...interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("json_merge: expected at least 2 arguments")
	}
	result := make(map[string]interface{})
	for _, arg := range args {
		m, ok := arg.(map[string]interface{})
		if !ok {
			continue
		}
		for k, v := range m {
			result[k] = v
		}
	}
	return result, nil
}

// mcp_transform(config, format) — transform MCP server config to agent format.
// Handles differences like "url" vs "httpUrl" (Gemini), "enabled" field (OpenCode).
func builtinMCPTransform(args ...interface{}) (interface{}, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("mcp_transform: expected 2 arguments")
	}
	servers, ok := args[0].(map[string]interface{})
	if !ok {
		return args[0], nil
	}
	format := toString(args[1])
	result := make(map[string]interface{})

	for name, srv := range servers {
		srvMap, ok := srv.(map[string]interface{})
		if !ok {
			result[name] = srv
			continue
		}
		result[name] = transformMCPServer(srvMap, format)
	}
	return result, nil
}

func transformMCPServer(srv map[string]interface{}, format string) map[string]interface{} {
	out := make(map[string]interface{})
	for k, v := range srv {
		out[k] = v
	}

	switch format {
	case "gemini":
		// Gemini uses "httpUrl" instead of "url"
		if url, ok := out["url"]; ok {
			out["httpUrl"] = url
			delete(out, "url")
		}
		delete(out, "type")
	case "opencode":
		// OpenCode requires type="local" + command=[...] format.
		// Convert HTTP MCP servers to streamable-http proxy via curl.
		if url, ok := out["url"].(string); ok {
			out["type"] = "local"
			// Use npx to run streamable-http proxy, or fall back to direct URL
			out["command"] = []interface{}{"npx", "-y", "mcp-remote", url}
			delete(out, "url")
			delete(out, "headers")
		}
		out["enabled"] = true
	case "codex":
		// Codex uses flat format, no transformation needed here
		// (Codex MCP is handled via -c args, not file)
	}
	return out
}

// codex_mcp_toml(config) serializes MCP server config to Codex Rust config.toml.
func builtinCodexMCPTOML(args ...interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("codex_mcp_toml: expected 1 argument, got %d", len(args))
	}
	servers, ok := args[0].(map[string]interface{})
	if !ok {
		return "", nil
	}

	names := make([]string, 0, len(servers))
	for name := range servers {
		names = append(names, name)
	}
	sort.Strings(names)

	var b strings.Builder
	for _, name := range names {
		srv, ok := servers[name].(map[string]interface{})
		if !ok {
			continue
		}
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		fmt.Fprintf(&b, "[mcp_servers.%s]\n", tomlKey(name))
		writeTomlStringField(&b, "type", srv["type"])
		writeTomlStringField(&b, "url", srv["url"])
		writeTomlStringField(&b, "command", srv["command"])
		writeTomlStringArrayField(&b, "args", srv["args"])
		headers := srv["headers"]
		if headers == nil {
			headers = srv["http_headers"]
		}
		writeTomlSubTable(&b, name, "http_headers", headers)
		writeTomlSubTable(&b, name, "env", srv["env"])
	}

	return b.String(), nil
}

func writeTomlStringField(b *strings.Builder, key string, val interface{}) {
	s, ok := val.(string)
	if !ok || s == "" {
		return
	}
	fmt.Fprintf(b, "%s = %s\n", tomlKey(key), tomlString(s))
}

func writeTomlStringArrayField(b *strings.Builder, key string, val interface{}) {
	items, ok := toStringSlice(val)
	if !ok || len(items) == 0 {
		return
	}
	quoted := make([]string, 0, len(items))
	for _, item := range items {
		quoted = append(quoted, tomlString(item))
	}
	fmt.Fprintf(b, "%s = [%s]\n", tomlKey(key), strings.Join(quoted, ", "))
}

func writeTomlSubTable(b *strings.Builder, serverName, tableName string, val interface{}) {
	fields, ok := toStringMap(val)
	if !ok || len(fields) == 0 {
		return
	}
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	fmt.Fprintf(b, "\n[mcp_servers.%s.%s]\n", tomlKey(serverName), tomlKey(tableName))
	for _, key := range keys {
		if fields[key] == "" {
			continue
		}
		fmt.Fprintf(b, "%s = %s\n", tomlKey(key), tomlString(fields[key]))
	}
}

func toStringSlice(val interface{}) ([]string, bool) {
	switch v := val.(type) {
	case []string:
		return v, true
	case []interface{}:
		items := make([]string, 0, len(v))
		for _, item := range v {
			items = append(items, toString(item))
		}
		return items, true
	default:
		return nil, false
	}
}

func toStringMap(val interface{}) (map[string]string, bool) {
	switch v := val.(type) {
	case map[string]string:
		return v, true
	case map[string]interface{}:
		result := make(map[string]string, len(v))
		for key, item := range v {
			result[key] = toString(item)
		}
		return result, true
	default:
		return nil, false
	}
}

func tomlString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\b':
			b.WriteString(`\b`)
		case '\t':
			b.WriteString(`\t`)
		case '\n':
			b.WriteString(`\n`)
		case '\f':
			b.WriteString(`\f`)
		case '\r':
			b.WriteString(`\r`)
		default:
			if r < 0x20 {
				fmt.Fprintf(&b, `\u%04X`, r)
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}

func tomlKey(s string) string {
	if s == "" {
		return `""`
	}
	for i, r := range s {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-') {
			return tomlString(s)
		}
		if i == 0 && unicode.IsDigit(r) {
			return tomlString(s)
		}
	}
	return s
}

func builtinStrReplace(args ...interface{}) (interface{}, error) {
	if len(args) != 3 {
		return nil, fmt.Errorf("str_replace: expected 3 arguments")
	}
	return strings.ReplaceAll(toString(args[0]), toString(args[1]), toString(args[2])), nil
}

func builtinStrContains(args ...interface{}) (interface{}, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("str_contains: expected 2 arguments")
	}
	return strings.Contains(toString(args[0]), toString(args[1])), nil
}

func builtinStrJoin(args ...interface{}) (interface{}, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("str_join: expected 2 arguments (list, separator)")
	}
	sep := toString(args[1])
	switch v := args[0].(type) {
	case []interface{}:
		parts := make([]string, len(v))
		for i, item := range v {
			parts[i] = toString(item)
		}
		return strings.Join(parts, sep), nil
	case map[string]interface{}:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		return strings.Join(keys, sep), nil
	default:
		return "", fmt.Errorf("str_join: first argument must be a list or map, got %T", args[0])
	}
}

func builtinLen(args ...interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("len: expected 1 argument")
	}
	switch v := args[0].(type) {
	case string:
		return float64(len(v)), nil
	case map[string]interface{}:
		return float64(len(v)), nil
	case []interface{}:
		return float64(len(v)), nil
	case nil:
		return float64(0), nil
	default:
		return float64(0), nil
	}
}

func builtinPrint(args ...interface{}) (interface{}, error) {
	parts := make([]string, len(args))
	for i, a := range args {
		parts[i] = toString(a)
	}
	// In Runner mode this would write to the pod build log.
	// For now just a no-op that returns the joined string.
	return strings.Join(parts, " "), nil
}
