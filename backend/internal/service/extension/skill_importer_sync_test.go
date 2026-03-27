package extension

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// gitClone — URL validation
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

	err := gitClone(ctx, "https://invalid-host.example.com/no-such-repo.git", "", targetDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "git clone failed")
	assert.NotContains(t, err.Error(), "only https:// URLs are allowed")
}

func TestGitClone_InvalidBranch(t *testing.T) {
	err := gitClone(context.Background(), "https://example.com/repo.git", "branch;inject", t.TempDir())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid branch")
}

// =============================================================================
// gitHead
// =============================================================================

func TestGitHead_ValidRepo(t *testing.T) {
	dir := t.TempDir()

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
	for _, c := range sha {
		assert.True(t, (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f'),
			"SHA should be hex, got char: %c", c)
	}
}

func TestGitHead_InvalidRepo(t *testing.T) {
	dir := t.TempDir()
	_, err := gitHead(context.Background(), dir)
	assert.Error(t, err)
}

// =============================================================================
// SyncSource — error paths
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

	imp := NewSkillImporter(repo, nil)
	err := imp.SyncSource(context.Background(), 1)

	assert.Error(t, err)
	assert.GreaterOrEqual(t, updateCalls, 2)
	assert.Equal(t, "failed", lastStatus)
	assert.NotEmpty(t, lastError)
}

func TestSyncSource_FinalUpdateError(t *testing.T) {
	repo := newMockExtensionRepo()
	repo.getSourceFunc = func(_ context.Context, id int64) (*extension.SkillRegistry, error) {
		return &extension.SkillRegistry{ID: id, RepositoryURL: "https://example.com/repo", Branch: "main"}, nil
	}

	updateCallCount := 0
	repo.updateSourceFunc = func(_ context.Context, _ *extension.SkillRegistry) error {
		updateCallCount++
		if updateCallCount == 2 {
			return errors.New("db write failed on final update")
		}
		return nil
	}

	imp := NewSkillImporter(repo, nil)
	err := imp.SyncSource(context.Background(), 1)

	assert.Error(t, err)
	assert.GreaterOrEqual(t, updateCallCount, 2)
}

// =============================================================================
// SyncSource — success path (with mocked git)
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
