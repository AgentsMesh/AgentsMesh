package extension

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

	assert.Len(t, stor.uploaded, 1)
}

func TestProcessSkill_ExistingSameSHA(t *testing.T) {
	repo := &importerMockRepo{}
	stor := newPackagerMockStorage()

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

	createCalled := false
	repo.createSkillMarketItemFunc = func(_ context.Context, _ *extension.SkillMarketItem) error {
		createCalled = true
		return nil
	}
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
			IsActive:   false,
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
	assert.Error(t, err, "should fail due to unreadable file")
}
