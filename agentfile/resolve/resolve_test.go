package resolve

import (
	"testing"

	"github.com/anthropics/agentsmesh/agentfile/parser"
	"github.com/stretchr/testify/assert"
)

func cfg(name, typeName string, def interface{}) *parser.ConfigDecl {
	return &parser.ConfigDecl{Name: name, TypeName: typeName, Default: def}
}

func prog(decls ...parser.Declaration) *parser.Program {
	return &parser.Program{Declarations: decls}
}

func TestExtractConfigNames(t *testing.T) {
	p := prog(
		cfg("model", "string", "sonnet"),
		&parser.AgentDecl{Command: "claude"},
		cfg("mcp_enabled", "boolean", true),
	)
	names := ExtractConfigNames(p)
	assert.True(t, names["model"])
	assert.True(t, names["mcp_enabled"])
	assert.False(t, names["agent"])
	assert.Len(t, names, 2)
}

func TestExtractConfigNames_Empty(t *testing.T) {
	names := ExtractConfigNames(prog())
	assert.Empty(t, names)
}

func TestResolveConfigValues_SystemOverrideWins(t *testing.T) {
	p := prog(cfg("permission_mode", "string", "plan"))
	ResolveConfigValues(p, nil, nil, map[string]interface{}{"permission_mode": "bypassPermissions"})
	assert.Equal(t, "bypassPermissions", p.Declarations[0].(*parser.ConfigDecl).Default)
}

func TestResolveConfigValues_LayerOverrideBeatsUserPrefs(t *testing.T) {
	p := prog(cfg("model", "select", "opus")) // Layer set to "opus" via merge
	layer := map[string]bool{"model": true}
	prefs := map[string]interface{}{"model": "haiku"}
	ResolveConfigValues(p, layer, prefs, nil)
	assert.Equal(t, "opus", p.Declarations[0].(*parser.ConfigDecl).Default)
}

func TestResolveConfigValues_UserPrefsBeatsBaseDefault(t *testing.T) {
	p := prog(cfg("model", "select", "sonnet")) // base default
	prefs := map[string]interface{}{"model": "haiku"}
	ResolveConfigValues(p, nil, prefs, nil)
	assert.Equal(t, "haiku", p.Declarations[0].(*parser.ConfigDecl).Default)
}

func TestResolveConfigValues_BaseDefaultPreserved(t *testing.T) {
	p := prog(cfg("model", "select", "sonnet"))
	ResolveConfigValues(p, nil, nil, nil)
	assert.Equal(t, "sonnet", p.Declarations[0].(*parser.ConfigDecl).Default)
}

func TestResolveConfigValues_FullPriorityChain(t *testing.T) {
	p := prog(
		cfg("a", "string", "base_a"),     // system override
		cfg("b", "string", "layer_b"),    // layer override
		cfg("c", "string", "base_c"),     // user pref
		cfg("d", "string", "base_d"),     // no override
	)
	layer := map[string]bool{"b": true}
	prefs := map[string]interface{}{"c": "pref_c", "b": "pref_b"}
	system := map[string]interface{}{"a": "sys_a"}

	ResolveConfigValues(p, layer, prefs, system)

	assert.Equal(t, "sys_a", p.Declarations[0].(*parser.ConfigDecl).Default)    // system wins
	assert.Equal(t, "layer_b", p.Declarations[1].(*parser.ConfigDecl).Default)  // layer wins over pref
	assert.Equal(t, "pref_c", p.Declarations[2].(*parser.ConfigDecl).Default)   // pref wins over base
	assert.Equal(t, "base_d", p.Declarations[3].(*parser.ConfigDecl).Default)   // base preserved
}

func TestResolveConfigValues_SystemOverrideAppendsNewDecl(t *testing.T) {
	p := prog(cfg("model", "string", "sonnet"))
	system := map[string]interface{}{"session_id": "abc-123"}
	ResolveConfigValues(p, nil, nil, system)

	assert.Len(t, p.Declarations, 2)
	newCfg := p.Declarations[1].(*parser.ConfigDecl)
	assert.Equal(t, "session_id", newCfg.Name)
	assert.Equal(t, "abc-123", newCfg.Default)
	assert.Equal(t, "string", newCfg.TypeName)
}

func TestResolveConfigValues_SystemOverrideBoolType(t *testing.T) {
	p := prog()
	system := map[string]interface{}{"resume_enabled": true}
	ResolveConfigValues(p, nil, nil, system)

	newCfg := p.Declarations[0].(*parser.ConfigDecl)
	assert.Equal(t, "boolean", newCfg.TypeName)
	assert.Equal(t, true, newCfg.Default)
}

func TestResolveConfigValues_PreservesNonConfigDecls(t *testing.T) {
	agent := &parser.AgentDecl{Command: "claude"}
	p := prog(agent, cfg("model", "string", "sonnet"))
	prefs := map[string]interface{}{"model": "opus"}
	ResolveConfigValues(p, nil, prefs, nil)

	assert.Equal(t, agent, p.Declarations[0]) // untouched
	assert.Equal(t, "opus", p.Declarations[1].(*parser.ConfigDecl).Default)
}

func TestResolveConfigValues_NilInputs(t *testing.T) {
	p := prog(cfg("model", "string", "sonnet"))
	ResolveConfigValues(p, nil, nil, nil)
	assert.Equal(t, "sonnet", p.Declarations[0].(*parser.ConfigDecl).Default)
}
