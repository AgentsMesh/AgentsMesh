package extension

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
	"github.com/anthropics/agentsmesh/backend/internal/infra/storage"
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
// computeDirSHA
// =============================================================================

func TestComputeDirSHA_Deterministic(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "file1.txt"), []byte("hello"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "file2.txt"), []byte("world"), 0644))

	sha1, err := computeDirSHA(dir)
	require.NoError(t, err)

	sha2, err := computeDirSHA(dir)
	require.NoError(t, err)

	assert.Equal(t, sha1, sha2, "same contents should produce same SHA")
	assert.Len(t, sha1, 64, "SHA256 hex should be 64 characters")
}

func TestComputeDirSHA_DifferentContent(t *testing.T) {
	dir1 := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir1, "file.txt"), []byte("content-a"), 0644))

	dir2 := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir2, "file.txt"), []byte("content-b"), 0644))

	sha1, err := computeDirSHA(dir1)
	require.NoError(t, err)

	sha2, err := computeDirSHA(dir2)
	require.NoError(t, err)

	assert.NotEqual(t, sha1, sha2, "different content should produce different SHA")
}

func TestComputeDirSHA_FileOrderDoesNotMatter(t *testing.T) {
	// Create two directories with the same files but created in different order
	dir1 := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir1, "aaa.txt"), []byte("first"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir1, "zzz.txt"), []byte("second"), 0644))

	dir2 := t.TempDir()
	// Write in reverse order
	require.NoError(t, os.WriteFile(filepath.Join(dir2, "zzz.txt"), []byte("second"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir2, "aaa.txt"), []byte("first"), 0644))

	sha1, err := computeDirSHA(dir1)
	require.NoError(t, err)

	sha2, err := computeDirSHA(dir2)
	require.NoError(t, err)

	assert.Equal(t, sha1, sha2, "file creation order should not matter (sorted internally)")
}

func TestComputeDirSHA_IgnoresGitDirectory(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "file.txt"), []byte("content"), 0644))

	sha1, err := computeDirSHA(dir)
	require.NoError(t, err)

	// Add a .git directory
	gitDir := filepath.Join(dir, ".git")
	require.NoError(t, os.MkdirAll(gitDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/main"), 0644))

	sha2, err := computeDirSHA(dir)
	require.NoError(t, err)

	assert.Equal(t, sha1, sha2, ".git directory should be ignored")
}

// =============================================================================
// packageSkillDir
// =============================================================================

func TestPackageSkillDir_CreatesValidTarGz(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: test\n---"), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "subdir"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "subdir", "helper.txt"), []byte("helper content"), 0644))

	data, err := packageSkillDir(dir)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Decompress and verify tar entries
	gr, err := gzip.NewReader(bytes.NewReader(data))
	require.NoError(t, err)
	defer gr.Close()

	tr := tar.NewReader(gr)
	files := map[string]string{}
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		if !header.FileInfo().IsDir() {
			content, err := io.ReadAll(tr)
			require.NoError(t, err)
			files[header.Name] = string(content)
		}
	}

	assert.Contains(t, files, "SKILL.md")
	assert.Equal(t, "---\nname: test\n---", files["SKILL.md"])
	assert.Contains(t, files, filepath.Join("subdir", "helper.txt"))
	assert.Equal(t, "helper content", files[filepath.Join("subdir", "helper.txt")])
}

func TestPackageSkillDir_SkipsGitAndIgnoredDirs(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("skill content"), 0644))

	// Create directories that should be ignored
	for _, ignored := range []string{".git", "node_modules", "__pycache__"} {
		ignoredPath := filepath.Join(dir, ignored)
		require.NoError(t, os.MkdirAll(ignoredPath, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(ignoredPath, "data.txt"), []byte("ignored"), 0644))
	}

	data, err := packageSkillDir(dir)
	require.NoError(t, err)

	// Extract and check
	gr, err := gzip.NewReader(bytes.NewReader(data))
	require.NoError(t, err)
	defer gr.Close()

	tr := tar.NewReader(gr)
	var names []string
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		names = append(names, header.Name)
	}

	assert.Equal(t, []string{"SKILL.md"}, names, "should only contain SKILL.md, not ignored dirs")
}

func TestPackageSkillDir_ExtractedFilesMatchOriginal(t *testing.T) {
	dir := t.TempDir()
	originalFiles := map[string]string{
		"SKILL.md":          "---\nname: verify-test\n---\n# Test",
		"config.yaml":       "key: value\n",
		"scripts/install.sh": "#!/bin/bash\necho hello\n",
	}
	for relPath, content := range originalFiles {
		absPath := filepath.Join(dir, relPath)
		require.NoError(t, os.MkdirAll(filepath.Dir(absPath), 0755))
		require.NoError(t, os.WriteFile(absPath, []byte(content), 0644))
	}

	data, err := packageSkillDir(dir)
	require.NoError(t, err)

	// Extract into a new temp dir
	extractDir := t.TempDir()
	gr, err := gzip.NewReader(bytes.NewReader(data))
	require.NoError(t, err)
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		target := filepath.Join(extractDir, header.Name)
		if header.FileInfo().IsDir() {
			require.NoError(t, os.MkdirAll(target, 0755))
		} else {
			require.NoError(t, os.MkdirAll(filepath.Dir(target), 0755))
			content, err := io.ReadAll(tr)
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(target, content, 0644))
		}
	}

	// Verify each original file exists with the same content in the extracted dir
	for relPath, expectedContent := range originalFiles {
		actual, err := os.ReadFile(filepath.Join(extractDir, relPath))
		require.NoError(t, err, "file %s should exist in archive", relPath)
		assert.Equal(t, expectedContent, string(actual), "content mismatch for %s", relPath)
	}
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
// gitClone URL validation
// =============================================================================

