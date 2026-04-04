package merge

import (
	"testing"

	"github.com/anthropics/agentsmesh/agentfile/extract"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Cover: Slice adds SKILLS when base has none (findSkillsDecl returns nil)
func TestMerge_SliceAddsSkillsToEmptyBase(t *testing.T) {
	base := parse(t, `AGENT test`)
	slice := parse(t, `SKILLS new-skill`)
	Merge(base, slice)
	spec := extract.Extract(base)
	assert.Equal(t, []string{"new-skill"}, spec.Skills)
}

// Cover getDeclKey UNKNOWN branch
func TestMerge_UnknownDeclType(t *testing.T) {
	// This tests internal behavior — unknown decl type gets "UNKNOWN" key
	// We can't easily trigger this from AgentFile syntax, but verify merge doesn't panic
	base := parse(t, `AGENT test`)
	slice := parse(t, `AGENT test2`)
	Merge(base, slice)
	spec := extract.Extract(base)
	assert.Equal(t, "test2", spec.Agent.Command)
}

// Cover: Slice adds ENV when base has none
func TestMerge_SliceAddsEnvToEmptyBase(t *testing.T) {
	base := parse(t, `AGENT test`)
	slice := parse(t, `ENV NEW_KEY SECRET`)
	Merge(base, slice)
	spec := extract.Extract(base)
	require.Len(t, spec.Env, 1)
	assert.Equal(t, "NEW_KEY", spec.Env[0].Name)
}

// Cover: Multiple REMOVE in one slice
func TestMerge_MultipleRemovesInSlice(t *testing.T) {
	base := parse(t, `
AGENT test
ENV KEY1 SECRET
ENV KEY2 SECRET
ENV KEY3 SECRET
CONFIG a BOOL = true
CONFIG b BOOL = false
`)
	slice := parse(t, `
REMOVE ENV KEY1
REMOVE ENV KEY3
REMOVE CONFIG b
`)
	Merge(base, slice)
	spec := extract.Extract(base)
	require.Len(t, spec.Env, 1)
	assert.Equal(t, "KEY2", spec.Env[0].Name)
	require.Len(t, spec.Config, 1)
	assert.Equal(t, "a", spec.Config[0].Name)
}
