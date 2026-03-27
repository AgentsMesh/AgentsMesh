package extension

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
	"github.com/anthropics/agentsmesh/backend/pkg/crypto"
)

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
// Tests: SyncSkillRegistry (error paths)
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

// ---------------------------------------------------------------------------
// Tests: DeleteSkillRegistry (error paths)
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