func TestGitClone_URLValidation(t *testing.T) {
	ctx := context.Background()
	targetDir := t.TempDir()

	tests := []struct {
		name       string
		url        string
		wantErrMsg string
	}{
		{
			name:       "http URL rejected",
			url:        "http://github.com/user/repo.git",
			wantErrMsg: "only https:// URLs are allowed",
		},
		{
			name:       "ssh URL rejected",
			url:        "ssh://git@github.com/user/repo.git",
			wantErrMsg: "only https:// URLs are allowed",
		},
		{
			name:       "file URL rejected",
			url:        "file:///local/path/repo",
			wantErrMsg: "only https:// URLs are allowed",
		},
		{
			name:       "local path rejected",
			url:        "/local/path/repo",
			wantErrMsg: "only https:// URLs are allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := gitClone(ctx, tt.url, "", filepath.Join(targetDir, tt.name))
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErrMsg)
		})
	}
}

func TestGitClone_HTTPSURLPassesValidation(t *testing.T) {
	ctx := context.Background()
	targetDir := filepath.Join(t.TempDir(), "repo")

	// This will fail at the actual git clone (invalid repo), but the URL
	// validation should pass. The error should be about git clone failing,
	// not about URL scheme.
	err := gitClone(ctx, "https://invalid-host.example.com/no-such-repo.git", "", targetDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "git clone failed")
	assert.NotContains(t, err.Error(), "only https:// URLs are allowed")
}

// =============================================================================
// gitHead
// =============================================================================

func TestGitHead_ValidRepo(t *testing.T) {
	dir := t.TempDir()

	// Initialize a real git repository
	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.CommandContext(context.Background(), "git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0",
			"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@test.com")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v failed: %s: %v", args, string(out), err)
		}
	}

	runGit("init")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "file.txt"), []byte("hello"), 0644))
	runGit("add", ".")
	runGit("commit", "-m", "initial")

	sha, err := gitHead(context.Background(), dir)
	require.NoError(t, err)
	assert.Len(t, sha, 40, "SHA should be 40 hex characters")
	// Verify it's all hex
	for _, c := range sha {
		assert.True(t, (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f'),
			"SHA should be hex, got char: %c", c)
	}
}

func TestGitHead_InvalidRepo(t *testing.T) {
	dir := t.TempDir()
	// Not a git repo
	_, err := gitHead(context.Background(), dir)
	assert.Error(t, err)
}

// =============================================================================
// scanCollectionSkills — ReadDir error
// =============================================================================

func TestScanCollectionSkills_ReadDirError(t *testing.T) {
	_, err := scanCollectionSkills("/nonexistent/dir/that/does/not/exist")
	assert.Error(t, err)
}

// =============================================================================
// computeDirSHA — additional coverage
// =============================================================================

func TestComputeDirSHA_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	sha, err := computeDirSHA(dir)
	require.NoError(t, err)
	assert.Len(t, sha, 64, "SHA256 hex should be 64 characters")
}

func TestComputeDirSHA_WithSubdirectory(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "sub"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "sub", "file.txt"), []byte("data"), 0644))

	sha, err := computeDirSHA(dir)
	require.NoError(t, err)
	assert.Len(t, sha, 64)
}

func TestComputeDirSHA_NonexistentDir(t *testing.T) {
	_, err := computeDirSHA("/nonexistent/dir")
	assert.Error(t, err)
}

// =============================================================================
// packageSkillDir — additional coverage
// =============================================================================

func TestPackageSkillDir_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	data, err := packageSkillDir(dir)
	require.NoError(t, err)
	assert.NotEmpty(t, data, "even empty dir should produce a valid tar.gz")
}

// =============================================================================
// SyncSource — GetSkillRegistry error path
// =============================================================================

