package extension

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
	"github.com/anthropics/agentsmesh/backend/internal/infra/storage"
	"github.com/anthropics/agentsmesh/backend/pkg/crypto"
)

// ---------------------------------------------------------------------------
// Mock: extension.Repository (prefixed to avoid conflict with skill_packager_test.go)
// ---------------------------------------------------------------------------

type svcMockRepo struct {
	// Skill Registries
	listSkillRegistriesFn            func(ctx context.Context, orgID *int64) ([]*extension.SkillRegistry, error)
	getSkillRegistryFn              func(ctx context.Context, id int64) (*extension.SkillRegistry, error)
	createSkillRegistryFn           func(ctx context.Context, source *extension.SkillRegistry) error
	updateSkillRegistryFn           func(ctx context.Context, source *extension.SkillRegistry) error
	deleteSkillRegistryFn           func(ctx context.Context, id int64) error
	findSkillRegistryByURLFn        func(ctx context.Context, orgID *int64, repoURL string) (*extension.SkillRegistry, error)

	// Skill Market Items
	listSkillMarketItemsFn              func(ctx context.Context, orgID *int64, query string, category string) ([]*extension.SkillMarketItem, error)
	getSkillMarketItemFn                func(ctx context.Context, id int64) (*extension.SkillMarketItem, error)
	findSkillMarketItemBySlugFn         func(ctx context.Context, registryID int64, slug string) (*extension.SkillMarketItem, error)
	createSkillMarketItemFn             func(ctx context.Context, item *extension.SkillMarketItem) error
	updateSkillMarketItemFn             func(ctx context.Context, item *extension.SkillMarketItem) error
	deactivateSkillMarketItemsNotInFn   func(ctx context.Context, registryID int64, slugs []string) error

	// MCP Market Items
	listMcpMarketItemsFn func(ctx context.Context, query string, category string, limit, offset int) ([]*extension.McpMarketItem, int64, error)
	getMcpMarketItemFn   func(ctx context.Context, id int64) (*extension.McpMarketItem, error)

	// Installed MCP Servers
	listInstalledMcpServersFn    func(ctx context.Context, orgID, repoID int64, scope string) ([]*extension.InstalledMcpServer, error)
	getInstalledMcpServerFn      func(ctx context.Context, id int64) (*extension.InstalledMcpServer, error)
	createInstalledMcpServerFn   func(ctx context.Context, server *extension.InstalledMcpServer) error
	updateInstalledMcpServerFn   func(ctx context.Context, server *extension.InstalledMcpServer) error
	deleteInstalledMcpServerFn   func(ctx context.Context, id int64) error
	getEffectiveMcpServersFn     func(ctx context.Context, orgID, userID, repoID int64) ([]*extension.InstalledMcpServer, error)

	// Installed Skills
	listInstalledSkillsFn    func(ctx context.Context, orgID, repoID int64, scope string) ([]*extension.InstalledSkill, error)
	getInstalledSkillFn      func(ctx context.Context, id int64) (*extension.InstalledSkill, error)
	createInstalledSkillFn   func(ctx context.Context, skill *extension.InstalledSkill) error
	updateInstalledSkillFn   func(ctx context.Context, skill *extension.InstalledSkill) error
	deleteInstalledSkillFn   func(ctx context.Context, id int64) error
	getEffectiveSkillsFn     func(ctx context.Context, orgID, userID, repoID int64) ([]*extension.InstalledSkill, error)

	// Skill Registry Overrides
	setSkillRegistryOverrideFn   func(ctx context.Context, orgID int64, registryID int64, isDisabled bool) error
	listSkillRegistryOverridesFn func(ctx context.Context, orgID int64) ([]*extension.SkillRegistryOverride, error)
}

func (m *svcMockRepo) ListSkillRegistries(ctx context.Context, orgID *int64) ([]*extension.SkillRegistry, error) {
	if m.listSkillRegistriesFn != nil {
		return m.listSkillRegistriesFn(ctx, orgID)
	}
	return nil, nil
}

func (m *svcMockRepo) GetSkillRegistry(ctx context.Context, id int64) (*extension.SkillRegistry, error) {
	if m.getSkillRegistryFn != nil {
		return m.getSkillRegistryFn(ctx, id)
	}
	return nil, fmt.Errorf("not found")
}

func (m *svcMockRepo) CreateSkillRegistry(ctx context.Context, source *extension.SkillRegistry) error {
	if m.createSkillRegistryFn != nil {
		return m.createSkillRegistryFn(ctx, source)
	}
	return nil
}

func (m *svcMockRepo) UpdateSkillRegistry(ctx context.Context, source *extension.SkillRegistry) error {
	if m.updateSkillRegistryFn != nil {
		return m.updateSkillRegistryFn(ctx, source)
	}
	return nil
}

func (m *svcMockRepo) DeleteSkillRegistry(ctx context.Context, id int64) error {
	if m.deleteSkillRegistryFn != nil {
		return m.deleteSkillRegistryFn(ctx, id)
	}
	return nil
}

func (m *svcMockRepo) FindSkillRegistryByURL(ctx context.Context, orgID *int64, repoURL string) (*extension.SkillRegistry, error) {
	if m.findSkillRegistryByURLFn != nil {
		return m.findSkillRegistryByURLFn(ctx, orgID, repoURL)
	}
	return nil, fmt.Errorf("not found")
}

func (m *svcMockRepo) ListSkillMarketItems(ctx context.Context, orgID *int64, query string, category string) ([]*extension.SkillMarketItem, error) {
	if m.listSkillMarketItemsFn != nil {
		return m.listSkillMarketItemsFn(ctx, orgID, query, category)
	}
	return nil, nil
}

func (m *svcMockRepo) GetSkillMarketItem(ctx context.Context, id int64) (*extension.SkillMarketItem, error) {
	if m.getSkillMarketItemFn != nil {
		return m.getSkillMarketItemFn(ctx, id)
	}
	return nil, fmt.Errorf("not found")
}

func (m *svcMockRepo) FindSkillMarketItemBySlug(ctx context.Context, registryID int64, slug string) (*extension.SkillMarketItem, error) {
	if m.findSkillMarketItemBySlugFn != nil {
		return m.findSkillMarketItemBySlugFn(ctx, registryID, slug)
	}
	return nil, fmt.Errorf("not found")
}

func (m *svcMockRepo) CreateSkillMarketItem(ctx context.Context, item *extension.SkillMarketItem) error {
	if m.createSkillMarketItemFn != nil {
		return m.createSkillMarketItemFn(ctx, item)
	}
	return nil
}

func (m *svcMockRepo) UpdateSkillMarketItem(ctx context.Context, item *extension.SkillMarketItem) error {
	if m.updateSkillMarketItemFn != nil {
		return m.updateSkillMarketItemFn(ctx, item)
	}
	return nil
}

func (m *svcMockRepo) DeactivateSkillMarketItemsNotIn(ctx context.Context, registryID int64, slugs []string) error {
	if m.deactivateSkillMarketItemsNotInFn != nil {
		return m.deactivateSkillMarketItemsNotInFn(ctx, registryID, slugs)
	}
	return nil
}

func (m *svcMockRepo) ListMcpMarketItems(ctx context.Context, query string, category string, limit, offset int) ([]*extension.McpMarketItem, int64, error) {
	if m.listMcpMarketItemsFn != nil {
		return m.listMcpMarketItemsFn(ctx, query, category, limit, offset)
	}
	return nil, 0, nil
}

func (m *svcMockRepo) GetMcpMarketItem(ctx context.Context, id int64) (*extension.McpMarketItem, error) {
	if m.getMcpMarketItemFn != nil {
		return m.getMcpMarketItemFn(ctx, id)
	}
	return nil, fmt.Errorf("not found")
}

func (m *svcMockRepo) FindMcpMarketItemByRegistryName(_ context.Context, _ string) (*extension.McpMarketItem, error) {
	return nil, fmt.Errorf("not found")
}

func (m *svcMockRepo) UpsertMcpMarketItem(_ context.Context, _ *extension.McpMarketItem) error {
	return nil
}

func (m *svcMockRepo) DeactivateMcpMarketItemsNotIn(_ context.Context, _ string, _ []string) (int64, error) {
	return 0, nil
}

func (m *svcMockRepo) ListInstalledMcpServers(ctx context.Context, orgID, repoID, userID int64, scope string) ([]*extension.InstalledMcpServer, error) {
	if m.listInstalledMcpServersFn != nil {
		return m.listInstalledMcpServersFn(ctx, orgID, repoID, scope)
	}
	return nil, nil
}

func (m *svcMockRepo) GetInstalledMcpServer(ctx context.Context, id int64) (*extension.InstalledMcpServer, error) {
	if m.getInstalledMcpServerFn != nil {
		return m.getInstalledMcpServerFn(ctx, id)
	}
	return nil, fmt.Errorf("not found")
}

func (m *svcMockRepo) CreateInstalledMcpServer(ctx context.Context, server *extension.InstalledMcpServer) error {
	if m.createInstalledMcpServerFn != nil {
		return m.createInstalledMcpServerFn(ctx, server)
	}
	return nil
}

func (m *svcMockRepo) UpdateInstalledMcpServer(ctx context.Context, server *extension.InstalledMcpServer) error {
	if m.updateInstalledMcpServerFn != nil {
		return m.updateInstalledMcpServerFn(ctx, server)
	}
	return nil
}

func (m *svcMockRepo) DeleteInstalledMcpServer(ctx context.Context, id int64) error {
	if m.deleteInstalledMcpServerFn != nil {
		return m.deleteInstalledMcpServerFn(ctx, id)
	}
	return nil
}

func (m *svcMockRepo) GetEffectiveMcpServers(ctx context.Context, orgID, userID, repoID int64) ([]*extension.InstalledMcpServer, error) {
	if m.getEffectiveMcpServersFn != nil {
		return m.getEffectiveMcpServersFn(ctx, orgID, userID, repoID)
	}
	return nil, nil
}

func (m *svcMockRepo) ListInstalledSkills(ctx context.Context, orgID, repoID, userID int64, scope string) ([]*extension.InstalledSkill, error) {
	if m.listInstalledSkillsFn != nil {
		return m.listInstalledSkillsFn(ctx, orgID, repoID, scope)
	}
	return nil, nil
}

func (m *svcMockRepo) GetInstalledSkill(ctx context.Context, id int64) (*extension.InstalledSkill, error) {
	if m.getInstalledSkillFn != nil {
		return m.getInstalledSkillFn(ctx, id)
	}
	return nil, fmt.Errorf("not found")
}

func (m *svcMockRepo) CreateInstalledSkill(ctx context.Context, skill *extension.InstalledSkill) error {
	if m.createInstalledSkillFn != nil {
		return m.createInstalledSkillFn(ctx, skill)
	}
	return nil
}

func (m *svcMockRepo) UpdateInstalledSkill(ctx context.Context, skill *extension.InstalledSkill) error {
	if m.updateInstalledSkillFn != nil {
		return m.updateInstalledSkillFn(ctx, skill)
	}
	return nil
}

func (m *svcMockRepo) DeleteInstalledSkill(ctx context.Context, id int64) error {
	if m.deleteInstalledSkillFn != nil {
		return m.deleteInstalledSkillFn(ctx, id)
	}
	return nil
}

func (m *svcMockRepo) GetEffectiveSkills(ctx context.Context, orgID, userID, repoID int64) ([]*extension.InstalledSkill, error) {
	if m.getEffectiveSkillsFn != nil {
		return m.getEffectiveSkillsFn(ctx, orgID, userID, repoID)
	}
	return nil, nil
}

func (m *svcMockRepo) SetSkillRegistryOverride(ctx context.Context, orgID int64, registryID int64, isDisabled bool) error {
	if m.setSkillRegistryOverrideFn != nil {
		return m.setSkillRegistryOverrideFn(ctx, orgID, registryID, isDisabled)
	}
	return nil
}

func (m *svcMockRepo) ListSkillRegistryOverrides(ctx context.Context, orgID int64) ([]*extension.SkillRegistryOverride, error) {
	if m.listSkillRegistryOverridesFn != nil {
		return m.listSkillRegistryOverridesFn(ctx, orgID)
	}
	return nil, nil
}

// Compile-time assertion
var _ extension.Repository = (*svcMockRepo)(nil)

// ---------------------------------------------------------------------------
// Mock: storage.Storage (prefixed to avoid conflict with skill_packager_test.go)
// ---------------------------------------------------------------------------

type svcMockStorage struct {
	uploadFn  func(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (*storage.FileInfo, error)
	deleteFn  func(ctx context.Context, key string) error
	getURLFn  func(ctx context.Context, key string, expiry time.Duration) (string, error)
	existsFn  func(ctx context.Context, key string) (bool, error)
}

func (m *svcMockStorage) Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (*storage.FileInfo, error) {
	if m.uploadFn != nil {
		return m.uploadFn(ctx, key, reader, size, contentType)
	}
	return &storage.FileInfo{Key: key}, nil
}

func (m *svcMockStorage) Delete(ctx context.Context, key string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, key)
	}
	return nil
}

func (m *svcMockStorage) GetURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	if m.getURLFn != nil {
		return m.getURLFn(ctx, key, expiry)
	}
	return "https://storage.example.com/" + key + "?signed=1", nil
}

func (m *svcMockStorage) GetInternalURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	// For tests, internal URL is the same as public URL
	return m.GetURL(ctx, key, expiry)
}

func (m *svcMockStorage) Exists(ctx context.Context, key string) (bool, error) {
	if m.existsFn != nil {
		return m.existsFn(ctx, key)
	}
	return true, nil
}

// Compile-time assertion
var _ storage.Storage = (*svcMockStorage)(nil)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newTestService(repo *svcMockRepo, stor *svcMockStorage, enc *crypto.Encryptor) *Service {
	return NewService(repo, stor, enc)
}

// int64Ptr returns a pointer to the given int64 value (test helper)
func int64Ptr(v int64) *int64 { return &v }

// newTestServiceWithPackager creates a Service with a SkillPackager that uses a
// fake git clone function. The fake clone creates a minimal skill directory with
// a SKILL.md whose slug is derived from the repository URL's last path segment.
func newTestServiceWithPackager(repo *svcMockRepo, stor *svcMockStorage, enc *crypto.Encryptor) *Service {
	svc := NewService(repo, stor, enc)
	pkg := NewSkillPackager(repo, stor)
	pkg.gitCloneFn = func(_ context.Context, url, branch, targetDir string) error {
		// Create a minimal skill directory with SKILL.md
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return err
		}
		slug := filepath.Base(url) // e.g. "repo" from "https://github.com/org/repo"
		content := "---\nslug: " + slug + "\nname: Test Skill\n---\n# Test Skill\nA test skill."
		return os.WriteFile(filepath.Join(targetDir, "SKILL.md"), []byte(content), 0644)
	}
	svc.SetSkillPackager(pkg)
	return svc
}

func ptrInt64(v int64) *int64 { return &v }
func ptrInt(v int) *int       { return &v }
func ptrBool(v bool) *bool    { return &v }

// ---------------------------------------------------------------------------
// Tests: validateScope
// ---------------------------------------------------------------------------

