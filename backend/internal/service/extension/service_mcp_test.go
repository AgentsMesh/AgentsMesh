package extension

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
	"github.com/anthropics/agentsmesh/backend/pkg/crypto"
)

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
