package extension

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
	"github.com/anthropics/agentsmesh/backend/internal/infra/storage"
)

// importerMockRepo embeds mockExtensionRepo and adds hooks for
// FindSkillMarketItemBySlug, CreateSkillMarketItem, UpdateSkillMarketItem,
// DeactivateSkillMarketItemsNotIn.
type importerMockRepo struct {
	mockExtensionRepo
	findSkillMarketItemBySlugFunc     func(ctx context.Context, sourceID int64, slug string) (*extension.SkillMarketItem, error)
	createSkillMarketItemFunc         func(ctx context.Context, item *extension.SkillMarketItem) error
	updateSkillMarketItemFunc         func(ctx context.Context, item *extension.SkillMarketItem) error
	deactivateSkillMarketItemsNotInFn func(ctx context.Context, sourceID int64, slugs []string) error
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

// fakeGitClone creates a directory structure mimicking a cloned repo.
func fakeGitClone(repoType string, skills map[string]string) func(ctx context.Context, url, branch, targetDir string) error {
	return func(_ context.Context, _, _, targetDir string) error {
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
			return nil
		}
		return nil
	}
}

func fakeGitHead(sha string) func(ctx context.Context, repoDir string) (string, error) {
	return func(_ context.Context, _ string) (string, error) {
		if sha == "" {
			return "", errors.New("no HEAD")
		}
		return sha, nil
	}
}