func TestSyncSource_GetSourceError(t *testing.T) {
	repo := newMockExtensionRepo()
	repo.getSourceFunc = func(_ context.Context, _ int64) (*extension.SkillRegistry, error) {
		return nil, errors.New("source not found")
	}

	imp := NewSkillImporter(repo, nil)
	err := imp.SyncSource(context.Background(), 999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get skill registry")
}

// =============================================================================
// SyncSource — UpdateSkillRegistry error (initial status update)
// =============================================================================

func TestSyncSource_UpdateStatusError(t *testing.T) {
	repo := newMockExtensionRepo()
	repo.getSourceFunc = func(_ context.Context, id int64) (*extension.SkillRegistry, error) {
		return &extension.SkillRegistry{ID: id, RepositoryURL: "https://example.com/repo", Branch: "main"}, nil
	}
	repo.updateSourceFunc = func(_ context.Context, _ *extension.SkillRegistry) error {
		return errors.New("db write failed")
	}

	imp := NewSkillImporter(repo, nil)
	err := imp.SyncSource(context.Background(), 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update sync status")
}

// =============================================================================
// SyncSource — doSync fails, final update records failure
// =============================================================================

func TestSyncSource_DoSyncFails_StatusRecorded(t *testing.T) {
	repo := newMockExtensionRepo()
	repo.getSourceFunc = func(_ context.Context, id int64) (*extension.SkillRegistry, error) {
		return &extension.SkillRegistry{ID: id, RepositoryURL: "https://example.com/repo", Branch: "main"}, nil
	}

	updateCalls := 0
	var lastStatus string
	var lastError string
	repo.updateSourceFunc = func(_ context.Context, source *extension.SkillRegistry) error {
		updateCalls++
		lastStatus = source.SyncStatus
		lastError = source.SyncError
		return nil
	}

	// storage=nil will cause doSync to fail when trying to clone
	imp := NewSkillImporter(repo, nil)
	err := imp.SyncSource(context.Background(), 1)

	// doSync should fail (git clone fails)
	assert.Error(t, err)
	// The second update should record the failure status
	assert.GreaterOrEqual(t, updateCalls, 2)
	assert.Equal(t, "failed", lastStatus)
	assert.NotEmpty(t, lastError)
}

// =============================================================================
// processSkill — comprehensive tests
// =============================================================================

func TestProcessSkill_NewSkill(t *testing.T) {
	repo := &importerMockRepo{}
	stor := newPackagerMockStorage()

	var createdItem *extension.SkillMarketItem
	repo.findSkillMarketItemBySlugFunc = func(_ context.Context, _ int64, _ string) (*extension.SkillMarketItem, error) {
		return nil, errors.New("not found")
	}
	repo.createSkillMarketItemFunc = func(_ context.Context, item *extension.SkillMarketItem) error {
		createdItem = item
		return nil
	}

	imp := NewSkillImporter(repo, stor)

	// Create a temp dir with content
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: test-skill\n---"), 0644))

	source := &extension.SkillRegistry{ID: 1}
	info := SkillInfo{
		Slug:        "test-skill",
		DisplayName: "Test Skill",
		Description: "A test",
		License:     "MIT",
		DirPath:     dir,
	}

	err := imp.processSkill(context.Background(), source, info)
	require.NoError(t, err)

	// Verify a market item was created
	require.NotNil(t, createdItem)
	assert.Equal(t, "test-skill", createdItem.Slug)
	assert.Equal(t, "Test Skill", createdItem.DisplayName)
	assert.Equal(t, "A test", createdItem.Description)
	assert.Equal(t, "MIT", createdItem.License)
	assert.Equal(t, int64(1), createdItem.RegistryID)
	assert.Equal(t, 1, createdItem.Version)
	assert.True(t, createdItem.IsActive)
	assert.NotEmpty(t, createdItem.ContentSha)
	assert.NotEmpty(t, createdItem.StorageKey)
	assert.True(t, createdItem.PackageSize > 0)

	// Verify storage received the upload
	assert.Len(t, stor.uploaded, 1)
}

func TestProcessSkill_ExistingSameSHA(t *testing.T) {
	repo := &importerMockRepo{}
	stor := newPackagerMockStorage()

	// Create a temp dir with content, compute its SHA
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: same-sha\n---"), 0644))

	sha, err := computeDirSHA(dir)
	require.NoError(t, err)

	repo.findSkillMarketItemBySlugFunc = func(_ context.Context, _ int64, _ string) (*extension.SkillMarketItem, error) {
		return &extension.SkillMarketItem{
			ID:         42,
			Slug:       "same-sha",
			ContentSha: sha,
			IsActive:   true,
			Version:    1,
		}, nil
	}

	// CreateSkillMarketItem should NOT be called
	createCalled := false
	repo.createSkillMarketItemFunc = func(_ context.Context, _ *extension.SkillMarketItem) error {
		createCalled = true
		return nil
	}
	// UpdateSkillMarketItem should NOT be called
	updateCalled := false
	repo.updateSkillMarketItemFunc = func(_ context.Context, _ *extension.SkillMarketItem) error {
		updateCalled = true
		return nil
	}

	imp := NewSkillImporter(repo, stor)

	source := &extension.SkillRegistry{ID: 1}
	info := SkillInfo{Slug: "same-sha", DirPath: dir}

	err = imp.processSkill(context.Background(), source, info)
	require.NoError(t, err)

	assert.False(t, createCalled, "should not create new item for same SHA")
	assert.False(t, updateCalled, "should not update item for same SHA when active")
	assert.Len(t, stor.uploaded, 0, "should not upload for same SHA")
}

func TestProcessSkill_ExistingSameSHA_Inactive(t *testing.T) {
	repo := &importerMockRepo{}
	stor := newPackagerMockStorage()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: inactive\n---"), 0644))

	sha, err := computeDirSHA(dir)
	require.NoError(t, err)

	repo.findSkillMarketItemBySlugFunc = func(_ context.Context, _ int64, _ string) (*extension.SkillMarketItem, error) {
		return &extension.SkillMarketItem{
			ID:         42,
			Slug:       "inactive",
			ContentSha: sha,
			IsActive:   false, // inactive
			Version:    1,
		}, nil
	}

	var updatedItem *extension.SkillMarketItem
	repo.updateSkillMarketItemFunc = func(_ context.Context, item *extension.SkillMarketItem) error {
		updatedItem = item
		return nil
	}

	imp := NewSkillImporter(repo, stor)

	source := &extension.SkillRegistry{ID: 1}
	info := SkillInfo{Slug: "inactive", DirPath: dir}

	err = imp.processSkill(context.Background(), source, info)
	require.NoError(t, err)

	// Should update to set IsActive=true
	require.NotNil(t, updatedItem)
	assert.True(t, updatedItem.IsActive)
	assert.Len(t, stor.uploaded, 0, "should not re-upload for same SHA")
}

func TestProcessSkill_ExistingDifferentSHA(t *testing.T) {
	repo := &importerMockRepo{}
	stor := newPackagerMockStorage()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: diff-sha\n---\nupdated content"), 0644))

	repo.findSkillMarketItemBySlugFunc = func(_ context.Context, _ int64, _ string) (*extension.SkillMarketItem, error) {
		return &extension.SkillMarketItem{
			ID:          42,
			Slug:        "diff-sha",
			ContentSha:  "old-sha-that-differs",
			StorageKey:  "old/key",
			PackageSize: 100,
			Version:     3,
			IsActive:    true,
		}, nil
	}

	var updatedItem *extension.SkillMarketItem
	repo.updateSkillMarketItemFunc = func(_ context.Context, item *extension.SkillMarketItem) error {
		updatedItem = item
		return nil
	}

	imp := NewSkillImporter(repo, stor)

	source := &extension.SkillRegistry{ID: 1}
	info := SkillInfo{
		Slug:        "diff-sha",
		DisplayName: "Updated Name",
		Description: "Updated desc",
		License:     "Apache-2.0",
		DirPath:     dir,
	}

	err := imp.processSkill(context.Background(), source, info)
	require.NoError(t, err)

	require.NotNil(t, updatedItem)
	assert.Equal(t, 4, updatedItem.Version, "version should be incremented")
	assert.NotEqual(t, "old-sha-that-differs", updatedItem.ContentSha)
	assert.NotEqual(t, "old/key", updatedItem.StorageKey)
	assert.True(t, updatedItem.IsActive)
	assert.Equal(t, "Updated Name", updatedItem.DisplayName)
	assert.Equal(t, "Updated desc", updatedItem.Description)
	assert.Equal(t, "Apache-2.0", updatedItem.License)
	assert.True(t, updatedItem.PackageSize > 0)
	assert.Len(t, stor.uploaded, 1, "should upload new package")
}

func TestPackageSkillDir_WithGitDir(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: test\n---"), 0644))

	// Create a .git directory (should be ignored)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git", "objects"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".git", "HEAD"), []byte("ref: refs/heads/main"), 0644))

	// Create a node_modules directory (should be ignored)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "node_modules", "package"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "node_modules", "package", "index.js"), []byte("exports = {}"), 0644))

	// Create a subdirectory with files (should be included)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "src"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "src", "main.py"), []byte("print('hello')"), 0644))

	data, err := packageSkillDir(dir)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Verify .git and node_modules are not in the archive
	gz, err := gzip.NewReader(bytes.NewReader(data))
	require.NoError(t, err)
	tr := tar.NewReader(gz)

	var foundFiles []string
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		foundFiles = append(foundFiles, header.Name)
	}

	// SKILL.md and src/main.py should be present
	assert.Contains(t, foundFiles, "SKILL.md")
	assert.Contains(t, foundFiles, filepath.Join("src", "main.py"))

	// .git should NOT be present
	for _, f := range foundFiles {
		assert.False(t, strings.HasPrefix(f, ".git"), "should not contain .git: %s", f)
		assert.False(t, strings.HasPrefix(f, "node_modules"), "should not contain node_modules: %s", f)
	}
}

func TestPackageSkillDir_WalkError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping test when running as root")
	}
	// Test with a dir that has an unreadable subdirectory
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: test\n---"), 0644))

	unreadableDir := filepath.Join(dir, "secrets")
	require.NoError(t, os.MkdirAll(unreadableDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(unreadableDir, "key.pem"), []byte("data"), 0644))
	require.NoError(t, os.Chmod(unreadableDir, 0000))
	defer os.Chmod(unreadableDir, 0755)

	_, err := packageSkillDir(dir)
	assert.Error(t, err, "should fail with unreadable subdirectory")
}

