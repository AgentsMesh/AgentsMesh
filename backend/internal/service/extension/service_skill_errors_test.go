package extension

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
)

// ---------------------------------------------------------------------------
// Tests: InstallSkillFromMarket (error paths)
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

// ---------------------------------------------------------------------------
// Tests: UpdateSkill (error paths / nil fields)
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

// ---------------------------------------------------------------------------
// Tests: UninstallSkill (error paths)
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

// ---------------------------------------------------------------------------
// Tests: InstallSkillFromGitHub (error paths)
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
