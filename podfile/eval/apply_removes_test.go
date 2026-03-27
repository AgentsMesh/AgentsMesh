package eval

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRemoveFromSlice_SimpleFlag(t *testing.T) {
	args := []string{"--verbose", "--model", "opus"}
	result := removeFromSlice(args, []string{"--verbose"})
	assert.Equal(t, []string{"--model", "opus"}, result)
}

func TestRemoveFromSlice_FlagWithValue(t *testing.T) {
	args := []string{"--model", "sonnet", "--permission-mode", "plan"}
	result := removeFromSlice(args, []string{"--model"})
	assert.Equal(t, []string{"--permission-mode", "plan"}, result)
}

func TestRemoveFromSlice_NonExistent(t *testing.T) {
	args := []string{"--model", "opus"}
	result := removeFromSlice(args, []string{"--nonexistent"})
	assert.Equal(t, []string{"--model", "opus"}, result)
}

func TestRemoveFromSlice_MultipleRemoves(t *testing.T) {
	args := []string{"--verbose", "--model", "opus", "--debug"}
	result := removeFromSlice(args, []string{"--verbose", "--debug"})
	assert.Equal(t, []string{"--model", "opus"}, result)
}

func TestRemoveFromSlice_Empty(t *testing.T) {
	result := removeFromSlice(nil, []string{"--flag"})
	assert.Nil(t, result)
}

func TestApplyRemoves_EnvVars(t *testing.T) {
	r := &BuildResult{
		EnvVars:    map[string]string{"KEY1": "a", "KEY2": "b", "KEY3": "c"},
		RemoveEnvs: []string{"KEY2"},
	}
	ApplyRemoves(r)
	assert.Equal(t, map[string]string{"KEY1": "a", "KEY3": "c"}, r.EnvVars)
}

func TestApplyRemoves_Skills(t *testing.T) {
	r := &BuildResult{
		Skills:       []string{"am-delegate", "am-channel", "custom"},
		RemoveSkills: []string{"am-channel"},
		EnvVars:      map[string]string{},
	}
	ApplyRemoves(r)
	assert.Equal(t, []string{"am-delegate", "custom"}, r.Skills)
}

func TestApplyRemoves_Files(t *testing.T) {
	r := &BuildResult{
		FilesToCreate: []FileEntry{
			{Path: "/a/config.json", Content: "{}"},
			{Path: "/a/remove-me.json", Content: "{}"},
		},
		Dirs:        []string{"/a", "/a/remove-dir"},
		RemoveFiles: []string{"/a/remove-me.json", "/a/remove-dir"},
		EnvVars:     map[string]string{},
	}
	ApplyRemoves(r)
	assert.Len(t, r.FilesToCreate, 1)
	assert.Equal(t, "/a/config.json", r.FilesToCreate[0].Path)
	assert.Equal(t, []string{"/a"}, r.Dirs)
}

func TestApplyRemoves_NoRemoves(t *testing.T) {
	r := &BuildResult{
		LaunchArgs: []string{"--flag"},
		EnvVars:    map[string]string{"K": "v"},
		Skills:     []string{"s1"},
	}
	ApplyRemoves(r)
	assert.Equal(t, []string{"--flag"}, r.LaunchArgs)
	assert.Equal(t, "v", r.EnvVars["K"])
	assert.Equal(t, []string{"s1"}, r.Skills)
}