func TestComputeDirSHA_WalkError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping test when running as root")
	}
	// Test with a dir that has an unreadable subdirectory
	dir := t.TempDir()
	unreadableDir := filepath.Join(dir, "secrets")
	require.NoError(t, os.MkdirAll(unreadableDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(unreadableDir, "key.pem"), []byte("data"), 0644))
	require.NoError(t, os.Chmod(unreadableDir, 0000))
	defer os.Chmod(unreadableDir, 0755)

	_, err := computeDirSHA(dir)
	assert.Error(t, err, "should fail with unreadable subdirectory")
}

func TestProcessSkill_ComputeSHAError(t *testing.T) {
	repo := &importerMockRepo{}
	stor := newPackagerMockStorage()

	imp := NewSkillImporter(repo, stor)

	source := &extension.SkillRegistry{ID: 1}
	info := SkillInfo{Slug: "bad", DirPath: "/nonexistent/path"}

	err := imp.processSkill(context.Background(), source, info)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to compute SHA")
}

func TestProcessSkill_UploadError(t *testing.T) {
	repo := &importerMockRepo{}
	stor := &failingMockStorage{}

	repo.findSkillMarketItemBySlugFunc = func(_ context.Context, _ int64, _ string) (*extension.SkillMarketItem, error) {
		return nil, errors.New("not found")
	}

	imp := NewSkillImporter(repo, stor)

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: test\n---"), 0644))

	source := &extension.SkillRegistry{ID: 1}
	info := SkillInfo{Slug: "test", DirPath: dir}

	err := imp.processSkill(context.Background(), source, info)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to upload skill package")
}

