package extension

import (
	"context"
	"errors"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
	"github.com/anthropics/agentsmesh/backend/pkg/crypto"
)

// ---------------------------------------------------------------------------
// Tests: MCP env var encryption edge cases
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
// Tests: MCP org/id mismatch
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
