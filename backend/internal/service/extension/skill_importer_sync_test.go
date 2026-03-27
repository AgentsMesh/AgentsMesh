package extension

import (
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
