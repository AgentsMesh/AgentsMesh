package extension

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
	"github.com/anthropics/agentsmesh/backend/pkg/crypto"
)

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
// Tests: GetEffectiveMcpServers (nil env vars + decrypt warning)
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
