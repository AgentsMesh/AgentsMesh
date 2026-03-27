package extension

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// scanCollectionSkills
// =============================================================================

func TestScanCollectionSkills_SkillsSubdir(t *testing.T) {
	root := t.TempDir()
	skillsDir := filepath.Join(root, "skills")
	require.NoError(t, os.MkdirAll(filepath.Join(skillsDir, "skill-a"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(skillsDir, "skill-b"), 0755))

	require.NoError(t, os.WriteFile(filepath.Join(skillsDir, "skill-a", "SKILL.md"),
		[]byte("---\nname: skill-a\ndescription: First skill\n---"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(skillsDir, "skill-b", "SKILL.md"),
		[]byte("---\nname: skill-b\ndescription: Second skill\n---"), 0644))

	skills, err := scanCollectionSkills(root)
	require.NoError(t, err)
	assert.Len(t, skills, 2)

	slugs := map[string]bool{}
	for _, s := range skills {
		slugs[s.Slug] = true
	}
	assert.True(t, slugs["skill-a"])
	assert.True(t, slugs["skill-b"])
}

func TestScanCollectionSkills_RootLevelSubdirs(t *testing.T) {
	root := t.TempDir()
	// No skills/ directory; skills are at root level
	require.NoError(t, os.MkdirAll(filepath.Join(root, "alpha"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "beta"), 0755))

	require.NoError(t, os.WriteFile(filepath.Join(root, "alpha", "SKILL.md"),
		[]byte("---\nname: alpha\n---"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "beta", "SKILL.md"),
		[]byte("---\nname: beta\n---"), 0644))

	skills, err := scanCollectionSkills(root)
	require.NoError(t, err)
	assert.Len(t, skills, 2)
}

func TestScanCollectionSkills_EmptySkillsDirFallsToRoot(t *testing.T) {
	root := t.TempDir()
	// skills/ directory exists but has no valid skills inside
	require.NoError(t, os.MkdirAll(filepath.Join(root, "skills"), 0755))
	// Root-level skill
	require.NoError(t, os.MkdirAll(filepath.Join(root, "my-skill"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "my-skill", "SKILL.md"),
		[]byte("---\nname: my-skill\n---"), 0644))

	skills, err := scanCollectionSkills(root)
	require.NoError(t, err)
	assert.Len(t, skills, 1)
	assert.Equal(t, "my-skill", skills[0].Slug)
}

func TestScanCollectionSkills_IgnoresSpecialDirs(t *testing.T) {
	root := t.TempDir()
	// Create ignored directories with SKILL.md files
	for _, ignoredDir := range []string{".git", "node_modules", "__pycache__"} {
		dirPath := filepath.Join(root, ignoredDir)
		require.NoError(t, os.MkdirAll(dirPath, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(dirPath, "SKILL.md"),
			[]byte("---\nname: should-not-find\n---"), 0644))
	}
	// One valid skill
	require.NoError(t, os.MkdirAll(filepath.Join(root, "valid-skill"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "valid-skill", "SKILL.md"),
		[]byte("---\nname: valid-skill\n---"), 0644))

	skills, err := scanCollectionSkills(root)
	require.NoError(t, err)
	assert.Len(t, skills, 1)
	assert.Equal(t, "valid-skill", skills[0].Slug)
}

func TestScanCollectionSkills_IgnoresNonDirectories(t *testing.T) {
	root := t.TempDir()
	// A regular file at root level (not a directory)
	require.NoError(t, os.WriteFile(filepath.Join(root, "README.md"), []byte("readme"), 0644))
	// One valid skill
	require.NoError(t, os.MkdirAll(filepath.Join(root, "real-skill"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "real-skill", "SKILL.md"),
		[]byte("---\nname: real-skill\n---"), 0644))

	skills, err := scanCollectionSkills(root)
	require.NoError(t, err)
	assert.Len(t, skills, 1)
	assert.Equal(t, "real-skill", skills[0].Slug)
}

func TestScanCollectionSkills_ReadDirError(t *testing.T) {
	_, err := scanCollectionSkills("/nonexistent/dir/that/does/not/exist")
	assert.Error(t, err)
}

func TestScanCollectionSkills_UnreadableSkillsDir(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping test when running as root")
	}
	root := t.TempDir()
	skillsDir := filepath.Join(root, "skills")
	require.NoError(t, os.MkdirAll(skillsDir, 0755))

	// Make skills/ dir unreadable
	require.NoError(t, os.Chmod(skillsDir, 0000))
	defer os.Chmod(skillsDir, 0755) // Restore for cleanup

	// Even though skills/ exists and is unreadable, should fall through to root-level scan
	// Root has no skills, so should return empty
	skills, err := scanCollectionSkills(root)
	require.NoError(t, err)
	assert.Len(t, skills, 0)
}

func TestScanCollectionSkills_SkillsDirInvalidSkillMD(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping test when running as root")
	}
	root := t.TempDir()
	skillsDir := filepath.Join(root, "skills")
	require.NoError(t, os.MkdirAll(filepath.Join(skillsDir, "bad-skill"), 0755))

	// Write an unreadable SKILL.md (make the file 0000 permissions)
	skillMdPath := filepath.Join(skillsDir, "bad-skill", "SKILL.md")
	require.NoError(t, os.WriteFile(skillMdPath, []byte("---\nname: bad\n---"), 0644))
	require.NoError(t, os.Chmod(skillMdPath, 0000))
	defer os.Chmod(skillMdPath, 0644)

	// Should skip the bad skill and return empty (falls to root-level)
	skills, err := scanCollectionSkills(root)
	require.NoError(t, err)
	assert.Len(t, skills, 0)
}

