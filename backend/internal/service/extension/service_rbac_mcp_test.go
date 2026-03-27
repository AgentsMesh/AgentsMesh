package extension

import (
	"context"
	"errors"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
)

// ---------------------------------------------------------------------------
// Tests: UpdateMcpServer RBAC
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

// ---------------------------------------------------------------------------
// Tests: UninstallMcpServer RBAC
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
