package user

import (
	"context"
	"log/slog"

	domainUser "github.com/anthropics/agentsmesh/backend/internal/domain/user"
	"github.com/anthropics/agentsmesh/backend/pkg/crypto"
)

// SetDefaultGitCredential sets a Git credential as the user's default
func (s *Service) SetDefaultGitCredential(ctx context.Context, userID, credentialID int64) error {
	// Verify ownership
	_, err := s.GetGitCredential(ctx, userID, credentialID)
	if err != nil {
		return err
	}

	if err := s.repo.SetDefaultGitCredential(ctx, userID, credentialID); err != nil {
		slog.Error("failed to set default git credential",
			"user_id", userID, "credential_id", credentialID, "error", err)
		return err
	}
	slog.Info("default git credential set", "user_id", userID, "credential_id", credentialID)
	return nil
}

// ClearDefaultGitCredential clears the user's default Git credential (falls back to runner_local)
func (s *Service) ClearDefaultGitCredential(ctx context.Context, userID int64) error {
	return s.repo.ClearAllDefaultGitCredentials(ctx, userID)
}

// GetDefaultGitCredential returns the user's default Git credential
// Returns nil if no default is set (meaning runner_local should be used)
func (s *Service) GetDefaultGitCredential(ctx context.Context, userID int64) (*domainUser.GitCredential, error) {
	return s.repo.GetDefaultGitCredential(ctx, userID)
}

// GetDecryptedCredentialToken retrieves and decrypts the token for a Git credential
func (s *Service) GetDecryptedCredentialToken(ctx context.Context, userID, credentialID int64) (*DecryptedCredential, error) {
	credential, err := s.GetGitCredential(ctx, userID, credentialID)
	if err != nil {
		return nil, err
	}

	result := &DecryptedCredential{
		Type: credential.CredentialType,
	}

	switch credential.CredentialType {
	case domainUser.CredentialTypeRunnerLocal:
		// No credentials to decrypt
		return result, nil

	case domainUser.CredentialTypeOAuth:
		if credential.RepositoryProviderID != nil {
			token, err := s.GetDecryptedProviderToken(ctx, userID, *credential.RepositoryProviderID)
			if err != nil {
				slog.Error("failed to decrypt oauth provider token",
					"user_id", userID, "credential_id", credentialID, "error", err)
				return nil, err
			}
			result.Token = token
		}

	case domainUser.CredentialTypePAT:
		if credential.PATEncrypted != nil && *credential.PATEncrypted != "" {
			if s.encryptionKey != "" {
				decrypted, err := crypto.DecryptWithKey(*credential.PATEncrypted, s.encryptionKey)
				if err != nil {
					slog.Error("failed to decrypt PAT",
						"user_id", userID, "credential_id", credentialID, "error", err)
					return nil, err
				}
				result.Token = decrypted
			} else {
				result.Token = *credential.PATEncrypted
			}
		}

	case domainUser.CredentialTypeSSHKey:
		if credential.PrivateKeyEncrypted != nil && *credential.PrivateKeyEncrypted != "" {
			if s.encryptionKey != "" {
				decrypted, err := crypto.DecryptWithKey(*credential.PrivateKeyEncrypted, s.encryptionKey)
				if err != nil {
					slog.Error("failed to decrypt SSH private key",
						"user_id", userID, "credential_id", credentialID, "error", err)
					return nil, err
				}
				result.SSHPrivateKey = decrypted
			} else {
				result.SSHPrivateKey = *credential.PrivateKeyEncrypted
			}
		}
		if credential.PublicKey != nil {
			result.SSHPublicKey = *credential.PublicKey
		}
	}

	return result, nil
}

// CreateCredentialFromProvider creates a Git credential linked to a repository provider (oauth type)
func (s *Service) CreateCredentialFromProvider(ctx context.Context, userID, providerID int64) (*domainUser.GitCredential, error) {
	provider, err := s.GetRepositoryProvider(ctx, userID, providerID)
	if err != nil {
		return nil, err
	}

	cred, err := s.CreateGitCredential(ctx, userID, &CreateGitCredentialRequest{
		Name:                 provider.Name + " (OAuth)",
		CredentialType:       domainUser.CredentialTypeOAuth,
		RepositoryProviderID: &providerID,
	})
	if err != nil {
		slog.Error("failed to create credential from provider",
			"user_id", userID, "provider_id", providerID, "error", err)
		return nil, err
	}
	slog.Info("credential created from provider",
		"user_id", userID, "credential_id", cred.ID, "provider_id", providerID)
	return cred, nil
}
