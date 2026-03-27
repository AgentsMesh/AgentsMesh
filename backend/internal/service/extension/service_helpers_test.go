package extension

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
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

func (m *svcMockRepo) BatchUpsertMcpMarketItems(_ context.Context, _ []*extension.McpMarketItem) error {
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

func (m *svcMockStorage) PresignPutURL(_ context.Context, _ string, _ string, _ time.Duration) (string, error) {
	return "", nil
}

func (m *svcMockStorage) InternalPresignPutURL(_ context.Context, _ string, _ string, _ time.Duration) (string, error) {
	return "", nil
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
