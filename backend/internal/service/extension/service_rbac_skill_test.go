package extension

import (
	"context"
	"errors"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
)

// ---------------------------------------------------------------------------
// Tests: UpdateSkill RBAC
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


// ---------------------------------------------------------------------------
// Tests: UninstallSkill RBAC
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