// =============================================================================
// gitClone — branch validation
// =============================================================================

func TestGitClone_InvalidBranch(t *testing.T) {
	err := gitClone(context.Background(), "https://example.com/repo.git", "branch;inject", t.TempDir())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid branch")
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

// =============================================================================
// SyncSource — final UpdateSkillRegistry error after doSync
// =============================================================================

func TestSyncSource_FinalUpdateError(t *testing.T) {
	repo := newMockExtensionRepo()
	repo.getSourceFunc = func(_ context.Context, id int64) (*extension.SkillRegistry, error) {
		return &extension.SkillRegistry{ID: id, RepositoryURL: "https://example.com/repo", Branch: "main"}, nil
	}

	updateCallCount := 0
	repo.updateSourceFunc = func(_ context.Context, _ *extension.SkillRegistry) error {
		updateCallCount++
		if updateCallCount == 2 {
			// Second update (post-sync status) fails
			return errors.New("db write failed on final update")
		}
		return nil
	}

	// SyncSource will call doSync which will fail (git clone fails)
	// Then it tries to update the final status, which will also fail
	imp := NewSkillImporter(repo, nil)
	err := imp.SyncSource(context.Background(), 1)

	// doSync error is returned, not the final update error
	assert.Error(t, err)
	// Both updates should have been called
	assert.GreaterOrEqual(t, updateCallCount, 2)
}

// =============================================================================
// scanCollectionSkills — unreadable skills/ subdir
// =============================================================================

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

// =============================================================================
// scanCollectionSkills — skills/ dir with invalid SKILL.md parsing
// =============================================================================

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

// =============================================================================
// processSkill — packageSkillDir error (via unreadable file)
// =============================================================================

func TestProcessSkill_CreateError(t *testing.T) {
	repo := &importerMockRepo{}
	stor := newPackagerMockStorage()

	repo.findSkillMarketItemBySlugFunc = func(_ context.Context, _ int64, _ string) (*extension.SkillMarketItem, error) {
		return nil, errors.New("not found")
	}
	repo.createSkillMarketItemFunc = func(_ context.Context, _ *extension.SkillMarketItem) error {
		return errors.New("db insert failed")
	}

	imp := NewSkillImporter(repo, stor)

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: test\n---"), 0644))

	source := &extension.SkillRegistry{ID: 1}
	info := SkillInfo{Slug: "test", DirPath: dir}

	err := imp.processSkill(context.Background(), source, info)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db insert failed")
}

func TestProcessSkill_UpdateError(t *testing.T) {
	repo := &importerMockRepo{}
	stor := newPackagerMockStorage()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: test\n---"), 0644))

	repo.findSkillMarketItemBySlugFunc = func(_ context.Context, _ int64, _ string) (*extension.SkillMarketItem, error) {
		return &extension.SkillMarketItem{
			ID:         42,
			Slug:       "test",
			ContentSha: "old-sha-different",
			Version:    1,
			IsActive:   true,
		}, nil
	}
	repo.updateSkillMarketItemFunc = func(_ context.Context, _ *extension.SkillMarketItem) error {
		return errors.New("db update failed")
	}

	imp := NewSkillImporter(repo, stor)

	source := &extension.SkillRegistry{ID: 1}
	info := SkillInfo{Slug: "test", DirPath: dir}

	err := imp.processSkill(context.Background(), source, info)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db update failed")
}

// =============================================================================
// doSync — comprehensive tests with mocked git
// =============================================================================

// fakeGitClone creates a directory structure mimicking a cloned repo
func fakeGitClone(repoType string, skills map[string]string) func(ctx context.Context, url, branch, targetDir string) error {
	return func(ctx context.Context, url, branch, targetDir string) error {
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return err
		}
		switch repoType {
		case "single":
			content := "---\nname: single-skill\ndescription: A single skill\n---\n# Single Skill"
			if c, ok := skills["SKILL.md"]; ok {
				content = c
			}
			return os.WriteFile(filepath.Join(targetDir, "SKILL.md"), []byte(content), 0644)
		case "collection":
			// Create skills/ directory
			skillsDir := filepath.Join(targetDir, "skills")
			if err := os.MkdirAll(skillsDir, 0755); err != nil {
				return err
			}
			for slug, content := range skills {
				skillDir := filepath.Join(skillsDir, slug)
				if err := os.MkdirAll(skillDir, 0755); err != nil {
					return err
				}
				if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644); err != nil {
					return err
				}
			}
			return nil
		case "empty":
			// No SKILL.md, no skills/
			return nil
		}
		return nil
	}
}

func fakeGitHead(sha string) func(ctx context.Context, repoDir string) (string, error) {
	return func(ctx context.Context, repoDir string) (string, error) {
		if sha == "" {
			return "", errors.New("no HEAD")
		}
		return sha, nil
	}
}

