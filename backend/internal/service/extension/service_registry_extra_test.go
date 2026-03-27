package extension

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
	"github.com/anthropics/agentsmesh/backend/pkg/crypto"
)

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

