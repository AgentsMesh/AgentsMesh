package extension

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
	"github.com/anthropics/agentsmesh/backend/internal/infra/storage"
	"github.com/anthropics/agentsmesh/backend/pkg/crypto"
)

var (
	ErrNotFound         = errors.New("resource not found")
	ErrForbidden        = errors.New("access denied")
	ErrInvalidScope     = errors.New("invalid scope")
	ErrInvalidInput     = errors.New("invalid input")
	ErrAlreadyInstalled = errors.New("already installed")
)

// validateScope checks that scope is "org" or "user".
func validateScope(scope string) error {
	if scope != extension.ScopeOrg && scope != extension.ScopeUser {
		return fmt.Errorf("%w: %s, must be 'org' or 'user'", ErrInvalidScope, scope)
	}
	return nil
}

const presignedURLExpiry = 15 * time.Minute

// Service provides extension management capabilities.
type Service struct {
	repo     extension.Repository
	storage  storage.Storage
	crypto   *crypto.Encryptor
	packager *SkillPackager
	importer *SkillImporter
}

func NewService(repo extension.Repository, storage storage.Storage, cryptoEncryptor *crypto.Encryptor) *Service {
	return &Service{
		repo:    repo,
		storage: storage,
		crypto:  cryptoEncryptor,
	}
}

// SetSkillPackager sets the SkillPackager dependency.
// This uses a setter to avoid circular initialization issues.
func (s *Service) SetSkillPackager(p *SkillPackager) {
	s.packager = p
}

// SetSkillImporter sets the SkillImporter dependency.
// This uses a setter to avoid circular initialization issues.
func (s *Service) SetSkillImporter(imp *SkillImporter) {
	s.importer = imp
}

// --- Skill Registries ---

func (s *Service) ListSkillRegistries(ctx context.Context, orgID int64) ([]*extension.SkillRegistry, error) {
	return s.repo.ListSkillRegistries(ctx, &orgID)
}

// CreateSkillRegistryInput holds the input for creating a skill registry.
type CreateSkillRegistryInput struct {
	RepositoryURL    string   `json:"repository_url"`
	Branch           string   `json:"branch"`
	SourceType       string   `json:"source_type"`
	CompatibleAgents []string `json:"compatible_agents"`
	AuthType         string   `json:"auth_type"`
	AuthCredential   string   `json:"auth_credential"`
}

func (s *Service) CreateSkillRegistry(ctx context.Context, orgID int64, input CreateSkillRegistryInput) (*extension.SkillRegistry, error) {
	if input.Branch == "" {
		input.Branch = "main"
	}
	if input.SourceType == "" {
		input.SourceType = "auto"
	}
	if input.AuthType == "" {
		input.AuthType = extension.AuthTypeNone
	}

	// Validate auth_type
	switch input.AuthType {
	case extension.AuthTypeNone, extension.AuthTypeGitHubPAT, extension.AuthTypeGitLabPAT, extension.AuthTypeSSHKey:
		// valid
	default:
		return nil, fmt.Errorf("%w: invalid auth_type %q", ErrInvalidInput, input.AuthType)
	}

	registry := &extension.SkillRegistry{
		OrganizationID: &orgID,
		RepositoryURL:  input.RepositoryURL,
		Branch:         input.Branch,
		SourceType:     input.SourceType,
		AuthType:       input.AuthType,
		SyncStatus:     "pending",
		IsActive:       true,
	}

	// Set compatible_agents as JSON
	if len(input.CompatibleAgents) > 0 {
		agentsJSON, err := marshalJSON(input.CompatibleAgents)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal compatible_agents: %w", err)
		}
		registry.CompatibleAgents = agentsJSON
	}
	// else: use DB default '["claude-code"]'

	// Encrypt credential if provided
	if input.AuthCredential != "" && input.AuthType != extension.AuthTypeNone {
		encrypted, err := s.encryptCredential(input.AuthCredential)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt auth credential: %w", err)
		}
		registry.AuthCredential = encrypted
	}

	if err := s.repo.CreateSkillRegistry(ctx, registry); err != nil {
		return nil, fmt.Errorf("failed to create skill registry: %w", err)
	}

	slog.Info("skill registry created", "registry_id", registry.ID, "org_id", orgID, "repository_url", input.RepositoryURL)

	return registry, nil
}