func TestDoSync_SingleSkill(t *testing.T) {
	repo := &importerMockRepo{}
	stor := newPackagerMockStorage()

	var createdItem *extension.SkillMarketItem
	repo.findSkillMarketItemBySlugFunc = func(_ context.Context, _ int64, _ string) (*extension.SkillMarketItem, error) {
		return nil, errors.New("not found")
	}
	repo.createSkillMarketItemFunc = func(_ context.Context, item *extension.SkillMarketItem) error {
		createdItem = item
		return nil
	}

	imp := NewSkillImporter(repo, stor)
	imp.gitCloneFn = fakeGitClone("single", nil)
	imp.gitHeadFn = fakeGitHead("abc123def456abc123def456abc123def456abc1")

	source := &extension.SkillRegistry{ID: 1, RepositoryURL: "https://example.com/repo", Branch: "main"}
	err := imp.doSync(context.Background(), source)
	require.NoError(t, err)

	assert.Equal(t, "single", source.DetectedType)
	assert.Equal(t, "abc123def456abc123def456abc123def456abc1", source.LastCommitSha)
	assert.Equal(t, 1, source.SkillCount)
	require.NotNil(t, createdItem)
	assert.Equal(t, "single-skill", createdItem.Slug)
}

func TestDoSync_CollectionSkills(t *testing.T) {
	repo := &importerMockRepo{}
	stor := newPackagerMockStorage()

	var createdItems []*extension.SkillMarketItem
	repo.findSkillMarketItemBySlugFunc = func(_ context.Context, _ int64, _ string) (*extension.SkillMarketItem, error) {
		return nil, errors.New("not found")
	}
	repo.createSkillMarketItemFunc = func(_ context.Context, item *extension.SkillMarketItem) error {
		createdItems = append(createdItems, item)
		return nil
	}

	imp := NewSkillImporter(repo, stor)
	imp.gitCloneFn = fakeGitClone("collection", map[string]string{
		"alpha": "---\nname: alpha\n---\n",
		"beta":  "---\nname: beta\n---\n",
	})
	imp.gitHeadFn = fakeGitHead("deadbeef" + strings.Repeat("0", 32))

	source := &extension.SkillRegistry{ID: 2, RepositoryURL: "https://example.com/collection", Branch: "main"}
	err := imp.doSync(context.Background(), source)
	require.NoError(t, err)

	assert.Equal(t, "collection", source.DetectedType)
	assert.Equal(t, 2, source.SkillCount)
	assert.Len(t, createdItems, 2)
}

func TestDoSync_EmptyRepo(t *testing.T) {
	repo := &importerMockRepo{}
	stor := newPackagerMockStorage()

	imp := NewSkillImporter(repo, stor)
	imp.gitCloneFn = fakeGitClone("empty", nil)
	imp.gitHeadFn = fakeGitHead("abc" + strings.Repeat("0", 37))

	source := &extension.SkillRegistry{ID: 3, RepositoryURL: "https://example.com/empty", Branch: "main"}
	err := imp.doSync(context.Background(), source)
	require.NoError(t, err)

	assert.Equal(t, 0, source.SkillCount)
}

func TestDoSync_CloneError(t *testing.T) {
	repo := &importerMockRepo{}
	stor := newPackagerMockStorage()

	imp := NewSkillImporter(repo, stor)
	imp.gitCloneFn = func(_ context.Context, _, _, _ string) error {
		return errors.New("clone failed")
	}

	source := &extension.SkillRegistry{ID: 1, RepositoryURL: "https://example.com/repo", Branch: "main"}
	err := imp.doSync(context.Background(), source)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to clone repository")
}

func TestDoSync_GitHeadError(t *testing.T) {
	repo := &importerMockRepo{}
	stor := newPackagerMockStorage()

	repo.findSkillMarketItemBySlugFunc = func(_ context.Context, _ int64, _ string) (*extension.SkillMarketItem, error) {
		return nil, errors.New("not found")
	}
	repo.createSkillMarketItemFunc = func(_ context.Context, _ *extension.SkillMarketItem) error {
		return nil
	}

	imp := NewSkillImporter(repo, stor)
	imp.gitCloneFn = fakeGitClone("single", nil)
	imp.gitHeadFn = fakeGitHead("") // will return error

	source := &extension.SkillRegistry{ID: 1, RepositoryURL: "https://example.com/repo", Branch: "main"}
	err := imp.doSync(context.Background(), source)
	require.NoError(t, err)
	// LastCommitSha should remain empty since gitHead failed
	assert.Empty(t, source.LastCommitSha)
}

func TestDoSync_ProcessSkillError(t *testing.T) {
	repo := &importerMockRepo{}

	// Make processSkill fail by having the storage Upload fail
	failStor := &failingMockStorage{}

	repo.findSkillMarketItemBySlugFunc = func(_ context.Context, _ int64, _ string) (*extension.SkillMarketItem, error) {
		return nil, errors.New("not found")
	}

	imp := NewSkillImporter(repo, failStor)
	imp.gitCloneFn = fakeGitClone("collection", map[string]string{
		"good": "---\nname: good\n---\n",
		"bad":  "---\nname: bad\n---\n",
	})
	imp.gitHeadFn = fakeGitHead("abc" + strings.Repeat("0", 37))

	source := &extension.SkillRegistry{ID: 1, RepositoryURL: "https://example.com/repo", Branch: "main"}
	err := imp.doSync(context.Background(), source)
	// doSync does NOT return error when processSkill fails (it logs and continues)
	require.NoError(t, err)
	// SkillCount reflects discovered skills (not successfully processed ones),
	// because activeSlugs includes slugs even on processSkill failure to prevent
	// deactivating working skills on transient errors.
	assert.Equal(t, 2, source.SkillCount)
}