func TestScanCollectionSkills_RootLevelParseFailure(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping test when running as root")
	}
	root := t.TempDir()
	// Create a root-level subdirectory with an unreadable SKILL.md
	badSkillDir := filepath.Join(root, "bad-skill")
	require.NoError(t, os.MkdirAll(badSkillDir, 0755))
	skillMdPath := filepath.Join(badSkillDir, "SKILL.md")
	require.NoError(t, os.WriteFile(skillMdPath, []byte("---\nname: bad\n---"), 0644))
	require.NoError(t, os.Chmod(skillMdPath, 0000))
	defer os.Chmod(skillMdPath, 0644)

	// Create a valid skill alongside it
	goodSkillDir := filepath.Join(root, "good-skill")
	require.NoError(t, os.MkdirAll(goodSkillDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(goodSkillDir, "SKILL.md"),
		[]byte("---\nname: good-skill\n---"), 0644))

	skills, err := scanCollectionSkills(root)
	require.NoError(t, err)
	// The bad skill should be skipped, only good skill returned
	assert.Len(t, skills, 1)
	assert.Equal(t, "good-skill", skills[0].Slug)
}

func TestScanCollectionSkills_SkillsDirIgnoresNonDirs(t *testing.T) {
	root := t.TempDir()
	skillsDir := filepath.Join(root, "skills")
	require.NoError(t, os.MkdirAll(skillsDir, 0755))

	// Create a regular file (not a directory) inside skills/
	require.NoError(t, os.WriteFile(filepath.Join(skillsDir, "README.md"), []byte("readme"), 0644))

	// Create a valid skill
	require.NoError(t, os.MkdirAll(filepath.Join(skillsDir, "valid"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(skillsDir, "valid", "SKILL.md"),
		[]byte("---\nname: valid\n---"), 0644))

	skills, err := scanCollectionSkills(root)
	require.NoError(t, err)
	assert.Len(t, skills, 1)
	assert.Equal(t, "valid", skills[0].Slug)
}

func TestScanCollectionSkills_RootDirUnreadable(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping test when running as root")
	}
	root := t.TempDir()
	// No skills/ directory (skip priority 1) and make root unreadable
	// to trigger os.ReadDir error at line 283-285
	require.NoError(t, os.Chmod(root, 0000))
	defer os.Chmod(root, 0755)

	_, err := scanCollectionSkills(root)
	assert.Error(t, err, "should fail when root dir is unreadable")
}