func TestValidateScope(t *testing.T) {
	tests := []struct {
		name    string
		scope   string
		wantErr bool
	}{
		{"valid org", "org", false},
		{"valid user", "user", false},
		{"invalid all", "all", true},
		{"invalid empty", "", true},
		{"invalid admin", "admin", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateScope(tt.scope)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateScope(%q) error = %v, wantErr %v", tt.scope, err, tt.wantErr)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Tests: CreateSkillRegistry
// ---------------------------------------------------------------------------

func TestCreateSkillRegistry_DefaultBranchAndSourceType(t *testing.T) {
	var captured *extension.SkillRegistry
	repo := &svcMockRepo{
		createSkillRegistryFn: func(_ context.Context, source *extension.SkillRegistry) error {
			captured = source
			return nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	result, err := svc.CreateSkillRegistry(context.Background(), 1, CreateSkillRegistryInput{
		RepositoryURL: "https://github.com/org/repo",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Branch != "main" {
		t.Errorf("expected branch 'main', got %q", result.Branch)
	}
	if result.SourceType != "auto" {
		t.Errorf("expected sourceType 'auto', got %q", result.SourceType)
	}
	if captured == nil {
		t.Fatal("repo.CreateSkillRegistry was not called")
	}
	if captured.SyncStatus != "pending" {
		t.Errorf("expected syncStatus 'pending', got %q", captured.SyncStatus)
	}
	if captured.OrganizationID == nil || *captured.OrganizationID != 1 {
		t.Errorf("expected orgID 1, got %v", captured.OrganizationID)
	}
}

func TestCreateSkillRegistry_CustomBranchAndSourceType(t *testing.T) {
	repo := &svcMockRepo{
		createSkillRegistryFn: func(_ context.Context, _ *extension.SkillRegistry) error {
			return nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	result, err := svc.CreateSkillRegistry(context.Background(), 1, CreateSkillRegistryInput{
		RepositoryURL: "https://github.com/org/repo",
		Branch:        "develop",
		SourceType:    "collection",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Branch != "develop" {
		t.Errorf("expected branch 'develop', got %q", result.Branch)
	}
	if result.SourceType != "collection" {
		t.Errorf("expected sourceType 'collection', got %q", result.SourceType)
	}
}

// ---------------------------------------------------------------------------
// Tests: SyncSkillRegistry
// ---------------------------------------------------------------------------

func TestSyncSkillRegistry_Success(t *testing.T) {
	orgID := int64(1)
	var updateCalls []string // track SyncStatus values passed to UpdateSkillRegistry
	lastStatus := "pending"

	repo := &svcMockRepo{
		getSkillRegistryFn: func(_ context.Context, id int64) (*extension.SkillRegistry, error) {
			return &extension.SkillRegistry{
				ID:             id,
				OrganizationID: &orgID,
				RepositoryURL:  "https://example.com/repo",
				Branch:         "main",
				SyncStatus:     lastStatus,
			}, nil
		},
		updateSkillRegistryFn: func(_ context.Context, source *extension.SkillRegistry) error {
			updateCalls = append(updateCalls, source.SyncStatus)
			lastStatus = source.SyncStatus
			return nil
		},
	}
	stor := &svcMockStorage{}
	svc := newTestService(repo, stor, nil)

	// Set up a SkillImporter with a fake git clone that creates an empty repo
	imp := NewSkillImporter(repo, stor)
	imp.gitCloneFn = func(_ context.Context, _, _, targetDir string) error {
		return os.MkdirAll(targetDir, 0755)
	}
	svc.SetSkillImporter(imp)

	result, err := svc.SyncSkillRegistry(context.Background(), orgID, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Importer should have called UpdateSkillRegistry at least twice:
	// 1. Set status to "syncing"
	// 2. Set status to "success" (final)
	if len(updateCalls) < 2 {
		t.Fatalf("expected at least 2 update calls, got %d: %v", len(updateCalls), updateCalls)
	}
	if updateCalls[0] != "syncing" {
		t.Errorf("first update should set status 'syncing', got %q", updateCalls[0])
	}

	// The final reload returns whatever getSkillRegistryFn returns
	// (which uses lastStatus updated by the importer)
	if result.SyncStatus != lastStatus {
		t.Errorf("expected status %q, got %q", lastStatus, result.SyncStatus)
	}
}

func TestSyncSkillRegistry_IDORDifferentOrg(t *testing.T) {
	otherOrg := int64(99)
	repo := &svcMockRepo{
		getSkillRegistryFn: func(_ context.Context, id int64) (*extension.SkillRegistry, error) {
			return &extension.SkillRegistry{
				ID:             id,
				OrganizationID: &otherOrg,
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	_, err := svc.SyncSkillRegistry(context.Background(), 1, 10)
	if err == nil {
		t.Fatal("expected IDOR error, got nil")
	}
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("expected ErrForbidden, got: %v", err)
	}
}

func TestSyncSkillRegistry_PlatformLevelSource(t *testing.T) {
	repo := &svcMockRepo{
		getSkillRegistryFn: func(_ context.Context, id int64) (*extension.SkillRegistry, error) {
			return &extension.SkillRegistry{
				ID:             id,
				OrganizationID: nil, // platform level
				SyncStatus:     "pending",
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	// Platform-level sources cannot be synced by org users
	_, err := svc.SyncSkillRegistry(context.Background(), 42, 10)
	if err == nil {
		t.Fatal("expected error for platform-level source, got nil")
	}
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("expected ErrForbidden, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Tests: DeleteSkillRegistry
// ---------------------------------------------------------------------------

func TestDeleteSkillRegistry_Success(t *testing.T) {
	orgID := int64(1)
	deleteCalled := false
	repo := &svcMockRepo{
		getSkillRegistryFn: func(_ context.Context, id int64) (*extension.SkillRegistry, error) {
			return &extension.SkillRegistry{
				ID:             id,
				OrganizationID: &orgID,
			}, nil
		},
		deleteSkillRegistryFn: func(_ context.Context, id int64) error {
			deleteCalled = true
			if id != 10 {
				t.Errorf("expected id 10, got %d", id)
			}
			return nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	err := svc.DeleteSkillRegistry(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleteCalled {
		t.Error("repo.DeleteSkillRegistry was not called")
	}
}

func TestDeleteSkillRegistry_IDORDifferentOrg(t *testing.T) {
	otherOrg := int64(99)
	repo := &svcMockRepo{
		getSkillRegistryFn: func(_ context.Context, id int64) (*extension.SkillRegistry, error) {
			return &extension.SkillRegistry{
				ID:             id,
				OrganizationID: &otherOrg,
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	err := svc.DeleteSkillRegistry(context.Background(), 1, 10)
	if err == nil {
		t.Fatal("expected IDOR error, got nil")
	}
}

// ---------------------------------------------------------------------------
// Tests: ListRepoSkills
// ---------------------------------------------------------------------------

func TestListRepoSkills_ScopeAllConvertsToEmpty(t *testing.T) {
	var capturedScope string
	repo := &svcMockRepo{
		listInstalledSkillsFn: func(_ context.Context, orgID, repoID int64, scope string) ([]*extension.InstalledSkill, error) {
			capturedScope = scope
			return nil, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	_, err := svc.ListRepoSkills(context.Background(), 1, 2, 100, "all")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedScope != "" {
		t.Errorf("expected empty scope, got %q", capturedScope)
	}
}

func TestListRepoSkills_ScopeOrgPassesThrough(t *testing.T) {
	var capturedScope string
	repo := &svcMockRepo{
		listInstalledSkillsFn: func(_ context.Context, orgID, repoID int64, scope string) ([]*extension.InstalledSkill, error) {
			capturedScope = scope
			return nil, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	_, err := svc.ListRepoSkills(context.Background(), 1, 2, 100, "org")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedScope != "org" {
		t.Errorf("expected scope 'org', got %q", capturedScope)
	}
}

// ---------------------------------------------------------------------------
// Tests: InstallSkillFromMarket
// ---------------------------------------------------------------------------

func TestInstallSkillFromMarket_Success(t *testing.T) {
	marketItemID := int64(100)
	repo := &svcMockRepo{
		getSkillMarketItemFn: func(_ context.Context, id int64) (*extension.SkillMarketItem, error) {
			return &extension.SkillMarketItem{
				ID:          id,
				Slug:        "test-skill",
				ContentSha:  "abc123",
				StorageKey:  "skills/test-skill/v1.tar.gz",
				PackageSize: 1024,
			}, nil
		},
		createInstalledSkillFn: func(_ context.Context, skill *extension.InstalledSkill) error {
			if skill.Slug != "test-skill" {
				t.Errorf("expected slug 'test-skill', got %q", skill.Slug)
			}
			if skill.InstallSource != "market" {
				t.Errorf("expected install_source 'market', got %q", skill.InstallSource)
			}
			if skill.ContentSha != "abc123" {
				t.Errorf("expected content_sha 'abc123', got %q", skill.ContentSha)
			}
			if skill.StorageKey != "skills/test-skill/v1.tar.gz" {
				t.Errorf("expected storage_key, got %q", skill.StorageKey)
			}
			if skill.PackageSize != 1024 {
				t.Errorf("expected package_size 1024, got %d", skill.PackageSize)
			}
			if skill.MarketItemID == nil || *skill.MarketItemID != marketItemID {
				t.Errorf("expected market_item_id %d, got %v", marketItemID, skill.MarketItemID)
			}
			if !skill.IsEnabled {
				t.Error("expected is_enabled true")
			}
			return nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	result, err := svc.InstallSkillFromMarket(context.Background(), 1, 2, 3, marketItemID, "org")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Slug != "test-skill" {
		t.Errorf("expected slug 'test-skill', got %q", result.Slug)
	}
	if result.OrganizationID != 1 {
		t.Errorf("expected org_id 1, got %d", result.OrganizationID)
	}
	if result.RepositoryID != 2 {
		t.Errorf("expected repo_id 2, got %d", result.RepositoryID)
	}
	if result.Scope != "org" {
		t.Errorf("expected scope 'org', got %q", result.Scope)
	}
}

func TestInstallSkillFromMarket_InvalidScope(t *testing.T) {
	svc := newTestService(&svcMockRepo{}, &svcMockStorage{}, nil)

	_, err := svc.InstallSkillFromMarket(context.Background(), 1, 2, 3, 100, "all")
	if err == nil {
		t.Fatal("expected error for invalid scope, got nil")
	}
}

func TestInstallSkillFromMarket_MarketItemNotFound(t *testing.T) {
	repo := &svcMockRepo{
		getSkillMarketItemFn: func(_ context.Context, id int64) (*extension.SkillMarketItem, error) {
			return nil, fmt.Errorf("record not found")
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	_, err := svc.InstallSkillFromMarket(context.Background(), 1, 2, 3, 999, "org")
	if err == nil {
		t.Fatal("expected error for missing market item, got nil")
	}
}

// ---------------------------------------------------------------------------
// Tests: InstallSkillFromGitHub
// ---------------------------------------------------------------------------

func TestInstallSkillFromGitHub_URLOnly(t *testing.T) {
	svc := newTestServiceWithPackager(&svcMockRepo{}, &svcMockStorage{}, nil)

	result, err := svc.InstallSkillFromGitHub(context.Background(), 1, 2, 3, "https://github.com/org/repo", "", "", "org")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.SourceURL != "https://github.com/org/repo" {
		t.Errorf("expected source URL 'https://github.com/org/repo', got %q", result.SourceURL)
	}
	if result.InstallSource != "github" {
		t.Errorf("expected install source 'github', got %q", result.InstallSource)
	}
}

func TestInstallSkillFromGitHub_URLAndBranch(t *testing.T) {
	svc := newTestServiceWithPackager(&svcMockRepo{}, &svcMockStorage{}, nil)

	result, err := svc.InstallSkillFromGitHub(context.Background(), 1, 2, 3, "https://github.com/org/repo", "develop", "", "org")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.SourceURL != "https://github.com/org/repo@develop" {
		t.Errorf("expected source URL with branch, got %q", result.SourceURL)
	}
}

func TestInstallSkillFromGitHub_URLBranchAndPath(t *testing.T) {
	repo := &svcMockRepo{}
	stor := &svcMockStorage{}
	svc := newTestServiceWithPackager(repo, stor, nil)

	// The mock git clone creates SKILL.md at the repo root, so path sub-dir
	// must also contain SKILL.md. Override gitCloneFn to place it under the path.
	svc.packager.gitCloneFn = func(_ context.Context, url, branch, targetDir string) error {
		skillDir := filepath.Join(targetDir, "skills", "my-skill")
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			return err
		}
		content := "---\nslug: my-skill\nname: My Skill\n---\n# My Skill\nA test skill."
		return os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644)
	}

	result, err := svc.InstallSkillFromGitHub(context.Background(), 1, 2, 3, "https://github.com/org/repo", "main", "skills/my-skill", "user")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "https://github.com/org/repo@main#skills/my-skill"
	if result.SourceURL != expected {
		t.Errorf("expected source URL %q, got %q", expected, result.SourceURL)
	}
}

// ---------------------------------------------------------------------------
// Tests: UpdateSkill
// ---------------------------------------------------------------------------

func TestUpdateSkill_Success(t *testing.T) {
	repo := &svcMockRepo{
		getInstalledSkillFn: func(_ context.Context, id int64) (*extension.InstalledSkill, error) {
			return &extension.InstalledSkill{
				ID:             id,
				OrganizationID: 1,
				IsEnabled:      true,
			}, nil
		},
		updateInstalledSkillFn: func(_ context.Context, skill *extension.InstalledSkill) error {
			if skill.IsEnabled != false {
				t.Error("expected IsEnabled to be false after update")
			}
			if skill.PinnedVersion == nil || *skill.PinnedVersion != 3 {
				t.Errorf("expected pinned version 3, got %v", skill.PinnedVersion)
			}
			return nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	result, err := svc.UpdateSkill(context.Background(), 1, 0, 10, 100, "admin", ptrBool(false), ptrInt(3))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsEnabled != false {
		t.Error("expected IsEnabled=false")
	}
	if result.PinnedVersion == nil || *result.PinnedVersion != 3 {
		t.Errorf("expected pinned version 3, got %v", result.PinnedVersion)
	}
}

func TestUpdateSkill_IDORDifferentOrg(t *testing.T) {
	repo := &svcMockRepo{
		getInstalledSkillFn: func(_ context.Context, id int64) (*extension.InstalledSkill, error) {
			return &extension.InstalledSkill{
				ID:             id,
				OrganizationID: 99,
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	_, err := svc.UpdateSkill(context.Background(), 1, 0, 10, 100, "admin", ptrBool(true), nil)
	if err == nil {
		t.Fatal("expected IDOR error, got nil")
	}
}

// ---------------------------------------------------------------------------
// Tests: UninstallSkill
// ---------------------------------------------------------------------------

func TestUninstallSkill_Success(t *testing.T) {
	deleteCalled := false
	repo := &svcMockRepo{
		getInstalledSkillFn: func(_ context.Context, id int64) (*extension.InstalledSkill, error) {
			return &extension.InstalledSkill{
				ID:             id,
				OrganizationID: 1,
			}, nil
		},
		deleteInstalledSkillFn: func(_ context.Context, id int64) error {
			deleteCalled = true
			if id != 10 {
				t.Errorf("expected id 10, got %d", id)
			}
			return nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	err := svc.UninstallSkill(context.Background(), 1, 0, 10, 100, "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleteCalled {
		t.Error("repo.DeleteInstalledSkill was not called")
	}
}

func TestUninstallSkill_IDORDifferentOrg(t *testing.T) {
	repo := &svcMockRepo{
		getInstalledSkillFn: func(_ context.Context, id int64) (*extension.InstalledSkill, error) {
			return &extension.InstalledSkill{
				ID:             id,
				OrganizationID: 99,
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	err := svc.UninstallSkill(context.Background(), 1, 0, 10, 100, "admin")
	if err == nil {
		t.Fatal("expected IDOR error, got nil")
	}
}

// ---------------------------------------------------------------------------
// Tests: InstallMcpFromMarket
// ---------------------------------------------------------------------------

func TestInstallMcpFromMarket_SuccessWithEncryption(t *testing.T) {
	enc := crypto.NewEncryptor("test-secret-key-1234567890123456")
	marketItemID := int64(50)

	repo := &svcMockRepo{
		getMcpMarketItemFn: func(_ context.Context, id int64) (*extension.McpMarketItem, error) {
			return &extension.McpMarketItem{
				ID:            id,
				Name:          "test-mcp",
				Slug:          "test-mcp",
				TransportType: "stdio",
				Command:       "npx",
				DefaultArgs:   json.RawMessage(`["-y","@test/mcp-server"]`),
				IsActive:      true,
			}, nil
		},
		createInstalledMcpServerFn: func(_ context.Context, server *extension.InstalledMcpServer) error {
			if server.Name != "test-mcp" {
				t.Errorf("expected name 'test-mcp', got %q", server.Name)
			}
			if server.Slug != "test-mcp" {
				t.Errorf("expected slug 'test-mcp', got %q", server.Slug)
			}
			if server.Command != "npx" {
				t.Errorf("expected command 'npx', got %q", server.Command)
			}
			if server.MarketItemID == nil || *server.MarketItemID != marketItemID {
				t.Errorf("expected market_item_id %d, got %v", marketItemID, server.MarketItemID)
			}
			// Verify env vars are encrypted (not plain text)
			if len(server.EnvVars) == 0 {
				t.Error("expected encrypted env vars to be set")
			}
			var envMap map[string]string
			if err := json.Unmarshal(server.EnvVars, &envMap); err != nil {
				t.Fatalf("failed to unmarshal env vars: %v", err)
			}
			// The value should be encrypted (not the original plain text)
			if envMap["API_KEY"] == "secret123" {
				t.Error("env var API_KEY should be encrypted, not plain text")
			}
			// Verify it can be decrypted back
			decrypted, err := enc.Decrypt(envMap["API_KEY"])
			if err != nil {
				t.Fatalf("failed to decrypt env var: %v", err)
			}
			if decrypted != "secret123" {
				t.Errorf("expected decrypted value 'secret123', got %q", decrypted)
			}
			return nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, enc)

	envVars := map[string]string{"API_KEY": "secret123"}
	result, err := svc.InstallMcpFromMarket(context.Background(), 1, 2, 3, marketItemID, envVars, "org")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsEnabled {
		t.Error("expected IsEnabled=true")
	}
}

func TestInstallMcpFromMarket_InvalidScope(t *testing.T) {
	svc := newTestService(&svcMockRepo{}, &svcMockStorage{}, nil)

	_, err := svc.InstallMcpFromMarket(context.Background(), 1, 2, 3, 50, nil, "invalid")
	if err == nil {
		t.Fatal("expected error for invalid scope, got nil")
	}
}

// ---------------------------------------------------------------------------
// Tests: InstallCustomMcpServer
// ---------------------------------------------------------------------------

func TestInstallCustomMcpServer_SuccessWithEnvVars(t *testing.T) {
	enc := crypto.NewEncryptor("test-secret-key-1234567890123456")
	var captured *extension.InstalledMcpServer
	repo := &svcMockRepo{
		createInstalledMcpServerFn: func(_ context.Context, server *extension.InstalledMcpServer) error {
			captured = server
			return nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, enc)

	server := &extension.InstalledMcpServer{
		Name:          "custom-mcp",
		Slug:          "custom-mcp",
		Scope:         "user",
		TransportType: "stdio",
		Command:       "node",
		Args:          json.RawMessage(`["server.js"]`),
	}
	envVars := map[string]string{"TOKEN": "my-token"}

	result, err := svc.InstallCustomMcpServer(context.Background(), 1, 2, 3, server, envVars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.OrganizationID != 1 {
		t.Errorf("expected org_id 1, got %d", result.OrganizationID)
	}
	if result.RepositoryID != 2 {
		t.Errorf("expected repo_id 2, got %d", result.RepositoryID)
	}
	if result.InstalledBy == nil || *result.InstalledBy != 3 {
		t.Errorf("expected installed_by 3, got %v", result.InstalledBy)
	}
	if !result.IsEnabled {
		t.Error("expected IsEnabled=true")
	}
	if captured == nil {
		t.Fatal("repo.CreateInstalledMcpServer was not called")
	}
	if len(captured.EnvVars) == 0 {
		t.Error("expected env vars to be set")
	}
}

func TestInstallCustomMcpServer_InvalidScope(t *testing.T) {
	svc := newTestService(&svcMockRepo{}, &svcMockStorage{}, nil)

	server := &extension.InstalledMcpServer{
		Scope: "bad",
	}
	_, err := svc.InstallCustomMcpServer(context.Background(), 1, 2, 3, server, nil)
	if err == nil {
		t.Fatal("expected error for invalid scope, got nil")
	}
}

// ---------------------------------------------------------------------------
// Tests: UpdateMcpServer
// ---------------------------------------------------------------------------

func TestUpdateMcpServer_SuccessWithEnvVarsUpdate(t *testing.T) {
	enc := crypto.NewEncryptor("test-secret-key-1234567890123456")
	repo := &svcMockRepo{
		getInstalledMcpServerFn: func(_ context.Context, id int64) (*extension.InstalledMcpServer, error) {
			return &extension.InstalledMcpServer{
				ID:             id,
				OrganizationID: 1,
				IsEnabled:      true,
			}, nil
		},
		updateInstalledMcpServerFn: func(_ context.Context, server *extension.InstalledMcpServer) error {
			if server.IsEnabled != false {
				t.Error("expected IsEnabled=false after update")
			}
			if len(server.EnvVars) == 0 {
				t.Error("expected encrypted env vars")
			}
			return nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, enc)

	envVars := map[string]string{"NEW_KEY": "new-value"}
	result, err := svc.UpdateMcpServer(context.Background(), 1, 0, 10, 100, "admin", ptrBool(false), envVars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsEnabled != false {
		t.Error("expected IsEnabled=false")
	}
}

func TestUpdateMcpServer_IDORDifferentOrg(t *testing.T) {
	repo := &svcMockRepo{
		getInstalledMcpServerFn: func(_ context.Context, id int64) (*extension.InstalledMcpServer, error) {
			return &extension.InstalledMcpServer{
				ID:             id,
				OrganizationID: 99,
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	_, err := svc.UpdateMcpServer(context.Background(), 1, 0, 10, 100, "admin", ptrBool(true), nil)
	if err == nil {
		t.Fatal("expected IDOR error, got nil")
	}
}

// ---------------------------------------------------------------------------
// Tests: UninstallMcpServer
// ---------------------------------------------------------------------------

func TestUninstallMcpServer_Success(t *testing.T) {
	deleteCalled := false
	repo := &svcMockRepo{
		getInstalledMcpServerFn: func(_ context.Context, id int64) (*extension.InstalledMcpServer, error) {
			return &extension.InstalledMcpServer{
				ID:             id,
				OrganizationID: 1,
			}, nil
		},
		deleteInstalledMcpServerFn: func(_ context.Context, id int64) error {
			deleteCalled = true
			if id != 10 {
				t.Errorf("expected id 10, got %d", id)
			}
			return nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	err := svc.UninstallMcpServer(context.Background(), 1, 0, 10, 100, "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleteCalled {
		t.Error("repo.DeleteInstalledMcpServer was not called")
	}
}

func TestUninstallMcpServer_IDORDifferentOrg(t *testing.T) {
	repo := &svcMockRepo{
		getInstalledMcpServerFn: func(_ context.Context, id int64) (*extension.InstalledMcpServer, error) {
			return &extension.InstalledMcpServer{
				ID:             id,
				OrganizationID: 99,
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	err := svc.UninstallMcpServer(context.Background(), 1, 0, 10, 100, "admin")
	if err == nil {
		t.Fatal("expected IDOR error, got nil")
	}
}

// ---------------------------------------------------------------------------
// Tests: GetEffectiveMcpServers
// ---------------------------------------------------------------------------

func TestGetEffectiveMcpServers_SuccessWithDecryptedEnvVars(t *testing.T) {
	enc := crypto.NewEncryptor("test-secret-key-1234567890123456")

	// Pre-encrypt a value
	encryptedVal, err := enc.Encrypt("my-secret")
	if err != nil {
		t.Fatalf("failed to encrypt test value: %v", err)
	}
	envJSON, _ := json.Marshal(map[string]string{"API_KEY": encryptedVal})

	repo := &svcMockRepo{
		getEffectiveMcpServersFn: func(_ context.Context, orgID, userID, repoID int64) ([]*extension.InstalledMcpServer, error) {
			return []*extension.InstalledMcpServer{
				{
					ID:             1,
					OrganizationID: orgID,
					Slug:           "server-1",
					EnvVars:        json.RawMessage(envJSON),
				},
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, enc)

	servers, err := svc.GetEffectiveMcpServers(context.Background(), 1, 2, 3, "claude-code")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(servers))
	}

	var envMap map[string]string
	if err := json.Unmarshal(servers[0].EnvVars, &envMap); err != nil {
		t.Fatalf("failed to unmarshal env vars: %v", err)
	}
	if envMap["API_KEY"] != "my-secret" {
		t.Errorf("expected decrypted value 'my-secret', got %q", envMap["API_KEY"])
	}
}

func TestGetEffectiveMcpServers_DecryptFailureKeepsOriginal(t *testing.T) {
	enc := crypto.NewEncryptor("test-secret-key-1234567890123456")

	// Store a value that is NOT valid encrypted text
	envJSON, _ := json.Marshal(map[string]string{"API_KEY": "not-encrypted-value"})

	repo := &svcMockRepo{
		getEffectiveMcpServersFn: func(_ context.Context, orgID, userID, repoID int64) ([]*extension.InstalledMcpServer, error) {
			return []*extension.InstalledMcpServer{
				{
					ID:             1,
					OrganizationID: orgID,
					Slug:           "server-1",
					EnvVars:        json.RawMessage(envJSON),
				},
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, enc)

	servers, err := svc.GetEffectiveMcpServers(context.Background(), 1, 2, 3, "claude-code")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(servers))
	}

	var envMap map[string]string
	if err := json.Unmarshal(servers[0].EnvVars, &envMap); err != nil {
		t.Fatalf("failed to unmarshal env vars: %v", err)
	}
	// When decryption fails, the original value should be kept
	if envMap["API_KEY"] != "not-encrypted-value" {
		t.Errorf("expected original value 'not-encrypted-value', got %q", envMap["API_KEY"])
	}
}

// ---------------------------------------------------------------------------
// Tests: GetEffectiveSkills
// ---------------------------------------------------------------------------

func TestGetEffectiveSkills_Success(t *testing.T) {
	repo := &svcMockRepo{
		getEffectiveSkillsFn: func(_ context.Context, orgID, userID, repoID int64) ([]*extension.InstalledSkill, error) {
			return []*extension.InstalledSkill{
				{
					ID:            1,
					Slug:          "skill-a",
					InstallSource: "github",
					ContentSha:    "sha-abc",
					StorageKey:    "skills/skill-a/v1.tar.gz",
					PackageSize:   2048,
				},
			}, nil
		},
	}
	stor := &svcMockStorage{
		getURLFn: func(_ context.Context, key string, expiry time.Duration) (string, error) {
			if expiry != presignedURLExpiry {
				t.Errorf("expected expiry %v, got %v", presignedURLExpiry, expiry)
			}
			return "https://cdn.example.com/" + key + "?token=abc", nil
		},
	}
	svc := newTestService(repo, stor, nil)

	resolved, err := svc.GetEffectiveSkills(context.Background(), 1, 2, 3, "claude-code")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved) != 1 {
		t.Fatalf("expected 1 resolved skill, got %d", len(resolved))
	}
	r := resolved[0]
	if r.Slug != "skill-a" {
		t.Errorf("expected slug 'skill-a', got %q", r.Slug)
	}
	if r.ContentSha != "sha-abc" {
		t.Errorf("expected sha 'sha-abc', got %q", r.ContentSha)
	}
	if r.DownloadURL != "https://cdn.example.com/skills/skill-a/v1.tar.gz?token=abc" {
		t.Errorf("unexpected download URL: %q", r.DownloadURL)
	}
	if r.PackageSize != 2048 {
		t.Errorf("expected package size 2048, got %d", r.PackageSize)
	}
	if r.TargetDir != "skills/skill-a" {
		t.Errorf("expected target dir 'skills/skill-a', got %q", r.TargetDir)
	}
}

func TestGetEffectiveSkills_SkipsEmptySHA(t *testing.T) {
	repo := &svcMockRepo{
		getEffectiveSkillsFn: func(_ context.Context, orgID, userID, repoID int64) ([]*extension.InstalledSkill, error) {
			return []*extension.InstalledSkill{
				{
					ID:            1,
					Slug:          "no-sha",
					InstallSource: "github",
					ContentSha:    "", // empty SHA
					StorageKey:    "skills/no-sha/v1.tar.gz",
				},
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	resolved, err := svc.GetEffectiveSkills(context.Background(), 1, 2, 3, "claude-code")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved) != 0 {
		t.Errorf("expected 0 resolved skills (empty SHA), got %d", len(resolved))
	}
}

func TestGetEffectiveSkills_SkipsEmptyStorageKey(t *testing.T) {
	repo := &svcMockRepo{
		getEffectiveSkillsFn: func(_ context.Context, orgID, userID, repoID int64) ([]*extension.InstalledSkill, error) {
			return []*extension.InstalledSkill{
				{
					ID:            1,
					Slug:          "no-key",
					InstallSource: "github",
					ContentSha:    "sha-abc",
					StorageKey:    "", // empty key
				},
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	resolved, err := svc.GetEffectiveSkills(context.Background(), 1, 2, 3, "claude-code")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved) != 0 {
		t.Errorf("expected 0 resolved skills (empty storage key), got %d", len(resolved))
	}
}

func TestGetEffectiveSkills_SkipsWhenPresignedURLFails(t *testing.T) {
	repo := &svcMockRepo{
		getEffectiveSkillsFn: func(_ context.Context, orgID, userID, repoID int64) ([]*extension.InstalledSkill, error) {
			return []*extension.InstalledSkill{
				{
					ID:            1,
					Slug:          "fail-url",
					InstallSource: "github",
					ContentSha:    "sha-abc",
					StorageKey:    "skills/fail-url/v1.tar.gz",
				},
			}, nil
		},
	}
	stor := &svcMockStorage{
		getURLFn: func(_ context.Context, key string, expiry time.Duration) (string, error) {
			return "", errors.New("storage unavailable")
		},
	}
	svc := newTestService(repo, stor, nil)

	resolved, err := svc.GetEffectiveSkills(context.Background(), 1, 2, 3, "claude-code")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved) != 0 {
		t.Errorf("expected 0 resolved skills (URL failure), got %d", len(resolved))
	}
}

// ---------------------------------------------------------------------------
// Tests: encryptEnvVars
// ---------------------------------------------------------------------------

func TestEncryptEnvVars_CryptoNil_StoresPlainJSON(t *testing.T) {
	svc := newTestService(&svcMockRepo{}, &svcMockStorage{}, nil)

	vars := map[string]string{"KEY": "value123"}
	data, err := svc.encryptEnvVars(vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]string
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if result["KEY"] != "value123" {
		t.Errorf("expected plain 'value123', got %q", result["KEY"])
	}
}

func TestEncryptEnvVars_CryptoPresent_EncryptsEachValue(t *testing.T) {
	enc := crypto.NewEncryptor("test-secret-key-1234567890123456")
	svc := newTestService(&svcMockRepo{}, &svcMockStorage{}, enc)

	vars := map[string]string{
		"KEY_A": "value-a",
		"KEY_B": "value-b",
	}
	data, err := svc.encryptEnvVars(vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]string
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Values should be encrypted (not plain text)
	if result["KEY_A"] == "value-a" {
		t.Error("KEY_A should be encrypted, not plain text")
	}
	if result["KEY_B"] == "value-b" {
		t.Error("KEY_B should be encrypted, not plain text")
	}

	// Verify decryption produces original values
	decA, err := enc.Decrypt(result["KEY_A"])
	if err != nil {
		t.Fatalf("failed to decrypt KEY_A: %v", err)
	}
	if decA != "value-a" {
		t.Errorf("expected 'value-a', got %q", decA)
	}

	decB, err := enc.Decrypt(result["KEY_B"])
	if err != nil {
		t.Fatalf("failed to decrypt KEY_B: %v", err)
	}
	if decB != "value-b" {
		t.Errorf("expected 'value-b', got %q", decB)
	}
}

// ---------------------------------------------------------------------------
// Tests: decryptServerEnvVars
// ---------------------------------------------------------------------------

func TestDecryptServerEnvVars_CryptoNil_ReturnsNil(t *testing.T) {
	svc := newTestService(&svcMockRepo{}, &svcMockStorage{}, nil)

	server := &extension.InstalledMcpServer{
		EnvVars: json.RawMessage(`{"KEY":"encrypted-val"}`),
	}
	err := svc.decryptServerEnvVars(server)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// EnvVars should remain unchanged (no-op)
	var envMap map[string]string
	if err := json.Unmarshal(server.EnvVars, &envMap); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if envMap["KEY"] != "encrypted-val" {
		t.Errorf("expected original value, got %q", envMap["KEY"])
	}
}

func TestDecryptServerEnvVars_EmptyEnvVars_ReturnsNil(t *testing.T) {
	enc := crypto.NewEncryptor("test-secret-key-1234567890123456")
	svc := newTestService(&svcMockRepo{}, &svcMockStorage{}, enc)

	server := &extension.InstalledMcpServer{
		EnvVars: nil,
	}
	err := svc.decryptServerEnvVars(server)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDecryptServerEnvVars_EmptyJSONEnvVars_ReturnsNil(t *testing.T) {
	enc := crypto.NewEncryptor("test-secret-key-1234567890123456")
	svc := newTestService(&svcMockRepo{}, &svcMockStorage{}, enc)

	server := &extension.InstalledMcpServer{
		EnvVars: json.RawMessage{},
	}
	err := svc.decryptServerEnvVars(server)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDecryptServerEnvVars_DecryptFailureKeepsOriginal(t *testing.T) {
	enc := crypto.NewEncryptor("test-secret-key-1234567890123456")
	svc := newTestService(&svcMockRepo{}, &svcMockStorage{}, enc)

	// Store a value that is NOT valid encrypted text
	envJSON, _ := json.Marshal(map[string]string{"KEY": "plain-text-not-encrypted"})
	server := &extension.InstalledMcpServer{
		EnvVars: json.RawMessage(envJSON),
	}

	err := svc.decryptServerEnvVars(server)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var envMap map[string]string
	if err := json.Unmarshal(server.EnvVars, &envMap); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	// When decryption fails, the original value is preserved
	if envMap["KEY"] != "plain-text-not-encrypted" {
		t.Errorf("expected original value preserved, got %q", envMap["KEY"])
	}
}

// ---------------------------------------------------------------------------
// Tests: NewService
// ---------------------------------------------------------------------------

func TestNewService(t *testing.T) {
	repo := &svcMockRepo{}
	stor := &svcMockStorage{}
	enc := crypto.NewEncryptor("test-secret-key-1234567890123456")

	svc := NewService(repo, stor, enc)
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	if svc.repo != repo {
		t.Error("repo not set correctly")
	}
	if svc.storage != stor {
		t.Error("storage not set correctly")
	}
	if svc.crypto != enc {
		t.Error("crypto not set correctly")
	}
}

// ---------------------------------------------------------------------------
// Tests: ListSkillRegistries
// ---------------------------------------------------------------------------

func TestListSkillRegistries_Success(t *testing.T) {
	called := false
	repo := &svcMockRepo{
		listSkillRegistriesFn: func(_ context.Context, orgID *int64) ([]*extension.SkillRegistry, error) {
			called = true
			if orgID == nil || *orgID != 42 {
				t.Errorf("expected orgID 42, got %v", orgID)
			}
			return []*extension.SkillRegistry{
				{ID: 1, RepositoryURL: "https://github.com/org/repo1"},
				{ID: 2, RepositoryURL: "https://github.com/org/repo2"},
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	result, err := svc.ListSkillRegistries(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("repo.ListSkillRegistries was not called")
	}
	if len(result) != 2 {
		t.Errorf("expected 2 sources, got %d", len(result))
	}
}

func TestListSkillRegistries_Error(t *testing.T) {
	repo := &svcMockRepo{
		listSkillRegistriesFn: func(_ context.Context, orgID *int64) ([]*extension.SkillRegistry, error) {
			return nil, errors.New("db connection lost")
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	_, err := svc.ListSkillRegistries(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// Tests: ListMarketSkills
// ---------------------------------------------------------------------------

func TestListMarketSkills_Success(t *testing.T) {
	called := false
	repo := &svcMockRepo{
		listSkillMarketItemsFn: func(_ context.Context, orgID *int64, query string, category string) ([]*extension.SkillMarketItem, error) {
			called = true
			if orgID == nil || *orgID != 10 {
				t.Errorf("expected orgID 10, got %v", orgID)
			}
			if query != "search" {
				t.Errorf("expected query 'search', got %q", query)
			}
			if category != "dev" {
				t.Errorf("expected category 'dev', got %q", category)
			}
			return []*extension.SkillMarketItem{
				{ID: 1, Slug: "skill-1"},
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	result, err := svc.ListMarketSkills(context.Background(), 10, "search", "dev")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("repo.ListSkillMarketItems was not called")
	}
	if len(result) != 1 {
		t.Errorf("expected 1 item, got %d", len(result))
	}
}

func TestListMarketSkills_Error(t *testing.T) {
	repo := &svcMockRepo{
		listSkillMarketItemsFn: func(_ context.Context, orgID *int64, query string, category string) ([]*extension.SkillMarketItem, error) {
			return nil, errors.New("query failed")
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	_, err := svc.ListMarketSkills(context.Background(), 1, "", "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// Tests: ListMarketMcpServers
// ---------------------------------------------------------------------------

func TestListMarketMcpServers_Success(t *testing.T) {
	called := false
	repo := &svcMockRepo{
		listMcpMarketItemsFn: func(_ context.Context, query string, category string, limit, offset int) ([]*extension.McpMarketItem, int64, error) {
			called = true
			if query != "mcp" {
				t.Errorf("expected query 'mcp', got %q", query)
			}
			return []*extension.McpMarketItem{
				{ID: 1, Slug: "mcp-1"},
			}, 1, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	result, total, err := svc.ListMarketMcpServers(context.Background(), "mcp", "tools", 50, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("repo.ListMcpMarketItems was not called")
	}
	if len(result) != 1 {
		t.Errorf("expected 1 item, got %d", len(result))
	}
	if total != 1 {
		t.Errorf("expected total 1, got %d", total)
	}
}

func TestListMarketMcpServers_Error(t *testing.T) {
	repo := &svcMockRepo{
		listMcpMarketItemsFn: func(_ context.Context, query string, category string, limit, offset int) ([]*extension.McpMarketItem, int64, error) {
			return nil, 0, errors.New("market unavailable")
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	_, _, err := svc.ListMarketMcpServers(context.Background(), "", "", 50, 0)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// Tests: ListRepoMcpServers (scope conversion)
// ---------------------------------------------------------------------------

func TestListRepoMcpServers_ScopeAllConvertsToEmpty(t *testing.T) {
	var capturedScope string
	repo := &svcMockRepo{
		listInstalledMcpServersFn: func(_ context.Context, orgID, repoID int64, scope string) ([]*extension.InstalledMcpServer, error) {
			capturedScope = scope
			return nil, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	_, err := svc.ListRepoMcpServers(context.Background(), 1, 2, 100, "all")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedScope != "" {
		t.Errorf("expected empty scope, got %q", capturedScope)
	}
}

func TestListRepoMcpServers_ScopeOrgPassesThrough(t *testing.T) {
	var capturedScope string
	repo := &svcMockRepo{
		listInstalledMcpServersFn: func(_ context.Context, orgID, repoID int64, scope string) ([]*extension.InstalledMcpServer, error) {
			capturedScope = scope
			return nil, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	_, err := svc.ListRepoMcpServers(context.Background(), 1, 2, 100, "org")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedScope != "org" {
		t.Errorf("expected scope 'org', got %q", capturedScope)
	}
}

// ---------------------------------------------------------------------------
// Tests: CreateSkillRegistry (repo error)
// ---------------------------------------------------------------------------

func TestCreateSkillRegistry_RepoError(t *testing.T) {
	repo := &svcMockRepo{
		createSkillRegistryFn: func(_ context.Context, _ *extension.SkillRegistry) error {
			return errors.New("unique constraint violation")
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	_, err := svc.CreateSkillRegistry(context.Background(), 1, CreateSkillRegistryInput{
		RepositoryURL: "https://github.com/org/repo",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, err) { // just check non-nil
		t.Errorf("unexpected error type: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Tests: GetEffectiveMcpServers (repo error + empty list)
// ---------------------------------------------------------------------------

func TestGetEffectiveMcpServers_RepoError(t *testing.T) {
	repo := &svcMockRepo{
		getEffectiveMcpServersFn: func(_ context.Context, orgID, userID, repoID int64) ([]*extension.InstalledMcpServer, error) {
			return nil, errors.New("db timeout")
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	_, err := svc.GetEffectiveMcpServers(context.Background(), 1, 2, 3, "claude-code")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetEffectiveMcpServers_EmptyList(t *testing.T) {
	repo := &svcMockRepo{
		getEffectiveMcpServersFn: func(_ context.Context, orgID, userID, repoID int64) ([]*extension.InstalledMcpServer, error) {
			return []*extension.InstalledMcpServer{}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	servers, err := svc.GetEffectiveMcpServers(context.Background(), 1, 2, 3, "claude-code")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(servers) != 0 {
		t.Errorf("expected 0 servers, got %d", len(servers))
	}
}

// ---------------------------------------------------------------------------
// Tests: GetEffectiveSkills (repo error + market source + mixed)
// ---------------------------------------------------------------------------

func TestGetEffectiveSkills_RepoError(t *testing.T) {
	repo := &svcMockRepo{
		getEffectiveSkillsFn: func(_ context.Context, orgID, userID, repoID int64) ([]*extension.InstalledSkill, error) {
			return nil, errors.New("connection refused")
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	_, err := svc.GetEffectiveSkills(context.Background(), 1, 2, 3, "claude-code")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetEffectiveSkills_MarketSourceUsesMarketItem(t *testing.T) {
	// When install_source=market and pinned_version=nil, sha/storageKey come from MarketItem
	repo := &svcMockRepo{
		getEffectiveSkillsFn: func(_ context.Context, orgID, userID, repoID int64) ([]*extension.InstalledSkill, error) {
			marketItemID := int64(100)
			return []*extension.InstalledSkill{
				{
					ID:            1,
					Slug:          "market-skill",
					InstallSource: "market",
					MarketItemID:  &marketItemID,
					ContentSha:    "old-sha",     // should be overridden by MarketItem
					StorageKey:    "old-key",      // should be overridden by MarketItem
					PackageSize:   100,            // should be overridden by MarketItem
					PinnedVersion: nil,            // not pinned → use MarketItem
					MarketItem: &extension.SkillMarketItem{
						ID:          100,
						ContentSha:  "market-sha-latest",
						StorageKey:  "market/skills/latest.tar.gz",
						PackageSize: 4096,
					},
				},
			}, nil
		},
	}
	stor := &svcMockStorage{
		getURLFn: func(_ context.Context, key string, _ time.Duration) (string, error) {
			return "https://cdn.example.com/" + key + "?signed=1", nil
		},
	}
	svc := newTestService(repo, stor, nil)

	resolved, err := svc.GetEffectiveSkills(context.Background(), 1, 2, 3, "claude-code")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved) != 1 {
		t.Fatalf("expected 1 resolved skill, got %d", len(resolved))
	}
	r := resolved[0]
	if r.ContentSha != "market-sha-latest" {
		t.Errorf("expected market SHA 'market-sha-latest', got %q", r.ContentSha)
	}
	if r.DownloadURL != "https://cdn.example.com/market/skills/latest.tar.gz?signed=1" {
		t.Errorf("expected market download URL, got %q", r.DownloadURL)
	}
	if r.PackageSize != 4096 {
		t.Errorf("expected market package size 4096, got %d", r.PackageSize)
	}
}

func TestGetEffectiveSkills_MixedValidAndInvalid(t *testing.T) {
	repo := &svcMockRepo{
		getEffectiveSkillsFn: func(_ context.Context, orgID, userID, repoID int64) ([]*extension.InstalledSkill, error) {
			return []*extension.InstalledSkill{
				{
					ID:            1,
					Slug:          "valid-skill",
					InstallSource: "github",
					ContentSha:    "sha-valid",
					StorageKey:    "skills/valid/v1.tar.gz",
					PackageSize:   512,
				},
				{
					ID:            2,
					Slug:          "no-sha-skill",
					InstallSource: "github",
					ContentSha:    "",
					StorageKey:    "skills/no-sha/v1.tar.gz",
				},
				{
					ID:            3,
					Slug:          "no-key-skill",
					InstallSource: "github",
					ContentSha:    "sha-exists",
					StorageKey:    "",
				},
				{
					ID:            4,
					Slug:          "url-fail-skill",
					InstallSource: "github",
					ContentSha:    "sha-fail",
					StorageKey:    "skills/fail/v1.tar.gz",
					PackageSize:   256,
				},
				{
					ID:            5,
					Slug:          "another-valid",
					InstallSource: "github",
					ContentSha:    "sha-another",
					StorageKey:    "skills/another/v1.tar.gz",
					PackageSize:   1024,
				},
			}, nil
		},
	}
	stor := &svcMockStorage{
		getURLFn: func(_ context.Context, key string, _ time.Duration) (string, error) {
			if key == "skills/fail/v1.tar.gz" {
				return "", errors.New("presign failed")
			}
			return "https://cdn.example.com/" + key + "?signed=1", nil
		},
	}
	svc := newTestService(repo, stor, nil)

	resolved, err := svc.GetEffectiveSkills(context.Background(), 1, 2, 3, "claude-code")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Only valid-skill and another-valid should be resolved
	if len(resolved) != 2 {
		t.Fatalf("expected 2 resolved skills (skipping no-sha, no-key, url-fail), got %d", len(resolved))
	}
	if resolved[0].Slug != "valid-skill" {
		t.Errorf("expected first resolved slug 'valid-skill', got %q", resolved[0].Slug)
	}
	if resolved[1].Slug != "another-valid" {
		t.Errorf("expected second resolved slug 'another-valid', got %q", resolved[1].Slug)
	}
}

// ---------------------------------------------------------------------------
// Tests: InstallMcpFromMarket (market item not found)
// ---------------------------------------------------------------------------

func TestInstallMcpFromMarket_MarketItemNotFound(t *testing.T) {
	repo := &svcMockRepo{
		getMcpMarketItemFn: func(_ context.Context, id int64) (*extension.McpMarketItem, error) {
			return nil, fmt.Errorf("record not found")
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	_, err := svc.InstallMcpFromMarket(context.Background(), 1, 2, 3, 999, nil, "org")
	if err == nil {
		t.Fatal("expected error for missing market item, got nil")
	}
}

// ---------------------------------------------------------------------------
// Tests: UpdateMcpServer (nil enabled)
// ---------------------------------------------------------------------------

func TestUpdateMcpServer_NilEnabled(t *testing.T) {
	repo := &svcMockRepo{
		getInstalledMcpServerFn: func(_ context.Context, id int64) (*extension.InstalledMcpServer, error) {
			return &extension.InstalledMcpServer{
				ID:             id,
				OrganizationID: 1,
				IsEnabled:      true,
			}, nil
		},
		updateInstalledMcpServerFn: func(_ context.Context, server *extension.InstalledMcpServer) error {
			// enabled=nil should not change IsEnabled
			if server.IsEnabled != true {
				t.Errorf("expected IsEnabled to remain true, got %v", server.IsEnabled)
			}
			return nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	result, err := svc.UpdateMcpServer(context.Background(), 1, 0, 10, 100, "admin", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsEnabled != true {
		t.Error("expected IsEnabled to remain true")
	}
}

// ---------------------------------------------------------------------------
// Tests: UpdateSkill (nil fields)
// ---------------------------------------------------------------------------

func TestUpdateSkill_NilFields(t *testing.T) {
	pinnedV := 5
	repo := &svcMockRepo{
		getInstalledSkillFn: func(_ context.Context, id int64) (*extension.InstalledSkill, error) {
			return &extension.InstalledSkill{
				ID:             id,
				OrganizationID: 1,
				IsEnabled:      true,
				PinnedVersion:  &pinnedV,
			}, nil
		},
		updateInstalledSkillFn: func(_ context.Context, skill *extension.InstalledSkill) error {
			// Both nil → nothing should change
			if skill.IsEnabled != true {
				t.Errorf("expected IsEnabled to remain true, got %v", skill.IsEnabled)
			}
			if skill.PinnedVersion == nil || *skill.PinnedVersion != 5 {
				t.Errorf("expected pinned version 5, got %v", skill.PinnedVersion)
			}
			return nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	result, err := svc.UpdateSkill(context.Background(), 1, 0, 10, 100, "admin", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsEnabled != true {
		t.Error("expected IsEnabled to remain true")
	}
	if result.PinnedVersion == nil || *result.PinnedVersion != 5 {
		t.Errorf("expected pinned version 5, got %v", result.PinnedVersion)
	}
}

// ---------------------------------------------------------------------------
// Tests: SyncSkillRegistry (get error + update error)
// ---------------------------------------------------------------------------

func TestSyncSkillRegistry_GetError(t *testing.T) {
	repo := &svcMockRepo{
		getSkillRegistryFn: func(_ context.Context, id int64) (*extension.SkillRegistry, error) {
			return nil, errors.New("not found")
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	_, err := svc.SyncSkillRegistry(context.Background(), 1, 999)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSyncSkillRegistry_UpdateError(t *testing.T) {
	orgID := int64(1)
	repo := &svcMockRepo{
		getSkillRegistryFn: func(_ context.Context, id int64) (*extension.SkillRegistry, error) {
			return &extension.SkillRegistry{
				ID:             id,
				OrganizationID: &orgID,
				SyncStatus:     "pending",
			}, nil
		},
		updateSkillRegistryFn: func(_ context.Context, _ *extension.SkillRegistry) error {
			return errors.New("db write failed")
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	_, err := svc.SyncSkillRegistry(context.Background(), 1, 10)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// Tests: DeleteSkillRegistry (get error)
// ---------------------------------------------------------------------------

func TestDeleteSkillRegistry_GetError(t *testing.T) {
	repo := &svcMockRepo{
		getSkillRegistryFn: func(_ context.Context, id int64) (*extension.SkillRegistry, error) {
			return nil, errors.New("not found")
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	err := svc.DeleteSkillRegistry(context.Background(), 1, 999)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// Tests: InstallSkillFromMarket (create error)
// ---------------------------------------------------------------------------

func TestInstallSkillFromMarket_CreateError(t *testing.T) {
	repo := &svcMockRepo{
		getSkillMarketItemFn: func(_ context.Context, id int64) (*extension.SkillMarketItem, error) {
			return &extension.SkillMarketItem{
				ID:   id,
				Slug: "test-skill",
			}, nil
		},
		createInstalledSkillFn: func(_ context.Context, _ *extension.InstalledSkill) error {
			return errors.New("duplicate entry")
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	_, err := svc.InstallSkillFromMarket(context.Background(), 1, 2, 3, 100, "org")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// Tests: InstallMcpFromMarket (create error + encrypt error)
// ---------------------------------------------------------------------------

func TestInstallMcpFromMarket_CreateError(t *testing.T) {
	repo := &svcMockRepo{
		getMcpMarketItemFn: func(_ context.Context, id int64) (*extension.McpMarketItem, error) {
			return &extension.McpMarketItem{
				ID:            id,
				Name:          "test-mcp",
				Slug:          "test-mcp",
				TransportType: "stdio",
				Command:       "npx",
			}, nil
		},
		createInstalledMcpServerFn: func(_ context.Context, _ *extension.InstalledMcpServer) error {
			return errors.New("db insert failed")
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	_, err := svc.InstallMcpFromMarket(context.Background(), 1, 2, 3, 50, nil, "org")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// Tests: InstallCustomMcpServer (create error)
// ---------------------------------------------------------------------------

func TestInstallCustomMcpServer_CreateError(t *testing.T) {
	repo := &svcMockRepo{
		createInstalledMcpServerFn: func(_ context.Context, _ *extension.InstalledMcpServer) error {
			return errors.New("db insert failed")
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	server := &extension.InstalledMcpServer{
		Name:          "custom-mcp",
		Slug:          "custom-mcp",
		Scope:         "org",
		TransportType: "stdio",
		Command:       "node",
	}
	_, err := svc.InstallCustomMcpServer(context.Background(), 1, 2, 3, server, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// Tests: UpdateMcpServer (get error + update error)
// ---------------------------------------------------------------------------

func TestUpdateMcpServer_GetError(t *testing.T) {
	repo := &svcMockRepo{
		getInstalledMcpServerFn: func(_ context.Context, id int64) (*extension.InstalledMcpServer, error) {
			return nil, errors.New("not found")
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	_, err := svc.UpdateMcpServer(context.Background(), 1, 0, 999, 100, "admin", ptrBool(true), nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestUpdateMcpServer_UpdateError(t *testing.T) {
	repo := &svcMockRepo{
		getInstalledMcpServerFn: func(_ context.Context, id int64) (*extension.InstalledMcpServer, error) {
			return &extension.InstalledMcpServer{
				ID:             id,
				OrganizationID: 1,
				IsEnabled:      true,
			}, nil
		},
		updateInstalledMcpServerFn: func(_ context.Context, _ *extension.InstalledMcpServer) error {
			return errors.New("db write failed")
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	_, err := svc.UpdateMcpServer(context.Background(), 1, 0, 10, 100, "admin", ptrBool(false), nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// Tests: UpdateSkill (get error + update error)
// ---------------------------------------------------------------------------

func TestUpdateSkill_GetError(t *testing.T) {
	repo := &svcMockRepo{
		getInstalledSkillFn: func(_ context.Context, id int64) (*extension.InstalledSkill, error) {
			return nil, errors.New("not found")
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	_, err := svc.UpdateSkill(context.Background(), 1, 0, 999, 100, "admin", ptrBool(true), nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestUpdateSkill_UpdateError(t *testing.T) {
	repo := &svcMockRepo{
		getInstalledSkillFn: func(_ context.Context, id int64) (*extension.InstalledSkill, error) {
			return &extension.InstalledSkill{
				ID:             id,
				OrganizationID: 1,
				IsEnabled:      true,
			}, nil
		},
		updateInstalledSkillFn: func(_ context.Context, _ *extension.InstalledSkill) error {
			return errors.New("db write failed")
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	_, err := svc.UpdateSkill(context.Background(), 1, 0, 10, 100, "admin", ptrBool(false), nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// Tests: UninstallSkill (get error)
// ---------------------------------------------------------------------------

func TestUninstallSkill_GetError(t *testing.T) {
	repo := &svcMockRepo{
		getInstalledSkillFn: func(_ context.Context, id int64) (*extension.InstalledSkill, error) {
			return nil, errors.New("not found")
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	err := svc.UninstallSkill(context.Background(), 1, 0, 999, 100, "admin")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// Tests: UninstallMcpServer (get error)
// ---------------------------------------------------------------------------

func TestUninstallMcpServer_GetError(t *testing.T) {
	repo := &svcMockRepo{
		getInstalledMcpServerFn: func(_ context.Context, id int64) (*extension.InstalledMcpServer, error) {
			return nil, errors.New("not found")
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	err := svc.UninstallMcpServer(context.Background(), 1, 0, 999, 100, "admin")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// Tests: InstallSkillFromGitHub (invalid scope)
// ---------------------------------------------------------------------------

func TestInstallSkillFromGitHub_PathOnlyNoBranch(t *testing.T) {
	repo := &svcMockRepo{}
	stor := &svcMockStorage{}
	svc := newTestServiceWithPackager(repo, stor, nil)

	// Override gitCloneFn to place SKILL.md under the path sub-directory
	svc.packager.gitCloneFn = func(_ context.Context, url, branch, targetDir string) error {
		skillDir := filepath.Join(targetDir, "skills", "my-skill")
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			return err
		}
		content := "---\nslug: my-skill\nname: My Skill\n---\n# My Skill\nA test skill."
		return os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644)
	}

	result, err := svc.InstallSkillFromGitHub(context.Background(), 1, 2, 3, "https://github.com/org/repo", "", "skills/my-skill", "user")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// When branch is empty but path is given, path is appended with #
	expected := "https://github.com/org/repo#skills/my-skill"
	if result.SourceURL != expected {
		t.Errorf("expected source URL %q, got %q", expected, result.SourceURL)
	}
}

// ---------------------------------------------------------------------------
// Tests: InstallCustomMcpServer (no env vars)
// ---------------------------------------------------------------------------

func TestInstallCustomMcpServer_NoEnvVars(t *testing.T) {
	var captured *extension.InstalledMcpServer
	repo := &svcMockRepo{
		createInstalledMcpServerFn: func(_ context.Context, server *extension.InstalledMcpServer) error {
			captured = server
			return nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	server := &extension.InstalledMcpServer{
		Name:          "bare-mcp",
		Slug:          "bare-mcp",
		Scope:         "org",
		TransportType: "stdio",
		Command:       "echo",
	}
	result, err := svc.InstallCustomMcpServer(context.Background(), 1, 2, 3, server, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsEnabled {
		t.Error("expected IsEnabled=true")
	}
	if captured == nil {
		t.Fatal("repo.CreateInstalledMcpServer was not called")
	}
	if len(captured.EnvVars) != 0 {
		t.Errorf("expected no env vars, got %s", string(captured.EnvVars))
	}
}

// ---------------------------------------------------------------------------
// Tests: GetEffectiveMcpServers (nil envvars server in list)
// ---------------------------------------------------------------------------

func TestGetEffectiveMcpServers_NilEnvVarsServer(t *testing.T) {
	enc := crypto.NewEncryptor("test-secret-key-1234567890123456")
	repo := &svcMockRepo{
		getEffectiveMcpServersFn: func(_ context.Context, orgID, userID, repoID int64) ([]*extension.InstalledMcpServer, error) {
			return []*extension.InstalledMcpServer{
				{
					ID:             1,
					OrganizationID: orgID,
					Slug:           "no-env",
					EnvVars:        nil,
				},
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, enc)

	servers, err := svc.GetEffectiveMcpServers(context.Background(), 1, 2, 3, "claude-code")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(servers))
	}
}

// ---------------------------------------------------------------------------
// Tests: InstallMcpFromMarket (with empty env vars)
// ---------------------------------------------------------------------------

func TestInstallMcpFromMarket_EmptyEnvVars(t *testing.T) {
	var captured *extension.InstalledMcpServer
	repo := &svcMockRepo{
		getMcpMarketItemFn: func(_ context.Context, id int64) (*extension.McpMarketItem, error) {
			return &extension.McpMarketItem{
				ID:            id,
				Name:          "test-mcp",
				Slug:          "test-mcp",
				TransportType: "stdio",
				Command:       "npx",
				IsActive:      true,
			}, nil
		},
		createInstalledMcpServerFn: func(_ context.Context, server *extension.InstalledMcpServer) error {
			captured = server
			return nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	// Pass empty map (len=0) → env vars should not be set
	result, err := svc.InstallMcpFromMarket(context.Background(), 1, 2, 3, 50, map[string]string{}, "user")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsEnabled {
		t.Error("expected IsEnabled=true")
	}
	if captured == nil {
		t.Fatal("repo.CreateInstalledMcpServer was not called")
	}
	if len(captured.EnvVars) != 0 {
		t.Errorf("expected no env vars for empty map, got %s", string(captured.EnvVars))
	}
}

// ---------------------------------------------------------------------------
// Tests: decryptServerEnvVars (invalid JSON triggers unmarshal error)
// ---------------------------------------------------------------------------

func TestDecryptServerEnvVars_InvalidJSON(t *testing.T) {
	enc := crypto.NewEncryptor("test-secret-key-1234567890123456")
	svc := newTestService(&svcMockRepo{}, &svcMockStorage{}, enc)

	server := &extension.InstalledMcpServer{
		EnvVars: json.RawMessage(`{invalid json`),
	}
	err := svc.decryptServerEnvVars(server)
	if err == nil {
		t.Fatal("expected unmarshal error, got nil")
	}
}

// ---------------------------------------------------------------------------
// Tests: UpdateSkill — repoID mismatch
// ---------------------------------------------------------------------------

func TestUpdateSkill_RepoIDMismatch(t *testing.T) {
	repo := &svcMockRepo{
		getInstalledSkillFn: func(_ context.Context, id int64) (*extension.InstalledSkill, error) {
			return &extension.InstalledSkill{
				ID:             id,
				OrganizationID: 1,
				RepositoryID:   100,
				Scope:          "org",
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	_, err := svc.UpdateSkill(context.Background(), 1, 999, 10, 100, "admin", ptrBool(true), nil)
	if err == nil {
		t.Fatal("expected error for repoID mismatch, got nil")
	}
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("expected ErrForbidden, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Tests: UpdateSkill — org scope + non-admin role
// ---------------------------------------------------------------------------

func TestUpdateSkill_OrgScope_NonAdminRole(t *testing.T) {
	repo := &svcMockRepo{
		getInstalledSkillFn: func(_ context.Context, id int64) (*extension.InstalledSkill, error) {
			return &extension.InstalledSkill{
				ID:             id,
				OrganizationID: 1,
				RepositoryID:   2,
				Scope:          "org",
				IsEnabled:      true,
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	_, err := svc.UpdateSkill(context.Background(), 1, 2, 10, 100, "member", ptrBool(false), nil)
	if err == nil {
		t.Fatal("expected error for non-admin role on org scope, got nil")
	}
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("expected ErrForbidden, got: %v", err)
	}
}

func TestUpdateSkill_OrgScope_AdminRole(t *testing.T) {
	repo := &svcMockRepo{
		getInstalledSkillFn: func(_ context.Context, id int64) (*extension.InstalledSkill, error) {
			return &extension.InstalledSkill{
				ID:             id,
				OrganizationID: 1,
				RepositoryID:   2,
				Scope:          "org",
				IsEnabled:      true,
			}, nil
		},
		updateInstalledSkillFn: func(_ context.Context, skill *extension.InstalledSkill) error {
			return nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	result, err := svc.UpdateSkill(context.Background(), 1, 2, 10, 100, "admin", ptrBool(false), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsEnabled != false {
		t.Error("expected IsEnabled=false after admin update")
	}
}

func TestUpdateSkill_OrgScope_OwnerRole(t *testing.T) {
	repo := &svcMockRepo{
		getInstalledSkillFn: func(_ context.Context, id int64) (*extension.InstalledSkill, error) {
			return &extension.InstalledSkill{
				ID:             id,
				OrganizationID: 1,
				RepositoryID:   2,
				Scope:          "org",
				IsEnabled:      true,
			}, nil
		},
		updateInstalledSkillFn: func(_ context.Context, skill *extension.InstalledSkill) error {
			return nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	_, err := svc.UpdateSkill(context.Background(), 1, 2, 10, 100, "owner", ptrBool(false), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateSkill_UserScope_NoRoleCheck(t *testing.T) {
	repo := &svcMockRepo{
		getInstalledSkillFn: func(_ context.Context, id int64) (*extension.InstalledSkill, error) {
			return &extension.InstalledSkill{
				ID:             id,
				OrganizationID: 1,
				RepositoryID:   2,
				Scope:          "user",
				IsEnabled:      true,
				InstalledBy:    int64Ptr(100),
			}, nil
		},
		updateInstalledSkillFn: func(_ context.Context, skill *extension.InstalledSkill) error {
			return nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	// "member" role should be allowed for user-scoped skills (no role check)
	_, err := svc.UpdateSkill(context.Background(), 1, 2, 10, 100, "member", ptrBool(false), nil)
	if err != nil {
		t.Fatalf("expected no error for user scope with member role, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Tests: UninstallSkill — repoID mismatch + role checks
// ---------------------------------------------------------------------------

func TestUninstallSkill_RepoIDMismatch(t *testing.T) {
	repo := &svcMockRepo{
		getInstalledSkillFn: func(_ context.Context, id int64) (*extension.InstalledSkill, error) {
			return &extension.InstalledSkill{
				ID:             id,
				OrganizationID: 1,
				RepositoryID:   100,
				Scope:          "org",
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	err := svc.UninstallSkill(context.Background(), 1, 999, 10, 100, "admin")
	if err == nil {
		t.Fatal("expected error for repoID mismatch, got nil")
	}
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("expected ErrForbidden, got: %v", err)
	}
}

func TestUninstallSkill_OrgScope_NonAdminRole(t *testing.T) {
	repo := &svcMockRepo{
		getInstalledSkillFn: func(_ context.Context, id int64) (*extension.InstalledSkill, error) {
			return &extension.InstalledSkill{
				ID:             id,
				OrganizationID: 1,
				RepositoryID:   2,
				Scope:          "org",
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	err := svc.UninstallSkill(context.Background(), 1, 2, 10, 100, "member")
	if err == nil {
		t.Fatal("expected error for non-admin role on org scope, got nil")
	}
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("expected ErrForbidden, got: %v", err)
	}
}

func TestUninstallSkill_OrgScope_OwnerRole(t *testing.T) {
	deleteCalled := false
	repo := &svcMockRepo{
		getInstalledSkillFn: func(_ context.Context, id int64) (*extension.InstalledSkill, error) {
			return &extension.InstalledSkill{
				ID:             id,
				OrganizationID: 1,
				RepositoryID:   2,
				Scope:          "org",
			}, nil
		},
		deleteInstalledSkillFn: func(_ context.Context, id int64) error {
			deleteCalled = true
			return nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	err := svc.UninstallSkill(context.Background(), 1, 2, 10, 100, "owner")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleteCalled {
		t.Error("repo.DeleteInstalledSkill was not called")
	}
}

func TestUninstallSkill_UserScope_NoRoleCheck(t *testing.T) {
	deleteCalled := false
	repo := &svcMockRepo{
		getInstalledSkillFn: func(_ context.Context, id int64) (*extension.InstalledSkill, error) {
			return &extension.InstalledSkill{
				ID:             id,
				OrganizationID: 1,
				RepositoryID:   2,
				Scope:          "user",
				InstalledBy:    int64Ptr(100),
			}, nil
		},
		deleteInstalledSkillFn: func(_ context.Context, id int64) error {
			deleteCalled = true
			return nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	err := svc.UninstallSkill(context.Background(), 1, 2, 10, 100, "member")
	if err != nil {
		t.Fatalf("expected no error for user scope with member role, got: %v", err)
	}
	if !deleteCalled {
		t.Error("repo.DeleteInstalledSkill was not called")
	}
}

// ---------------------------------------------------------------------------
// Tests: UpdateMcpServer — repoID mismatch + role checks
// ---------------------------------------------------------------------------

func TestUpdateMcpServer_RepoIDMismatch(t *testing.T) {
	repo := &svcMockRepo{
		getInstalledMcpServerFn: func(_ context.Context, id int64) (*extension.InstalledMcpServer, error) {
			return &extension.InstalledMcpServer{
				ID:             id,
				OrganizationID: 1,
				RepositoryID:   100,
				Scope:          "org",
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	_, err := svc.UpdateMcpServer(context.Background(), 1, 999, 10, 100, "admin", ptrBool(true), nil)
	if err == nil {
		t.Fatal("expected error for repoID mismatch, got nil")
	}
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("expected ErrForbidden, got: %v", err)
	}
}

func TestUpdateMcpServer_OrgScope_NonAdminRole(t *testing.T) {
	repo := &svcMockRepo{
		getInstalledMcpServerFn: func(_ context.Context, id int64) (*extension.InstalledMcpServer, error) {
			return &extension.InstalledMcpServer{
				ID:             id,
				OrganizationID: 1,
				RepositoryID:   2,
				Scope:          "org",
				IsEnabled:      true,
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	_, err := svc.UpdateMcpServer(context.Background(), 1, 2, 10, 100, "member", ptrBool(false), nil)
	if err == nil {
		t.Fatal("expected error for non-admin role on org scope, got nil")
	}
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("expected ErrForbidden, got: %v", err)
	}
}

func TestUpdateMcpServer_OrgScope_OwnerRole(t *testing.T) {
	repo := &svcMockRepo{
		getInstalledMcpServerFn: func(_ context.Context, id int64) (*extension.InstalledMcpServer, error) {
			return &extension.InstalledMcpServer{
				ID:             id,
				OrganizationID: 1,
				RepositoryID:   2,
				Scope:          "org",
				IsEnabled:      true,
			}, nil
		},
		updateInstalledMcpServerFn: func(_ context.Context, _ *extension.InstalledMcpServer) error {
			return nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	_, err := svc.UpdateMcpServer(context.Background(), 1, 2, 10, 100, "owner", ptrBool(false), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateMcpServer_UserScope_NoRoleCheck(t *testing.T) {
	repo := &svcMockRepo{
		getInstalledMcpServerFn: func(_ context.Context, id int64) (*extension.InstalledMcpServer, error) {
			return &extension.InstalledMcpServer{
				ID:             id,
				OrganizationID: 1,
				RepositoryID:   2,
				Scope:          "user",
				IsEnabled:      true,
				InstalledBy:    int64Ptr(100),
			}, nil
		},
		updateInstalledMcpServerFn: func(_ context.Context, _ *extension.InstalledMcpServer) error {
			return nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	_, err := svc.UpdateMcpServer(context.Background(), 1, 2, 10, 100, "member", ptrBool(false), nil)
	if err != nil {
		t.Fatalf("expected no error for user scope with member role, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Tests: UninstallMcpServer — repoID mismatch + role checks
// ---------------------------------------------------------------------------

func TestUninstallMcpServer_RepoIDMismatch(t *testing.T) {
	repo := &svcMockRepo{
		getInstalledMcpServerFn: func(_ context.Context, id int64) (*extension.InstalledMcpServer, error) {
			return &extension.InstalledMcpServer{
				ID:             id,
				OrganizationID: 1,
				RepositoryID:   100,
				Scope:          "org",
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	err := svc.UninstallMcpServer(context.Background(), 1, 999, 10, 100, "admin")
	if err == nil {
		t.Fatal("expected error for repoID mismatch, got nil")
	}
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("expected ErrForbidden, got: %v", err)
	}
}

func TestUninstallMcpServer_OrgScope_NonAdminRole(t *testing.T) {
	repo := &svcMockRepo{
		getInstalledMcpServerFn: func(_ context.Context, id int64) (*extension.InstalledMcpServer, error) {
			return &extension.InstalledMcpServer{
				ID:             id,
				OrganizationID: 1,
				RepositoryID:   2,
				Scope:          "org",
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	err := svc.UninstallMcpServer(context.Background(), 1, 2, 10, 100, "member")
	if err == nil {
		t.Fatal("expected error for non-admin role on org scope, got nil")
	}
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("expected ErrForbidden, got: %v", err)
	}
}

func TestUninstallMcpServer_OrgScope_OwnerRole(t *testing.T) {
	deleteCalled := false
	repo := &svcMockRepo{
		getInstalledMcpServerFn: func(_ context.Context, id int64) (*extension.InstalledMcpServer, error) {
			return &extension.InstalledMcpServer{
				ID:             id,
				OrganizationID: 1,
				RepositoryID:   2,
				Scope:          "org",
			}, nil
		},
		deleteInstalledMcpServerFn: func(_ context.Context, id int64) error {
			deleteCalled = true
			return nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	err := svc.UninstallMcpServer(context.Background(), 1, 2, 10, 100, "owner")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleteCalled {
		t.Error("repo.DeleteInstalledMcpServer was not called")
	}
}

func TestUninstallMcpServer_UserScope_NoRoleCheck(t *testing.T) {
	deleteCalled := false
	repo := &svcMockRepo{
		getInstalledMcpServerFn: func(_ context.Context, id int64) (*extension.InstalledMcpServer, error) {
			return &extension.InstalledMcpServer{
				ID:             id,
				OrganizationID: 1,
				RepositoryID:   2,
				Scope:          "user",
				InstalledBy:    int64Ptr(100),
			}, nil
		},
		deleteInstalledMcpServerFn: func(_ context.Context, id int64) error {
			deleteCalled = true
			return nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	err := svc.UninstallMcpServer(context.Background(), 1, 2, 10, 100, "member")
	if err != nil {
		t.Fatalf("expected no error for user scope with member role, got: %v", err)
	}
	if !deleteCalled {
		t.Error("repo.DeleteInstalledMcpServer was not called")
	}
}

// ---------------------------------------------------------------------------
// Tests: DeleteSkillRegistry — platform-level source protection
// ---------------------------------------------------------------------------

func TestDeleteSkillRegistry_PlatformLevelSource(t *testing.T) {
	repo := &svcMockRepo{
		getSkillRegistryFn: func(_ context.Context, id int64) (*extension.SkillRegistry, error) {
			return &extension.SkillRegistry{
				ID:             id,
				OrganizationID: nil, // platform level
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	err := svc.DeleteSkillRegistry(context.Background(), 42, 10)
	if err == nil {
		t.Fatal("expected error for platform-level source, got nil")
	}
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("expected ErrForbidden, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Tests: InstallSkillFromGitHub — CreateInstalledSkill error + scope validation
// ---------------------------------------------------------------------------

func TestInstallSkillFromGitHub_InvalidScope(t *testing.T) {
	svc := newTestService(&svcMockRepo{}, &svcMockStorage{}, nil)

	_, err := svc.InstallSkillFromGitHub(context.Background(), 1, 2, 3, "https://github.com/org/repo", "", "", "bad")
	if err == nil {
		t.Fatal("expected error for invalid scope, got nil")
	}
	if !errors.Is(err, ErrInvalidScope) {
		t.Errorf("expected ErrInvalidScope, got: %v", err)
	}
}

func TestInstallSkillFromGitHub_CreateError(t *testing.T) {
	repo := &svcMockRepo{
		createInstalledSkillFn: func(_ context.Context, _ *extension.InstalledSkill) error {
			return errors.New("db insert failed")
		},
	}
	svc := newTestServiceWithPackager(repo, &svcMockStorage{}, nil)

	_, err := svc.InstallSkillFromGitHub(context.Background(), 1, 2, 3, "https://github.com/org/repo", "", "", "org")
	if err == nil {
		t.Fatal("expected error for create failure, got nil")
	}
	if !strings.Contains(err.Error(), "db insert failed") {
		t.Errorf("expected DB insert error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Tests: Standard error types (errors.Is matching)
// ---------------------------------------------------------------------------

func TestStandardErrorTypes(t *testing.T) {
	t.Run("validateScope_returns_ErrInvalidScope", func(t *testing.T) {
		err := validateScope("bad")
		if !errors.Is(err, ErrInvalidScope) {
			t.Errorf("expected errors.Is(err, ErrInvalidScope), got: %v", err)
		}
	})

	t.Run("SyncSkillRegistry_notfound_returns_ErrNotFound", func(t *testing.T) {
		repo := &svcMockRepo{
			getSkillRegistryFn: func(_ context.Context, id int64) (*extension.SkillRegistry, error) {
				return nil, errors.New("not found")
			},
		}
		svc := newTestService(repo, &svcMockStorage{}, nil)
		_, err := svc.SyncSkillRegistry(context.Background(), 1, 999)
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected ErrNotFound, got: %v", err)
		}
	})

	t.Run("DeleteSkillRegistry_notfound_returns_ErrNotFound", func(t *testing.T) {
		repo := &svcMockRepo{
			getSkillRegistryFn: func(_ context.Context, id int64) (*extension.SkillRegistry, error) {
				return nil, errors.New("not found")
			},
		}
		svc := newTestService(repo, &svcMockStorage{}, nil)
		err := svc.DeleteSkillRegistry(context.Background(), 1, 999)
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected ErrNotFound, got: %v", err)
		}
	})

	t.Run("UpdateSkill_notfound_returns_ErrNotFound", func(t *testing.T) {
		repo := &svcMockRepo{
			getInstalledSkillFn: func(_ context.Context, id int64) (*extension.InstalledSkill, error) {
				return nil, errors.New("not found")
			},
		}
		svc := newTestService(repo, &svcMockStorage{}, nil)
		_, err := svc.UpdateSkill(context.Background(), 1, 0, 999, 100, "admin", nil, nil)
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected ErrNotFound, got: %v", err)
		}
	})

	t.Run("UninstallSkill_notfound_returns_ErrNotFound", func(t *testing.T) {
		repo := &svcMockRepo{
			getInstalledSkillFn: func(_ context.Context, id int64) (*extension.InstalledSkill, error) {
				return nil, errors.New("not found")
			},
		}
		svc := newTestService(repo, &svcMockStorage{}, nil)
		err := svc.UninstallSkill(context.Background(), 1, 0, 999, 100, "admin")
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected ErrNotFound, got: %v", err)
		}
	})

	t.Run("UpdateMcpServer_notfound_returns_ErrNotFound", func(t *testing.T) {
		repo := &svcMockRepo{
			getInstalledMcpServerFn: func(_ context.Context, id int64) (*extension.InstalledMcpServer, error) {
				return nil, errors.New("not found")
			},
		}
		svc := newTestService(repo, &svcMockStorage{}, nil)
		_, err := svc.UpdateMcpServer(context.Background(), 1, 0, 999, 100, "admin", nil, nil)
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected ErrNotFound, got: %v", err)
		}
	})

	t.Run("UninstallMcpServer_notfound_returns_ErrNotFound", func(t *testing.T) {
		repo := &svcMockRepo{
			getInstalledMcpServerFn: func(_ context.Context, id int64) (*extension.InstalledMcpServer, error) {
				return nil, errors.New("not found")
			},
		}
		svc := newTestService(repo, &svcMockStorage{}, nil)
		err := svc.UninstallMcpServer(context.Background(), 1, 0, 999, 100, "admin")
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected ErrNotFound, got: %v", err)
		}
	})

	t.Run("InstallSkillFromMarket_notfound_returns_ErrNotFound", func(t *testing.T) {
		repo := &svcMockRepo{
			getSkillMarketItemFn: func(_ context.Context, id int64) (*extension.SkillMarketItem, error) {
				return nil, errors.New("record not found")
			},
		}
		svc := newTestService(repo, &svcMockStorage{}, nil)
		_, err := svc.InstallSkillFromMarket(context.Background(), 1, 2, 3, 999, "org")
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected ErrNotFound, got: %v", err)
		}
	})

	t.Run("InstallMcpFromMarket_notfound_returns_ErrNotFound", func(t *testing.T) {
		repo := &svcMockRepo{
			getMcpMarketItemFn: func(_ context.Context, id int64) (*extension.McpMarketItem, error) {
				return nil, errors.New("record not found")
			},
		}
		svc := newTestService(repo, &svcMockStorage{}, nil)
		_, err := svc.InstallMcpFromMarket(context.Background(), 1, 2, 3, 999, nil, "org")
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected ErrNotFound, got: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// Tests: GetEffectiveMcpServers (decrypt warning path)
// ---------------------------------------------------------------------------

func TestGetEffectiveMcpServers_DecryptWarning(t *testing.T) {
	enc := crypto.NewEncryptor("test-secret-key-1234567890123456")
	repo := &svcMockRepo{
		getEffectiveMcpServersFn: func(_ context.Context, orgID, userID, repoID int64) ([]*extension.InstalledMcpServer, error) {
			return []*extension.InstalledMcpServer{
				{
					ID:             1,
					OrganizationID: orgID,
					Slug:           "bad-json-env",
					EnvVars:        json.RawMessage(`{broken`),
				},
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, enc)

	// Should not return error; just log a warning
	servers, err := svc.GetEffectiveMcpServers(context.Background(), 1, 2, 3, "claude-code")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(servers))
	}
}

// ---------------------------------------------------------------------------
// Tests: UpdateMcpServer (encrypt error via envVars)
// ---------------------------------------------------------------------------

func TestUpdateMcpServer_EnvVarsEncryptErrorPath(t *testing.T) {
	enc := crypto.NewEncryptor("test-secret-key-1234567890123456")
	repo := &svcMockRepo{
		getInstalledMcpServerFn: func(_ context.Context, id int64) (*extension.InstalledMcpServer, error) {
			return &extension.InstalledMcpServer{
				ID:             id,
				OrganizationID: 1,
				IsEnabled:      true,
			}, nil
		},
		updateInstalledMcpServerFn: func(_ context.Context, server *extension.InstalledMcpServer) error {
			return nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, enc)

	// Valid env vars with encryption should succeed
	envVars := map[string]string{"SECRET": "value"}
	result, err := svc.UpdateMcpServer(context.Background(), 1, 0, 10, 100, "admin", nil, envVars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.EnvVars) == 0 {
		t.Error("expected env vars to be set after update")
	}
}

// ---------------------------------------------------------------------------
// Tests: UpdateMcpServer — envVars with nil crypto (development mode)
// ---------------------------------------------------------------------------

func TestUpdateMcpServer_EnvVarsNoCrypto(t *testing.T) {
	repo := &svcMockRepo{
		getInstalledMcpServerFn: func(_ context.Context, id int64) (*extension.InstalledMcpServer, error) {
			return &extension.InstalledMcpServer{
				ID:             id,
				OrganizationID: 1,
				RepositoryID:   2,
				Scope:          "org",
				IsEnabled:      true,
			}, nil
		},
		updateInstalledMcpServerFn: func(_ context.Context, server *extension.InstalledMcpServer) error {
			return nil
		},
	}
	// nil crypto = development mode, stores as-is
	svc := newTestService(repo, &svcMockStorage{}, nil)

	envVars := map[string]string{"API_KEY": "secret123"}
	result, err := svc.UpdateMcpServer(context.Background(), 1, 2, 10, 100, "admin", nil, envVars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.EnvVars) == 0 {
		t.Error("expected env vars to be set")
	}
}

// ---------------------------------------------------------------------------
// Tests: InstallMcpFromMarket — envVars with nil crypto (development mode)
// ---------------------------------------------------------------------------

func TestInstallMcpFromMarket_EnvVarsNoCrypto(t *testing.T) {
	repo := &svcMockRepo{
		getMcpMarketItemFn: func(_ context.Context, id int64) (*extension.McpMarketItem, error) {
			return &extension.McpMarketItem{
				ID:            id,
				Slug:          "test-mcp",
				Name:          "Test MCP",
				TransportType: "stdio",
				Command:       "node",
				IsActive:      true,
			}, nil
		},
		createInstalledMcpServerFn: func(_ context.Context, server *extension.InstalledMcpServer) error {
			server.ID = 1
			return nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	envVars := map[string]string{"API_KEY": "key123"}
	result, err := svc.InstallMcpFromMarket(context.Background(), 1, 2, 3, 10, envVars, "org")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.EnvVars) == 0 {
		t.Error("expected env vars to be set")
	}
}

// ---------------------------------------------------------------------------
// Tests: InstallCustomMcpServer — envVars with nil crypto (development mode)
// ---------------------------------------------------------------------------

func TestInstallCustomMcpServer_EnvVarsNoCrypto(t *testing.T) {
	repo := &svcMockRepo{
		createInstalledMcpServerFn: func(_ context.Context, server *extension.InstalledMcpServer) error {
			server.ID = 1
			return nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	server := &extension.InstalledMcpServer{
		Slug:          "custom-mcp",
		Name:          "Custom MCP",
		TransportType: "http",
		HttpURL:       "https://example.com/mcp",
		Scope:         "org",
	}
	envVars := map[string]string{"API_KEY": "key123"}
	result, err := svc.InstallCustomMcpServer(context.Background(), 1, 2, 3, server, envVars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.EnvVars) == 0 {
		t.Error("expected env vars to be set")
	}
}

// ---------------------------------------------------------------------------
// Tests: InstallSkillFromGitHub — with branch and path
// ---------------------------------------------------------------------------

func TestInstallSkillFromGitHub_WithBranchAndPath(t *testing.T) {
	var captured *extension.InstalledSkill
	repo := &svcMockRepo{
		createInstalledSkillFn: func(_ context.Context, skill *extension.InstalledSkill) error {
			skill.ID = 1
			captured = skill
			return nil
		},
	}
	svc := newTestServiceWithPackager(repo, &svcMockStorage{}, nil)

	// Override gitCloneFn to place SKILL.md under the path sub-directory
	svc.packager.gitCloneFn = func(_ context.Context, url, branch, targetDir string) error {
		skillDir := filepath.Join(targetDir, "skills", "my-skill")
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			return err
		}
		content := "---\nslug: my-skill\nname: My Skill\n---\n# My Skill\nA test skill."
		return os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644)
	}

	result, err := svc.InstallSkillFromGitHub(context.Background(), 1, 2, 3, "https://github.com/org/repo", "develop", "skills/my-skill", "org")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.SourceURL != "https://github.com/org/repo@develop#skills/my-skill" {
		t.Errorf("expected source URL with branch and path, got %q", result.SourceURL)
	}
	if captured.InstallSource != "github" {
		t.Errorf("expected install_source 'github', got %q", captured.InstallSource)
	}
}

func TestInstallSkillFromGitHub_WithBranchOnly(t *testing.T) {
	repo := &svcMockRepo{
		createInstalledSkillFn: func(_ context.Context, skill *extension.InstalledSkill) error {
			skill.ID = 1
			return nil
		},
	}
	svc := newTestServiceWithPackager(repo, &svcMockStorage{}, nil)

	result, err := svc.InstallSkillFromGitHub(context.Background(), 1, 2, 3, "https://github.com/org/repo", "main", "", "user")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.SourceURL != "https://github.com/org/repo@main" {
		t.Errorf("expected source URL with branch, got %q", result.SourceURL)
	}
}

func TestInstallSkillFromGitHub_NoPathNoBranch(t *testing.T) {
	repo := &svcMockRepo{
		createInstalledSkillFn: func(_ context.Context, skill *extension.InstalledSkill) error {
			skill.ID = 1
			return nil
		},
	}
	svc := newTestServiceWithPackager(repo, &svcMockStorage{}, nil)

	result, err := svc.InstallSkillFromGitHub(context.Background(), 1, 2, 3, "https://github.com/org/repo", "", "", "org")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.SourceURL != "https://github.com/org/repo" {
		t.Errorf("expected plain source URL, got %q", result.SourceURL)
	}
}

// ---------------------------------------------------------------------------
// Tests: UpdateMcpServer — orgID mismatch
// ---------------------------------------------------------------------------

func TestUpdateMcpServer_OrgIDMismatch(t *testing.T) {
	repo := &svcMockRepo{
		getInstalledMcpServerFn: func(_ context.Context, id int64) (*extension.InstalledMcpServer, error) {
			return &extension.InstalledMcpServer{
				ID:             id,
				OrganizationID: 1,
				RepositoryID:   2,
				Scope:          "org",
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	_, err := svc.UpdateMcpServer(context.Background(), 999, 2, 10, 100, "admin", ptrBool(true), nil)
	if err == nil {
		t.Fatal("expected error for orgID mismatch, got nil")
	}
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("expected ErrForbidden, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Tests: UninstallMcpServer — orgID mismatch
// ---------------------------------------------------------------------------

func TestUninstallMcpServer_OrgIDMismatch(t *testing.T) {
	repo := &svcMockRepo{
		getInstalledMcpServerFn: func(_ context.Context, id int64) (*extension.InstalledMcpServer, error) {
			return &extension.InstalledMcpServer{
				ID:             id,
				OrganizationID: 1,
				RepositoryID:   2,
				Scope:          "org",
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	err := svc.UninstallMcpServer(context.Background(), 999, 2, 10, 100, "admin")
	if err == nil {
		t.Fatal("expected error for orgID mismatch, got nil")
	}
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("expected ErrForbidden, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Tests: InstallMcpFromMarket — CreateInstalledMcpServer DB error
// ---------------------------------------------------------------------------

func TestInstallMcpFromMarket_CreateServerError(t *testing.T) {
	repo := &svcMockRepo{
		getMcpMarketItemFn: func(_ context.Context, id int64) (*extension.McpMarketItem, error) {
			return &extension.McpMarketItem{
				ID:            id,
				Name:          "test-mcp",
				Slug:          "test-mcp",
				TransportType: "stdio",
				Command:       "node",
			}, nil
		},
		createInstalledMcpServerFn: func(_ context.Context, _ *extension.InstalledMcpServer) error {
			return errors.New("db insert failed")
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	_, err := svc.InstallMcpFromMarket(context.Background(), 1, 2, 3, 50, nil, "org")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestInstallMcpFromMarket_GetMarketItemError(t *testing.T) {
	repo := &svcMockRepo{
		getMcpMarketItemFn: func(_ context.Context, id int64) (*extension.McpMarketItem, error) {
			return nil, errors.New("not found")
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	_, err := svc.InstallMcpFromMarket(context.Background(), 1, 2, 3, 50, nil, "org")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Tests: InstallCustomMcpServer — CreateInstalledMcpServer DB error
// ---------------------------------------------------------------------------

func TestInstallCustomMcpServer_CreateServerError(t *testing.T) {
	repo := &svcMockRepo{
		createInstalledMcpServerFn: func(_ context.Context, _ *extension.InstalledMcpServer) error {
			return errors.New("db insert failed")
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	server := &extension.InstalledMcpServer{
		Name:          "custom-mcp",
		Slug:          "custom-mcp",
		Scope:         "user",
		TransportType: "stdio",
		Command:       "node",
	}
	_, err := svc.InstallCustomMcpServer(context.Background(), 1, 2, 3, server, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// Tests: Agent type filtering for GetEffectiveMcpServers
// ---------------------------------------------------------------------------

func TestGetEffectiveMcpServers_AgentTypeFilter_MatchingAgent(t *testing.T) {
	// MCP server with MarketItem filter ["claude-code"] should be included when agentSlug="claude-code"
	repo := &svcMockRepo{
		getEffectiveMcpServersFn: func(_ context.Context, orgID, userID, repoID int64) ([]*extension.InstalledMcpServer, error) {
			return []*extension.InstalledMcpServer{
				{
					ID:           1,
					Slug:         "filtered-server",
					MarketItemID: int64Ptr(100),
					MarketItem: &extension.McpMarketItem{
						ID:              100,
						Slug:            "filtered-server",
						AgentTypeFilter: json.RawMessage(`["claude-code"]`),
					},
				},
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	servers, err := svc.GetEffectiveMcpServers(context.Background(), 1, 2, 3, "claude-code")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(servers))
	}
	if servers[0].Slug != "filtered-server" {
		t.Errorf("expected slug 'filtered-server', got %q", servers[0].Slug)
	}
}

func TestGetEffectiveMcpServers_AgentTypeFilter_NonMatchingAgent(t *testing.T) {
	// MCP server with MarketItem filter ["claude-code"] should be excluded when agentSlug="aider"
	repo := &svcMockRepo{
		getEffectiveMcpServersFn: func(_ context.Context, orgID, userID, repoID int64) ([]*extension.InstalledMcpServer, error) {
			return []*extension.InstalledMcpServer{
				{
					ID:           1,
					Slug:         "claude-only-server",
					MarketItemID: int64Ptr(100),
					MarketItem: &extension.McpMarketItem{
						ID:              100,
						Slug:            "claude-only-server",
						AgentTypeFilter: json.RawMessage(`["claude-code"]`),
					},
				},
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	servers, err := svc.GetEffectiveMcpServers(context.Background(), 1, 2, 3, "aider")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(servers) != 0 {
		t.Fatalf("expected 0 servers (filtered out), got %d", len(servers))
	}
}

func TestGetEffectiveMcpServers_AgentTypeFilter_CustomServerAlwaysIncluded(t *testing.T) {
	// MCP server without MarketItem (custom install) should always be included
	repo := &svcMockRepo{
		getEffectiveMcpServersFn: func(_ context.Context, orgID, userID, repoID int64) ([]*extension.InstalledMcpServer, error) {
			return []*extension.InstalledMcpServer{
				{
					ID:         1,
					Slug:       "custom-server",
					MarketItem: nil, // custom install, no market item
				},
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	servers, err := svc.GetEffectiveMcpServers(context.Background(), 1, 2, 3, "aider")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(servers) != 1 {
		t.Fatalf("expected 1 server (custom always included), got %d", len(servers))
	}
}

func TestGetEffectiveMcpServers_AgentTypeFilter_NullFilterAllowsAll(t *testing.T) {
	// MCP server with MarketItem that has null/empty agent_type_filter should be included for any agent
	repo := &svcMockRepo{
		getEffectiveMcpServersFn: func(_ context.Context, orgID, userID, repoID int64) ([]*extension.InstalledMcpServer, error) {
			return []*extension.InstalledMcpServer{
				{
					ID:           1,
					Slug:         "universal-server",
					MarketItemID: int64Ptr(100),
					MarketItem: &extension.McpMarketItem{
						ID:              100,
						Slug:            "universal-server",
						AgentTypeFilter: nil, // null = all agents
					},
				},
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	servers, err := svc.GetEffectiveMcpServers(context.Background(), 1, 2, 3, "any-agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(servers) != 1 {
		t.Fatalf("expected 1 server (null filter = all agents), got %d", len(servers))
	}
}

func TestGetEffectiveMcpServers_AgentTypeFilter_EmptySlugDisablesFilter(t *testing.T) {
	// When agentSlug is empty, no filtering should happen
	repo := &svcMockRepo{
		getEffectiveMcpServersFn: func(_ context.Context, orgID, userID, repoID int64) ([]*extension.InstalledMcpServer, error) {
			return []*extension.InstalledMcpServer{
				{
					ID:           1,
					Slug:         "claude-only",
					MarketItemID: int64Ptr(100),
					MarketItem: &extension.McpMarketItem{
						ID:              100,
						Slug:            "claude-only",
						AgentTypeFilter: json.RawMessage(`["claude-code"]`),
					},
				},
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	servers, err := svc.GetEffectiveMcpServers(context.Background(), 1, 2, 3, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(servers) != 1 {
		t.Fatalf("expected 1 server (empty agentSlug = no filtering), got %d", len(servers))
	}
}

func TestGetEffectiveMcpServers_AgentTypeFilter_MultipleAgentTypes(t *testing.T) {
	// MCP server with filter ["claude-code", "aider"] should be included for both
	repo := &svcMockRepo{
		getEffectiveMcpServersFn: func(_ context.Context, orgID, userID, repoID int64) ([]*extension.InstalledMcpServer, error) {
			return []*extension.InstalledMcpServer{
				{
					ID:           1,
					Slug:         "multi-agent-server",
					MarketItemID: int64Ptr(100),
					MarketItem: &extension.McpMarketItem{
						ID:              100,
						Slug:            "multi-agent-server",
						AgentTypeFilter: json.RawMessage(`["claude-code", "aider"]`),
					},
				},
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	// Should be included for claude-code
	servers, err := svc.GetEffectiveMcpServers(context.Background(), 1, 2, 3, "claude-code")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(servers) != 1 {
		t.Fatalf("expected 1 server for claude-code, got %d", len(servers))
	}

	// Should be included for aider
	servers, err = svc.GetEffectiveMcpServers(context.Background(), 1, 2, 3, "aider")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(servers) != 1 {
		t.Fatalf("expected 1 server for aider, got %d", len(servers))
	}

	// Should NOT be included for codex
	servers, err = svc.GetEffectiveMcpServers(context.Background(), 1, 2, 3, "codex")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(servers) != 0 {
		t.Fatalf("expected 0 servers for codex, got %d", len(servers))
	}
}

func TestGetEffectiveMcpServers_AgentTypeFilter_MixedServers(t *testing.T) {
	// Mix of filtered, unfiltered, and custom servers
	repo := &svcMockRepo{
		getEffectiveMcpServersFn: func(_ context.Context, orgID, userID, repoID int64) ([]*extension.InstalledMcpServer, error) {
			return []*extension.InstalledMcpServer{
				{
					ID:           1,
					Slug:         "claude-only",
					MarketItemID: int64Ptr(100),
					MarketItem: &extension.McpMarketItem{
						ID:              100,
						AgentTypeFilter: json.RawMessage(`["claude-code"]`),
					},
				},
				{
					ID:           2,
					Slug:         "universal",
					MarketItemID: int64Ptr(101),
					MarketItem: &extension.McpMarketItem{
						ID:              101,
						AgentTypeFilter: nil,
					},
				},
				{
					ID:         3,
					Slug:       "custom",
					MarketItem: nil,
				},
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	// For aider: should get universal + custom (not claude-only)
	servers, err := svc.GetEffectiveMcpServers(context.Background(), 1, 2, 3, "aider")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(servers) != 2 {
		t.Fatalf("expected 2 servers for aider, got %d", len(servers))
	}
	slugs := make(map[string]bool)
	for _, s := range servers {
		slugs[s.Slug] = true
	}
	if slugs["claude-only"] {
		t.Error("claude-only server should have been filtered out for aider")
	}
	if !slugs["universal"] {
		t.Error("universal server should be included")
	}
	if !slugs["custom"] {
		t.Error("custom server should be included")
	}
}

// ---------------------------------------------------------------------------
// Tests: Agent type filtering for GetEffectiveSkills
// ---------------------------------------------------------------------------

func TestGetEffectiveSkills_AgentTypeFilter_MatchingAgent(t *testing.T) {
	// Skill with MarketItem filter ["claude-code"] should be included when agentSlug="claude-code"
	repo := &svcMockRepo{
		getEffectiveSkillsFn: func(_ context.Context, orgID, userID, repoID int64) ([]*extension.InstalledSkill, error) {
			return []*extension.InstalledSkill{
				{
					ID:            1,
					Slug:          "filtered-skill",
					InstallSource: "market",
					ContentSha:    "abc123",
					StorageKey:    "skills/filtered-skill/v1.tar.gz",
					PackageSize:   1024,
					MarketItemID:  int64Ptr(100),
					MarketItem: &extension.SkillMarketItem{
						ID:              100,
						Slug:            "filtered-skill",
						AgentTypeFilter: json.RawMessage(`["claude-code"]`),
						ContentSha:      "abc123",
						StorageKey:      "skills/filtered-skill/v1.tar.gz",
						PackageSize:     1024,
					},
				},
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	resolved, err := svc.GetEffectiveSkills(context.Background(), 1, 2, 3, "claude-code")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(resolved))
	}
	if resolved[0].Slug != "filtered-skill" {
		t.Errorf("expected slug 'filtered-skill', got %q", resolved[0].Slug)
	}
}

func TestGetEffectiveSkills_AgentTypeFilter_NonMatchingAgent(t *testing.T) {
	// Skill with MarketItem filter ["claude-code"] should be excluded when agentSlug="aider"
	repo := &svcMockRepo{
		getEffectiveSkillsFn: func(_ context.Context, orgID, userID, repoID int64) ([]*extension.InstalledSkill, error) {
			return []*extension.InstalledSkill{
				{
					ID:            1,
					Slug:          "claude-only-skill",
					InstallSource: "market",
					ContentSha:    "abc123",
					StorageKey:    "skills/claude-only-skill/v1.tar.gz",
					PackageSize:   1024,
					MarketItemID:  int64Ptr(100),
					MarketItem: &extension.SkillMarketItem{
						ID:              100,
						Slug:            "claude-only-skill",
						AgentTypeFilter: json.RawMessage(`["claude-code"]`),
						ContentSha:      "abc123",
						StorageKey:      "skills/claude-only-skill/v1.tar.gz",
						PackageSize:     1024,
					},
				},
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	resolved, err := svc.GetEffectiveSkills(context.Background(), 1, 2, 3, "aider")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved) != 0 {
		t.Fatalf("expected 0 skills (filtered out), got %d", len(resolved))
	}
}

func TestGetEffectiveSkills_AgentTypeFilter_GitHubInstallAlwaysIncluded(t *testing.T) {
	// Skill without MarketItem (github install) should always be included
	repo := &svcMockRepo{
		getEffectiveSkillsFn: func(_ context.Context, orgID, userID, repoID int64) ([]*extension.InstalledSkill, error) {
			return []*extension.InstalledSkill{
				{
					ID:            1,
					Slug:          "github-skill",
					InstallSource: "github",
					ContentSha:    "def456",
					StorageKey:    "skills/github-skill/v1.tar.gz",
					PackageSize:   2048,
					MarketItem:    nil, // github install, no market item
				},
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	resolved, err := svc.GetEffectiveSkills(context.Background(), 1, 2, 3, "aider")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved) != 1 {
		t.Fatalf("expected 1 skill (github always included), got %d", len(resolved))
	}
}

func TestGetEffectiveSkills_AgentTypeFilter_NullFilterAllowsAll(t *testing.T) {
	// Skill with MarketItem that has null agent_type_filter should be included for any agent
	repo := &svcMockRepo{
		getEffectiveSkillsFn: func(_ context.Context, orgID, userID, repoID int64) ([]*extension.InstalledSkill, error) {
			return []*extension.InstalledSkill{
				{
					ID:            1,
					Slug:          "universal-skill",
					InstallSource: "market",
					ContentSha:    "ghi789",
					StorageKey:    "skills/universal-skill/v1.tar.gz",
					PackageSize:   512,
					MarketItemID:  int64Ptr(100),
					MarketItem: &extension.SkillMarketItem{
						ID:              100,
						Slug:            "universal-skill",
						AgentTypeFilter: nil,
						ContentSha:      "ghi789",
						StorageKey:      "skills/universal-skill/v1.tar.gz",
						PackageSize:     512,
					},
				},
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	resolved, err := svc.GetEffectiveSkills(context.Background(), 1, 2, 3, "any-agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved) != 1 {
		t.Fatalf("expected 1 skill (null filter = all agents), got %d", len(resolved))
	}
}

func TestGetEffectiveSkills_AgentTypeFilter_EmptySlugDisablesFilter(t *testing.T) {
	// When agentSlug is empty, no filtering should happen
	repo := &svcMockRepo{
		getEffectiveSkillsFn: func(_ context.Context, orgID, userID, repoID int64) ([]*extension.InstalledSkill, error) {
			return []*extension.InstalledSkill{
				{
					ID:            1,
					Slug:          "claude-only-skill",
					InstallSource: "market",
					ContentSha:    "abc123",
					StorageKey:    "skills/claude-only-skill/v1.tar.gz",
					PackageSize:   1024,
					MarketItemID:  int64Ptr(100),
					MarketItem: &extension.SkillMarketItem{
						ID:              100,
						Slug:            "claude-only-skill",
						AgentTypeFilter: json.RawMessage(`["claude-code"]`),
						ContentSha:      "abc123",
						StorageKey:      "skills/claude-only-skill/v1.tar.gz",
						PackageSize:     1024,
					},
				},
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	resolved, err := svc.GetEffectiveSkills(context.Background(), 1, 2, 3, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved) != 1 {
		t.Fatalf("expected 1 skill (empty agentSlug = no filtering), got %d", len(resolved))
	}
}

func TestGetEffectiveSkills_AgentTypeFilter_MixedSkills(t *testing.T) {
	// Mix of filtered, unfiltered, and non-market skills
	repo := &svcMockRepo{
		getEffectiveSkillsFn: func(_ context.Context, orgID, userID, repoID int64) ([]*extension.InstalledSkill, error) {
			return []*extension.InstalledSkill{
				{
					ID:            1,
					Slug:          "claude-only",
					InstallSource: "market",
					ContentSha:    "sha1",
					StorageKey:    "skills/claude-only/v1.tar.gz",
					PackageSize:   100,
					MarketItemID:  int64Ptr(100),
					MarketItem: &extension.SkillMarketItem{
						ID:              100,
						AgentTypeFilter: json.RawMessage(`["claude-code"]`),
						ContentSha:      "sha1",
						StorageKey:      "skills/claude-only/v1.tar.gz",
						PackageSize:     100,
					},
				},
				{
					ID:            2,
					Slug:          "universal",
					InstallSource: "market",
					ContentSha:    "sha2",
					StorageKey:    "skills/universal/v1.tar.gz",
					PackageSize:   200,
					MarketItemID:  int64Ptr(101),
					MarketItem: &extension.SkillMarketItem{
						ID:              101,
						AgentTypeFilter: nil,
						ContentSha:      "sha2",
						StorageKey:      "skills/universal/v1.tar.gz",
						PackageSize:     200,
					},
				},
				{
					ID:            3,
					Slug:          "github-skill",
					InstallSource: "github",
					ContentSha:    "sha3",
					StorageKey:    "skills/github-skill/v1.tar.gz",
					PackageSize:   300,
					MarketItem:    nil,
				},
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	// For aider: should get universal + github-skill (not claude-only)
	resolved, err := svc.GetEffectiveSkills(context.Background(), 1, 2, 3, "aider")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved) != 2 {
		t.Fatalf("expected 2 skills for aider, got %d", len(resolved))
	}
	slugs := make(map[string]bool)
	for _, r := range resolved {
		slugs[r.Slug] = true
	}
	if slugs["claude-only"] {
		t.Error("claude-only skill should have been filtered out for aider")
	}
	if !slugs["universal"] {
		t.Error("universal skill should be included")
	}
	if !slugs["github-skill"] {
		t.Error("github-skill should be included")
	}
}

// ---------------------------------------------------------------------------
// Tests: TogglePlatformRegistry
// ---------------------------------------------------------------------------

func TestTogglePlatformRegistry_Success(t *testing.T) {
	var capturedOrgID, capturedRegistryID int64
	var capturedDisabled bool
	repo := &svcMockRepo{
		getSkillRegistryFn: func(_ context.Context, id int64) (*extension.SkillRegistry, error) {
			return &extension.SkillRegistry{
				ID:             id,
				OrganizationID: nil, // platform-level
				IsActive:       true,
			}, nil
		},
		setSkillRegistryOverrideFn: func(_ context.Context, orgID int64, registryID int64, isDisabled bool) error {
			capturedOrgID = orgID
			capturedRegistryID = registryID
			capturedDisabled = isDisabled
			return nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	err := svc.TogglePlatformRegistry(context.Background(), 42, 10, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedOrgID != 42 {
		t.Errorf("expected orgID 42, got %d", capturedOrgID)
	}
	if capturedRegistryID != 10 {
		t.Errorf("expected registryID 10, got %d", capturedRegistryID)
	}
	if !capturedDisabled {
		t.Error("expected disabled=true")
	}
}

func TestTogglePlatformRegistry_SourceNotFound(t *testing.T) {
	repo := &svcMockRepo{
		getSkillRegistryFn: func(_ context.Context, id int64) (*extension.SkillRegistry, error) {
			return nil, fmt.Errorf("not found")
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	err := svc.TogglePlatformRegistry(context.Background(), 1, 999, true)
	if err == nil {
		t.Fatal("expected error for missing source, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestTogglePlatformRegistry_NotPlatformLevel(t *testing.T) {
	orgID := int64(1)
	repo := &svcMockRepo{
		getSkillRegistryFn: func(_ context.Context, id int64) (*extension.SkillRegistry, error) {
			return &extension.SkillRegistry{
				ID:             id,
				OrganizationID: &orgID, // org-level, not platform
				IsActive:       true,
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	err := svc.TogglePlatformRegistry(context.Background(), 1, 10, true)
	if err == nil {
		t.Fatal("expected error for non-platform-level source, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Tests: ListSkillRegistryOverrides
// ---------------------------------------------------------------------------

func TestListSkillRegistryOverrides_Success(t *testing.T) {
	repo := &svcMockRepo{
		listSkillRegistryOverridesFn: func(_ context.Context, orgID int64) ([]*extension.SkillRegistryOverride, error) {
			return []*extension.SkillRegistryOverride{
				{ID: 1, OrganizationID: orgID, RegistryID: 10, IsDisabled: true},
				{ID: 2, OrganizationID: orgID, RegistryID: 20, IsDisabled: false},
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	result, err := svc.ListSkillRegistryOverrides(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 overrides, got %d", len(result))
	}
	if result[0].RegistryID != 10 {
		t.Errorf("expected first override registryID 10, got %d", result[0].RegistryID)
	}
}

func TestListSkillRegistryOverrides_Error(t *testing.T) {
	repo := &svcMockRepo{
		listSkillRegistryOverridesFn: func(_ context.Context, orgID int64) ([]*extension.SkillRegistryOverride, error) {
			return nil, errors.New("db error")
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	_, err := svc.ListSkillRegistryOverrides(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// Tests: InstallSkillFromUpload
// ---------------------------------------------------------------------------

func TestInstallSkillFromUpload_Success(t *testing.T) {
	repo := &svcMockRepo{
		createInstalledSkillFn: func(_ context.Context, skill *extension.InstalledSkill) error {
			skill.ID = 1
			return nil
		},
	}
	stor := &svcMockStorage{}
	svc := newTestServiceWithPackager(repo, stor, nil)

	// The slug is derived from the 'name' field in SKILL.md frontmatter (see parseSkillDir)
	reader := createMinimalTarGz(t, "SKILL.md", "---\nname: upload-skill\n---\n# Upload Skill\nA test skill.")

	result, err := svc.InstallSkillFromUpload(context.Background(), 1, 2, 3, reader, "skill.tar.gz", "org")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.InstallSource != "upload" {
		t.Errorf("expected install_source 'upload', got %q", result.InstallSource)
	}
	if result.Slug != "upload-skill" {
		t.Errorf("expected slug 'upload-skill', got %q", result.Slug)
	}
}

func TestInstallSkillFromUpload_InvalidScope(t *testing.T) {
	svc := newTestServiceWithPackager(&svcMockRepo{}, &svcMockStorage{}, nil)

	_, err := svc.InstallSkillFromUpload(context.Background(), 1, 2, 3, strings.NewReader("data"), "file.tar.gz", "bad")
	if err == nil {
		t.Fatal("expected error for invalid scope, got nil")
	}
	if !errors.Is(err, ErrInvalidScope) {
		t.Errorf("expected ErrInvalidScope, got: %v", err)
	}
}

func TestInstallSkillFromUpload_NoPackager(t *testing.T) {
	// Service without packager set
	svc := newTestService(&svcMockRepo{}, &svcMockStorage{}, nil)

	_, err := svc.InstallSkillFromUpload(context.Background(), 1, 2, 3, strings.NewReader("data"), "file.tar.gz", "org")
	if err == nil {
		t.Fatal("expected error for nil packager, got nil")
	}
	if !strings.Contains(err.Error(), "skill packager not configured") {
		t.Errorf("expected 'skill packager not configured' error, got: %v", err)
	}
}

// createMinimalTarGz creates a tar.gz archive in memory containing a single file.
func createMinimalTarGz(t *testing.T, filename, content string) io.Reader {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	hdr := &tar.Header{
		Name: filename,
		Mode: 0644,
		Size: int64(len(content)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("failed to write tar header: %v", err)
	}
	if _, err := tw.Write([]byte(content)); err != nil {
		t.Fatalf("failed to write tar content: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("failed to close tar writer: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("failed to close gzip writer: %v", err)
	}
	return &buf
}

// ---------------------------------------------------------------------------
// Tests: CreateSkillRegistry — additional branches
// ---------------------------------------------------------------------------

func TestCreateSkillRegistry_WithCompatibleAgents(t *testing.T) {
	var captured *extension.SkillRegistry
	repo := &svcMockRepo{
		createSkillRegistryFn: func(_ context.Context, source *extension.SkillRegistry) error {
			captured = source
			return nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	result, err := svc.CreateSkillRegistry(context.Background(), 1, CreateSkillRegistryInput{
		RepositoryURL:    "https://github.com/org/repo",
		CompatibleAgents: []string{"claude-code", "aider"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if captured == nil {
		t.Fatal("repo.CreateSkillRegistry was not called")
	}
	// Check that compatible_agents is set as JSON
	agents := result.GetCompatibleAgents()
	if len(agents) != 2 {
		t.Fatalf("expected 2 compatible agents, got %d", len(agents))
	}
	if agents[0] != "claude-code" {
		t.Errorf("expected first agent 'claude-code', got %q", agents[0])
	}
	if agents[1] != "aider" {
		t.Errorf("expected second agent 'aider', got %q", agents[1])
	}
}

func TestCreateSkillRegistry_WithAuthCredential(t *testing.T) {
	enc := crypto.NewEncryptor("test-secret-key-1234567890123456")
	var captured *extension.SkillRegistry
	repo := &svcMockRepo{
		createSkillRegistryFn: func(_ context.Context, source *extension.SkillRegistry) error {
			captured = source
			return nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, enc)

	result, err := svc.CreateSkillRegistry(context.Background(), 1, CreateSkillRegistryInput{
		RepositoryURL:  "https://github.com/org/private-repo",
		AuthType:       extension.AuthTypeGitHubPAT,
		AuthCredential: "ghp_testtoken123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if captured == nil {
		t.Fatal("repo.CreateSkillRegistry was not called")
	}
	if result.AuthType != extension.AuthTypeGitHubPAT {
		t.Errorf("expected auth_type '%s', got %q", extension.AuthTypeGitHubPAT, result.AuthType)
	}
	// Credential should be encrypted (not plain text)
	if result.AuthCredential == "ghp_testtoken123" {
		t.Error("auth credential should be encrypted, not plain text")
	}
	if result.AuthCredential == "" {
		t.Error("auth credential should not be empty")
	}
	// Verify it can be decrypted back
	decrypted, err := enc.Decrypt(result.AuthCredential)
	if err != nil {
		t.Fatalf("failed to decrypt auth credential: %v", err)
	}
	if decrypted != "ghp_testtoken123" {
		t.Errorf("expected decrypted value 'ghp_testtoken123', got %q", decrypted)
	}
}

func TestCreateSkillRegistry_InvalidAuthType(t *testing.T) {
	svc := newTestService(&svcMockRepo{}, &svcMockStorage{}, nil)

	_, err := svc.CreateSkillRegistry(context.Background(), 1, CreateSkillRegistryInput{
		RepositoryURL: "https://github.com/org/repo",
		AuthType:      "invalid_type",
	})
	if err == nil {
		t.Fatal("expected error for invalid auth_type, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Tests: InstallSkillFromGitHub — no packager
// ---------------------------------------------------------------------------

func TestInstallSkillFromGitHub_NoPackager(t *testing.T) {
	// Service without packager set
	svc := newTestService(&svcMockRepo{}, &svcMockStorage{}, nil)

	_, err := svc.InstallSkillFromGitHub(context.Background(), 1, 2, 3, "https://github.com/org/repo", "", "", "org")
	if err == nil {
		t.Fatal("expected error for nil packager, got nil")
	}
	if !strings.Contains(err.Error(), "skill packager not configured") {
		t.Errorf("expected 'skill packager not configured' error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Tests: Encryption helpers (encryptCredential / decryptCredential)
// ---------------------------------------------------------------------------

func TestDecryptCredential_NoCrypto(t *testing.T) {
	svc := newTestService(&svcMockRepo{}, &svcMockStorage{}, nil)

	result, err := svc.DecryptCredential("some-value")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "some-value" {
		t.Errorf("expected 'some-value' returned as-is, got %q", result)
	}
}

func TestDecryptCredential_EmptyString(t *testing.T) {
	enc := crypto.NewEncryptor("test-secret-key-1234567890123456")
	svc := newTestService(&svcMockRepo{}, &svcMockStorage{}, enc)

	result, err := svc.DecryptCredential("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestDecryptCredential_Success(t *testing.T) {
	enc := crypto.NewEncryptor("test-secret-key-1234567890123456")
	svc := newTestService(&svcMockRepo{}, &svcMockStorage{}, enc)

	// Encrypt a value first
	encrypted, err := enc.Encrypt("my-secret-token")
	if err != nil {
		t.Fatalf("failed to encrypt: %v", err)
	}

	// Decrypt via service
	result, err := svc.DecryptCredential(encrypted)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "my-secret-token" {
		t.Errorf("expected 'my-secret-token', got %q", result)
	}
}

func TestEncryptCredential_NoCrypto(t *testing.T) {
	svc := newTestService(&svcMockRepo{}, &svcMockStorage{}, nil)

	result, err := svc.encryptCredential("plain-value")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "plain-value" {
		t.Errorf("expected 'plain-value' returned as-is (no crypto), got %q", result)
	}
}

func TestEncryptCredential_Success(t *testing.T) {
	enc := crypto.NewEncryptor("test-secret-key-1234567890123456")
	svc := newTestService(&svcMockRepo{}, &svcMockStorage{}, enc)

	encrypted, err := svc.encryptCredential("my-secret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Encrypted value should be different from plain text
	if encrypted == "my-secret" {
		t.Error("expected encrypted value to differ from plain text")
	}
	// Verify it can be decrypted back
	decrypted, err := enc.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("failed to decrypt: %v", err)
	}
	if decrypted != "my-secret" {
		t.Errorf("expected 'my-secret', got %q", decrypted)
	}
}

// ---------------------------------------------------------------------------
// Tests: decryptCredential — additional coverage for decrypt-failure path
// ---------------------------------------------------------------------------

func TestDecryptCredential_CryptoPresent_DecryptFails_ReturnsOriginal(t *testing.T) {
	// When crypto is present but the value is not valid encrypted data,
	// decryptCredential should return the original value as-is (not an error).
	enc := crypto.NewEncryptor("test-secret-key-1234567890123456")
	svc := newTestService(&svcMockRepo{}, &svcMockStorage{}, enc)

	// "not-encrypted-data" is not valid AES-GCM ciphertext
	result, err := svc.decryptCredential("not-encrypted-data")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should return the original value when decryption fails
	if result != "not-encrypted-data" {
		t.Errorf("expected original value 'not-encrypted-data', got %q", result)
	}
}

func TestDecryptCredential_CryptoPresent_DecryptFailsWithBase64Data_ReturnsOriginal(t *testing.T) {
	// Use a different key to encrypt, so decrypting with the service's key fails
	otherEnc := crypto.NewEncryptor("different-key-1234567890123456")
	encrypted, err := otherEnc.Encrypt("secret-value")
	if err != nil {
		t.Fatalf("failed to encrypt with other key: %v", err)
	}

	// Service uses a different key
	svcEnc := crypto.NewEncryptor("test-secret-key-1234567890123456")
	svc := newTestService(&svcMockRepo{}, &svcMockStorage{}, svcEnc)

	// Decryption should fail (wrong key), return original value
	result, err := svc.decryptCredential(encrypted)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != encrypted {
		t.Errorf("expected original encrypted value returned as-is, got different value")
	}
}

// ---------------------------------------------------------------------------
// Tests: encryptEnvVars — additional coverage
// ---------------------------------------------------------------------------

func TestEncryptEnvVars_EmptyMap_ReturnsEmptyJSONObject(t *testing.T) {
	enc := crypto.NewEncryptor("test-secret-key-1234567890123456")
	svc := newTestService(&svcMockRepo{}, &svcMockStorage{}, enc)

	data, err := svc.encryptEnvVars(map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]string
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty map, got %v", result)
	}
}

func TestEncryptEnvVars_NoCrypto_EmptyMap(t *testing.T) {
	svc := newTestService(&svcMockRepo{}, &svcMockStorage{}, nil)

	data, err := svc.encryptEnvVars(map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]string
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty map, got %v", result)
	}
}

func TestEncryptEnvVars_MultipleKeys_AllEncrypted(t *testing.T) {
	enc := crypto.NewEncryptor("test-secret-key-1234567890123456")
	svc := newTestService(&svcMockRepo{}, &svcMockStorage{}, enc)

	vars := map[string]string{
		"DB_PASSWORD":  "secret1",
		"API_KEY":      "secret2",
		"WEBHOOK_TOKEN": "secret3",
	}
	data, err := svc.encryptEnvVars(vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]string
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// All keys should be present
	if len(result) != 3 {
		t.Errorf("expected 3 keys, got %d", len(result))
	}

	// Each value should be encrypted (different from plain text) and decryptable
	for k, plain := range vars {
		encrypted, ok := result[k]
		if !ok {
			t.Errorf("missing key %s", k)
			continue
		}
		if encrypted == plain {
			t.Errorf("key %s should be encrypted, not plain text", k)
		}
		decrypted, err := enc.Decrypt(encrypted)
		if err != nil {
			t.Fatalf("failed to decrypt %s: %v", k, err)
		}
		if decrypted != plain {
			t.Errorf("expected %q for key %s, got %q", plain, k, decrypted)
		}
	}
}