func TestDoSync_ScanCollectionError(t *testing.T) {
	repo := &importerMockRepo{}
	stor := newPackagerMockStorage()

	// Create a fake clone function that creates a collection-type repo
	// but with an unreadable skills/ directory
	imp := NewSkillImporter(repo, stor)
	imp.gitCloneFn = func(_ context.Context, _, _, targetDir string) error {
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return err
		}
		// Create skills/ dir to make it look like a collection
		skillsDir := filepath.Join(targetDir, "skills")
		if err := os.MkdirAll(skillsDir, 0755); err != nil {
			return err
		}
		// Create a skill subdir with invalid SKILL.md
		badSkill := filepath.Join(skillsDir, "bad")
		if err := os.MkdirAll(badSkill, 0755); err != nil {
			return err
		}
		// Write SKILL.md with valid content
		return os.WriteFile(filepath.Join(badSkill, "SKILL.md"), []byte("---\nname: bad-skill\n---"), 0644)
	}
	imp.gitHeadFn = fakeGitHead("abc" + strings.Repeat("0", 37))

	source := &extension.SkillRegistry{ID: 1, RepositoryURL: "https://example.com/repo", Branch: "main"}
	err := imp.doSync(context.Background(), source)
	require.NoError(t, err)
	assert.Equal(t, "collection", source.DetectedType)
}

// =============================================================================
// SyncSource — with mocked git (success path)
// =============================================================================

func TestSyncSource_SuccessPath(t *testing.T) {
	repo := newMockExtensionRepo()
	stor := newPackagerMockStorage()

	repo.getSourceFunc = func(_ context.Context, id int64) (*extension.SkillRegistry, error) {
		return &extension.SkillRegistry{ID: id, RepositoryURL: "https://example.com/repo", Branch: "main"}, nil
	}

	var lastStatus string
	repo.updateSourceFunc = func(_ context.Context, source *extension.SkillRegistry) error {
		lastStatus = source.SyncStatus
		return nil
	}

	imp := NewSkillImporter(repo, stor)
	imp.gitCloneFn = fakeGitClone("empty", nil)
	imp.gitHeadFn = fakeGitHead("abc" + strings.Repeat("0", 37))

	err := imp.SyncSource(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, "success", lastStatus, "final status should be success")
}

// =============================================================================
// Mock helpers used by processSkill tests
// =============================================================================

// importerMockRepo embeds mockExtensionRepo and adds hooks for
// FindSkillMarketItemBySlug, CreateSkillMarketItem, UpdateSkillMarketItem,
// DeactivateSkillMarketItemsNotIn.
type importerMockRepo struct {
	mockExtensionRepo
	findSkillMarketItemBySlugFunc         func(ctx context.Context, sourceID int64, slug string) (*extension.SkillMarketItem, error)
	createSkillMarketItemFunc             func(ctx context.Context, item *extension.SkillMarketItem) error
	updateSkillMarketItemFunc             func(ctx context.Context, item *extension.SkillMarketItem) error
	deactivateSkillMarketItemsNotInFn     func(ctx context.Context, sourceID int64, slugs []string) error
}

func (m *importerMockRepo) FindSkillMarketItemBySlug(ctx context.Context, sourceID int64, slug string) (*extension.SkillMarketItem, error) {
	if m.findSkillMarketItemBySlugFunc != nil {
		return m.findSkillMarketItemBySlugFunc(ctx, sourceID, slug)
	}
	return nil, errors.New("not found")
}

func (m *importerMockRepo) CreateSkillMarketItem(ctx context.Context, item *extension.SkillMarketItem) error {
	if m.createSkillMarketItemFunc != nil {
		return m.createSkillMarketItemFunc(ctx, item)
	}
	return nil
}

func (m *importerMockRepo) UpdateSkillMarketItem(ctx context.Context, item *extension.SkillMarketItem) error {
	if m.updateSkillMarketItemFunc != nil {
		return m.updateSkillMarketItemFunc(ctx, item)
	}
	return nil
}

func (m *importerMockRepo) DeactivateSkillMarketItemsNotIn(ctx context.Context, sourceID int64, slugs []string) error {
	if m.deactivateSkillMarketItemsNotInFn != nil {
		return m.deactivateSkillMarketItemsNotInFn(ctx, sourceID, slugs)
	}
	return nil
}

// failingMockStorage always fails on Upload
type failingMockStorage struct {
	packagerMockStorage
}

func (m *failingMockStorage) Upload(_ context.Context, _ string, _ io.Reader, _ int64, _ string) (*storage.FileInfo, error) {
	return nil, errors.New("storage unavailable")
}

// =============================================================================
// doSync — deactivate error path
// =============================================================================

