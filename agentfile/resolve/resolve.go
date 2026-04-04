// Package resolve injects final resolved CONFIG values into an AgentFile AST.
// This is the bridge between the 3-layer config merge (defaults → user prefs → overrides)
// and the AgentFile SSOT: resolved values are written back into ConfigDecl.Default
// so Runner eval reads them directly from the AgentFile.
package resolve

import "github.com/anthropics/agentsmesh/agentfile/parser"

// ExtractConfigNames returns the set of CONFIG declaration names present in a Program.
// Used to identify which CONFIG fields were explicitly set by a user's AgentFile Layer.
func ExtractConfigNames(prog *parser.Program) map[string]bool {
	names := make(map[string]bool)
	for _, d := range prog.Declarations {
		if cfg, ok := d.(*parser.ConfigDecl); ok {
			names[cfg.Name] = true
		}
	}
	return names
}

// ResolveConfigValues injects final resolved values into ConfigDecl.Default in-place.
//
// Priority (highest to lowest):
//  1. systemOverrides — system-injected values (session_id, permission_mode, resume_*)
//  2. layerConfigNames — values explicitly set in user's AgentFile Layer (already in merged AST)
//  3. userPrefs — user's personal agent preferences from DB
//  4. base default — original CONFIG declaration default (already in merged AST)
//
// For systemOverrides keys with no existing CONFIG declaration, a new ConfigDecl is appended.
func ResolveConfigValues(
	prog *parser.Program,
	layerConfigNames map[string]bool,
	userPrefs map[string]interface{},
	systemOverrides map[string]interface{},
) {
	seen := make(map[string]bool)

	for _, d := range prog.Declarations {
		cfg, ok := d.(*parser.ConfigDecl)
		if !ok {
			continue
		}
		seen[cfg.Name] = true

		// Priority 1: system overrides (highest)
		if v, ok := systemOverrides[cfg.Name]; ok {
			cfg.Default = v
			continue
		}
		// Priority 2: layer override — keep current Default (already set by merge)
		if layerConfigNames[cfg.Name] {
			continue
		}
		// Priority 3: user preferences
		if v, ok := userPrefs[cfg.Name]; ok {
			cfg.Default = v
			continue
		}
		// Priority 4: keep base default (no change needed)
	}

	// Append CONFIG declarations for systemOverrides not already in AgentFile.
	// This ensures Runner eval can read system-injected values like session_id.
	for name, value := range systemOverrides {
		if seen[name] {
			continue
		}
		prog.Declarations = append(prog.Declarations, &parser.ConfigDecl{
			Name:     name,
			TypeName: inferTypeName(value),
			Default:  value,
		})
	}
}

func inferTypeName(v interface{}) string {
	switch v.(type) {
	case bool:
		return "boolean"
	case float64, int, int64:
		return "number"
	default:
		return "string"
	}
}
