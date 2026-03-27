package extension

import (
	"context"
	"errors"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
)

// ---------------------------------------------------------------------------
// Tests: InstallMcpFromMarket (error paths)
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

// ---------------------------------------------------------------------------
// Tests: InstallCustomMcpServer (error paths)
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
// Tests: Agent filtering for GetEffectiveMcpServers
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Tests: UpdateMcpServer (error paths)
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
// Tests: UninstallMcpServer (error paths)
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

// ---------------------------------------------------------------------------
// Tests: InstallMcpFromMarket (env var edge cases)
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
