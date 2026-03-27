package extension

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	assert.Empty(t, source.LastCommitSha)
}

func TestDoSync_ProcessSkillError(t *testing.T) {
	repo := &importerMockRepo{}
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
	require.NoError(t, err)
	assert.Equal(t, 2, source.SkillCount)
}

func TestDoSync_ScanCollectionError(t *testing.T) {
	repo := &importerMockRepo{}
	stor := newPackagerMockStorage()

	imp := NewSkillImporter(repo, stor)
	imp.gitCloneFn = func(_ context.Context, _, _, targetDir string) error {
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return err
		}
		skillsDir := filepath.Join(targetDir, "skills")
		if err := os.MkdirAll(skillsDir, 0755); err != nil {
			return err
		}
		badSkill := filepath.Join(skillsDir, "bad")
		if err := os.MkdirAll(badSkill, 0755); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(badSkill, "SKILL.md"), []byte("---\nname: bad-skill\n---"), 0644)
	}
	imp.gitHeadFn = fakeGitHead("abc" + strings.Repeat("0", 37))

	source := &extension.SkillRegistry{ID: 1, RepositoryURL: "https://example.com/repo", Branch: "main"}
	err := imp.doSync(context.Background(), source)
	require.NoError(t, err)
	assert.Equal(t, "collection", source.DetectedType)
}

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
	require.NoError(t, err)
	assert.True(t, deactivateCalled, "DeactivateSkillMarketItemsNotIn should have been called")
	assert.Equal(t, 1, source.SkillCount)
}

func TestDoSync_SingleSkillParseError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping test when running as root")
	}
	repo := &importerMockRepo{}
	stor := newPackagerMockStorage()

	imp := NewSkillImporter(repo, stor)
	imp.gitCloneFn = func(_ context.Context, _, _, targetDir string) error {
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return err
		}
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