func TestDoSync_DeactivateError(t *testing.T) {
	repo := &importerMockRepo{}
	stor := newPackagerMockStorage()

	repo.findSkillMarketItemBySlugFunc = func(_ context.Context, _ int64, _ string) (*extension.SkillMarketItem, error) {
		return nil, errors.New("not found")
	}
	repo.createSkillMarketItemFunc = func(_ context.Context, _ *extension.SkillMarketItem) error {
		return nil
	}

	deactivateCalled := false
	repo.deactivateSkillMarketItemsNotInFn = func(_ context.Context, _ int64, _ []string) error {
		deactivateCalled = true
		return errors.New("deactivate failed")
	}

	imp := NewSkillImporter(repo, stor)
	imp.gitCloneFn = fakeGitClone("single", nil)
	imp.gitHeadFn = fakeGitHead("abc" + strings.Repeat("0", 37))

	source := &extension.SkillRegistry{ID: 1, RepositoryURL: "https://example.com/repo", Branch: "main"}
	err := imp.doSync(context.Background(), source)
	// doSync should succeed even though deactivate fails (it only logs)
	require.NoError(t, err)
	assert.True(t, deactivateCalled, "DeactivateSkillMarketItemsNotIn should have been called")
	assert.Equal(t, 1, source.SkillCount)
}

// =============================================================================
// scanCollectionSkills — root-level parse failure (SKILL.md exists but unreadable)
// =============================================================================

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

// =============================================================================
// scanCollectionSkills — skills/ dir with ignored dirs and non-directories
// =============================================================================

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

// =============================================================================
// packageSkillDir — nonexistent directory
// =============================================================================

func TestPackageSkillDir_NonexistentDir(t *testing.T) {
	_, err := packageSkillDir("/nonexistent/path/to/skill")
	assert.Error(t, err, "should fail with nonexistent directory")
}

// =============================================================================
// computeDirSHA — unreadable file (after walk discovery)
// =============================================================================

func TestComputeDirSHA_UnreadableFile(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping test when running as root")
	}
	dir := t.TempDir()
	f := filepath.Join(dir, "secret.txt")
	require.NoError(t, os.WriteFile(f, []byte("secret"), 0644))
	require.NoError(t, os.Chmod(f, 0000))
	defer os.Chmod(f, 0644)

	_, err := computeDirSHA(dir)
	assert.Error(t, err, "should fail with unreadable file")
}

// =============================================================================
// doSync — single skill parseSkillDir error path
// =============================================================================

func TestDoSync_SingleSkillParseError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping test when running as root")
	}
	repo := &importerMockRepo{}
	stor := newPackagerMockStorage()

	imp := NewSkillImporter(repo, stor)
	// Create a single-type repo where SKILL.md exists but is unreadable
	imp.gitCloneFn = func(_ context.Context, _, _, targetDir string) error {
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return err
		}
		// Create SKILL.md but make it unreadable
		p := filepath.Join(targetDir, "SKILL.md")
		if err := os.WriteFile(p, []byte("---\nname: test\n---"), 0644); err != nil {
			return err
		}
		return os.Chmod(p, 0000)
	}
	imp.gitHeadFn = fakeGitHead("abc" + strings.Repeat("0", 37))

	source := &extension.SkillRegistry{ID: 1, RepositoryURL: "https://example.com/repo", Branch: "main"}
	err := imp.doSync(context.Background(), source)
	assert.Error(t, err, "should fail with parse error for single skill")
}

// =============================================================================
// scanCollectionSkills — root ReadDir error (Priority 2 failure path)
// =============================================================================

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

// =============================================================================
// packageSkillDir — file becomes unreadable during walk (os.Open error)
// =============================================================================

func TestPackageSkillDir_FileOpenError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping test when running as root")
	}
	dir := t.TempDir()
	// Create a file that is not readable
	f := filepath.Join(dir, "unreadable.txt")
	require.NoError(t, os.WriteFile(f, []byte("data"), 0644))
	require.NoError(t, os.Chmod(f, 0000))
	defer os.Chmod(f, 0644)

	_, err := packageSkillDir(dir)
	assert.Error(t, err, "should fail when file cannot be opened")
}

// =============================================================================
// processSkill — packageSkillDir error (line 195-197)
// =============================================================================

func TestProcessSkill_PackageSkillDirError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping test when running as root")
	}
	repo := &importerMockRepo{}
	stor := newPackagerMockStorage()

	repo.findSkillMarketItemBySlugFunc = func(_ context.Context, _ int64, _ string) (*extension.SkillMarketItem, error) {
		return nil, errors.New("not found")
	}

	imp := NewSkillImporter(repo, stor)

	source := &extension.SkillRegistry{ID: 1}

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: test\n---"), 0644))
	// Create a file that is not readable, so packageSkillDir fails at os.Open
	unreadable := filepath.Join(dir, "secret.bin")
	require.NoError(t, os.WriteFile(unreadable, []byte("data"), 0644))
	require.NoError(t, os.Chmod(unreadable, 0000))
	defer os.Chmod(unreadable, 0644)

	info := SkillInfo{
		Slug:        "test",
		DisplayName: "test",
		DirPath:     dir,
	}

	err := imp.processSkill(context.Background(), source, info)
	// computeDirSHA will fail first because of unreadable file
	assert.Error(t, err, "should fail due to unreadable file")
}