func (s *Service) SyncSkillRegistry(ctx context.Context, orgID, sourceID int64) (*extension.SkillRegistry, error) {
	registry, err := s.repo.GetSkillRegistry(ctx, sourceID)
	if err != nil {
		return nil, fmt.Errorf("%w: skill registry %d", ErrNotFound, sourceID)
	}

	// Platform-level registries (OrganizationID == nil) cannot be synced by org users
	if registry.OrganizationID == nil {
		return nil, fmt.Errorf("%w: cannot sync platform-level registry", ErrForbidden)
	}

	// Validate org ownership
	if *registry.OrganizationID != orgID {
		return nil, fmt.Errorf("%w: skill registry does not belong to this organization", ErrForbidden)
	}

	if s.importer == nil {
		return nil, fmt.Errorf("skill importer not available")
	}

	// Trigger sync — runs synchronously so caller gets final status
	if err := s.importer.SyncSource(ctx, sourceID); err != nil {
		slog.Error("Skill registry sync failed", "registry_id", sourceID, "error", err)
		// Reload to get the error status set by importer
		registry, _ = s.repo.GetSkillRegistry(ctx, sourceID)
		if registry != nil {
			return registry, nil
		}
		return nil, fmt.Errorf("sync failed: %w", err)
	}

	// Reload to get updated status
	registry, err = s.repo.GetSkillRegistry(ctx, sourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to reload registry after sync: %w", err)
	}

	slog.Info("skill registry synced", "registry_id", sourceID, "org_id", orgID, "sync_status", registry.SyncStatus)

	return registry, nil
}

func (s *Service) DeleteSkillRegistry(ctx context.Context, orgID, sourceID int64) error {
	registry, err := s.repo.GetSkillRegistry(ctx, sourceID)
	if err != nil {
		return fmt.Errorf("%w: skill registry %d", ErrNotFound, sourceID)
	}

	// Platform-level registries (OrganizationID == nil) cannot be deleted by org users
	if registry.OrganizationID == nil {
		return fmt.Errorf("%w: cannot delete platform-level registry", ErrForbidden)
	}

	// Validate org ownership
	if *registry.OrganizationID != orgID {
		return fmt.Errorf("%w: skill registry does not belong to this organization", ErrForbidden)
	}

	return s.repo.DeleteSkillRegistry(ctx, sourceID)
}

// --- Skill Registry Overrides ---

// TogglePlatformRegistry enables or disables a platform-level skill registry for an organization.
func (s *Service) TogglePlatformRegistry(ctx context.Context, orgID int64, sourceID int64, disabled bool) error {
	registry, err := s.repo.GetSkillRegistry(ctx, sourceID)
	if err != nil {
		return fmt.Errorf("%w: registry not found", ErrNotFound)
	}
	if !registry.IsPlatformLevel() {
		return fmt.Errorf("%w: can only toggle platform-level registries", ErrInvalidInput)
	}
	return s.repo.SetSkillRegistryOverride(ctx, orgID, sourceID, disabled)
}

func (s *Service) ListSkillRegistryOverrides(ctx context.Context, orgID int64) ([]*extension.SkillRegistryOverride, error) {
	return s.repo.ListSkillRegistryOverrides(ctx, orgID)
}

// --- Encryption helpers ---

// DecryptCredential decrypts a single credential string.
// Exported so it can be passed to SkillImporter.SetCredentialDecryptor.
func (s *Service) DecryptCredential(encrypted string) (string, error) {
	return s.decryptCredential(encrypted)
}

// encryptCredential encrypts a single credential string.
func (s *Service) encryptCredential(credential string) (string, error) {
	if s.crypto == nil {
		// No encryption configured (development mode), store as-is
		return credential, nil
	}
	return s.crypto.Encrypt(credential)
}

// decryptCredential decrypts a single credential string.
func (s *Service) decryptCredential(encrypted string) (string, error) {
	if s.crypto == nil || encrypted == "" {
		return encrypted, nil
	}
	decrypted, err := s.crypto.Decrypt(encrypted)
	if err != nil {
		// Log warning — silently returning ciphertext could leak encrypted values
		slog.Warn("Failed to decrypt credential, value may be unencrypted or corrupted", "error", err)
		return encrypted, nil
	}
	return decrypted, nil
}

func (s *Service) encryptEnvVars(vars map[string]string) ([]byte, error) {
	if s.crypto == nil {
		// No encryption configured, store as-is (development mode)
		return marshalJSON(vars)
	}
	// Encrypt each sensitive value
	encrypted := make(map[string]string, len(vars))
	for k, v := range vars {
		enc, err := s.crypto.Encrypt(v)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt env var %s: %w", k, err)
		}
		encrypted[k] = enc
	}
	return marshalJSON(encrypted)
}

func (s *Service) decryptServerEnvVars(server *extension.InstalledMcpServer) error {
	if s.crypto == nil || len(server.EnvVars) == 0 {
		return nil
	}

	var encrypted map[string]string
	if err := unmarshalJSON(server.EnvVars, &encrypted); err != nil {
		return err
	}

	decrypted := make(map[string]string, len(encrypted))
	for k, v := range encrypted {
		dec, err := s.crypto.Decrypt(v)
		if err != nil {
			// Log warning — silently keeping ciphertext could leak encrypted values
			slog.Warn("Failed to decrypt env var, value may be unencrypted or corrupted",
				"key", k, "error", err)
			decrypted[k] = v
			continue
		}
		decrypted[k] = dec
	}

	data, err := marshalJSON(decrypted)
	if err != nil {
		return fmt.Errorf("failed to marshal decrypted env vars: %w", err)
	}
	server.EnvVars = data
	return nil
}
