package extension

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// parseFrontmatter
// =============================================================================

func TestParseFrontmatter_NoFrontmatter(t *testing.T) {
	content := "# My Skill\n\nSome description."
	fm := parseFrontmatter(content)
	assert.Empty(t, fm)
}

func TestParseFrontmatter_ValidKeyValuePairs(t *testing.T) {
	content := `---
name: my-skill
description: A useful skill
license: MIT
---
# Content below`
	fm := parseFrontmatter(content)
	assert.Equal(t, "my-skill", fm["name"])
	assert.Equal(t, "A useful skill", fm["description"])
	assert.Equal(t, "MIT", fm["license"])
}

func TestParseFrontmatter_QuotedValues(t *testing.T) {
	content := `---
name: "my-skill"
description: 'A useful skill'
---`
	fm := parseFrontmatter(content)
	assert.Equal(t, "my-skill", fm["name"])
	assert.Equal(t, "A useful skill", fm["description"])
}

func TestParseFrontmatter_ColonsInValue(t *testing.T) {
	content := `---
url: https://example.com
name: my-skill
---`
	fm := parseFrontmatter(content)
	assert.Equal(t, "https://example.com", fm["url"])
	assert.Equal(t, "my-skill", fm["name"])
}

func TestParseFrontmatter_EmptyFrontmatter(t *testing.T) {
	content := "---\n---\n# Content"
	fm := parseFrontmatter(content)
	assert.Empty(t, fm)
}

func TestParseFrontmatter_ExtraWhitespace(t *testing.T) {
	content := `---
  name:   my-skill
  description:   A useful skill
---`
	fm := parseFrontmatter(content)
	assert.Equal(t, "my-skill", fm["name"])
	assert.Equal(t, "A useful skill", fm["description"])
}

func TestParseFrontmatter_NoClosingDelimiter(t *testing.T) {
	// The parser should still extract everything before EOF
	content := `---
name: my-skill
description: no closing`
	fm := parseFrontmatter(content)
	assert.Equal(t, "my-skill", fm["name"])
	assert.Equal(t, "no closing", fm["description"])
}

func TestParseFrontmatter_OnlyOneLine(t *testing.T) {
	content := "---"
	fm := parseFrontmatter(content)
	assert.Empty(t, fm)
}

// =============================================================================
// detectRepoType
// =============================================================================

func TestDetectRepoType_Single(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: test\n---"), 0644))
	assert.Equal(t, "single", detectRepoType(dir))
}

func TestDetectRepoType_Collection(t *testing.T) {
	dir := t.TempDir()
	// No SKILL.md at root
	assert.Equal(t, "collection", detectRepoType(dir))
}

// =============================================================================
// shouldIgnoreDir
// =============================================================================

func TestShouldIgnoreDir(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{".git", ".git", true},
		{".github", ".github", true},
		{".vscode", ".vscode", true},
		{"spec", "spec", true},
		{"template", "template", true},
		{".claude-plugin", ".claude-plugin", true},
		{"node_modules", "node_modules", true},
		{"__pycache__", "__pycache__", true},
		{"dot-prefixed hidden dir", ".hidden", true},
		{"normal skill dir", "my-skill", false},
		{"another normal dir", "pdf-processing", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, shouldIgnoreDir(tt.input))
		})
	}
}

// =============================================================================
// parseSkillDir
// =============================================================================

func TestParseSkillDir_ValidFrontmatter(t *testing.T) {
	dir := t.TempDir()
	content := `---
name: my-awesome-skill
description: Does awesome things
license: MIT
compatibility: claude-code
allowed-tools: Read,Write,Bash
---
# My Awesome Skill

Detailed docs here.
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0644))

	info, err := parseSkillDir(dir)
	require.NoError(t, err)
	assert.Equal(t, "my-awesome-skill", info.Slug)
	assert.Equal(t, "my-awesome-skill", info.DisplayName)
	assert.Equal(t, "Does awesome things", info.Description)
	assert.Equal(t, "MIT", info.License)
	assert.Equal(t, "claude-code", info.Compatibility)
	assert.Equal(t, "Read,Write,Bash", info.AllowedTools)
	assert.Equal(t, dir, info.DirPath)
}

func TestParseSkillDir_NoNameFallsBackToDirName(t *testing.T) {
	dir := t.TempDir()
	// Create a named subdirectory for a meaningful fallback name
	skillDir := filepath.Join(dir, "fallback-skill")
	require.NoError(t, os.MkdirAll(skillDir, 0755))

	content := `---
description: No name field here
---
`
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644))

	info, err := parseSkillDir(skillDir)
	require.NoError(t, err)
	assert.Equal(t, "fallback-skill", info.Slug)
	assert.Equal(t, "", info.DisplayName)
	assert.Equal(t, "No name field here", info.Description)
}

func TestParseSkillDir_MissingSkillMD(t *testing.T) {
	dir := t.TempDir()
	// No SKILL.md file
	_, err := parseSkillDir(dir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "SKILL.md")
}

// =============================================================================
// validateGitBranch
// =============================================================================

func TestValidateGitBranch(t *testing.T) {
	tests := []struct {
		name    string
		branch  string
		wantErr bool
	}{
		{"main", "main", false},
		{"feature branch with slash", "feature/my-branch", false},
		{"release dash version", "release-1.0", false},
		{"semver tag", "v1.2.3", false},
		{"underscore", "my_branch", false},
		{"dots", "release.1.0", false},
		{"empty string is valid", "", false},
		{"space is invalid", "branch name", true},
		{"semicolon is invalid", "branch;rm", true},
		{"dollar sign is invalid", "branch$(cmd)", true},
		{"backtick is invalid", "branch`cmd`", true},
		{"pipe is invalid", "branch|cmd", true},
		{"ampersand is invalid", "branch&cmd", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateGitBranch(tt.branch)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// =============================================================================
// fileExists / dirExists edge cases
// =============================================================================

func TestFileExists_Directory(t *testing.T) {
	dir := t.TempDir()
	// A directory should not count as a "file"
	assert.False(t, fileExists(dir))
}

func TestDirExists_File(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "file.txt")
	require.NoError(t, os.WriteFile(f, []byte("data"), 0644))
	// A file should not count as a "dir"
	assert.False(t, dirExists(f))
}

func TestFileExists_Nonexistent(t *testing.T) {
	assert.False(t, fileExists("/nonexistent/path/file.txt"))
}

func TestDirExists_Nonexistent(t *testing.T) {
	assert.False(t, dirExists("/nonexistent/path"))
}
