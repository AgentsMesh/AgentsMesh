package user

import (
	"context"
	"errors"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/user"
	"github.com/AgentsMesh/AgentsMesh/backend/pkg/crypto"
	"gorm.io/gorm"
)

// SetDefaultGitCredential sets a Git credential as the user's default
func (s *Service) SetDefaultGitCredential(ctx context.Context, userID, credentialID int64) error {
	// Verify ownership
	_, err := s.GetGitCredential(ctx, userID, credentialID)
	if err != nil {
		return err
	}

	// Start transaction
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Clear all defaults for this user
		if err := tx.Model(&user.GitCredential{}).
			Where("user_id = ?", userID).
			Update("is_default", false).Error; err != nil {
			return err
		}

		// Set the new default
		if err := tx.Model(&user.GitCredential{}).
			Where("id = ? AND user_id = ?", credentialID, userID).
			Update("is_default", true).Error; err != nil {
			return err
		}

		// Update user's default credential reference
		return tx.Model(&user.User{}).
			Where("id = ?", userID).
			Update("default_git_credential_id", credentialID).Error
	})
}

// ClearDefaultGitCredential clears the user's default Git credential (falls back to runner_local)
func (s *Service) ClearDefaultGitCredential(ctx context.Context, userID int64) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Clear all is_default flags
		if err := tx.Model(&user.GitCredential{}).
			Where("user_id = ?", userID).
			Update("is_default", false).Error; err != nil {
			return err
		}

		// Clear user's default credential reference
		return tx.Model(&user.User{}).
			Where("id = ?", userID).
			Update("default_git_credential_id", nil).Error
	})
}

// GetDefaultGitCredential returns the user's default Git credential
// Returns nil if no default is set (meaning runner_local should be used)
func (s *Service) GetDefaultGitCredential(ctx context.Context, userID int64) (*user.GitCredential, error) {
	var credential user.GitCredential
	err := s.db.WithContext(ctx).
		Preload("RepositoryProvider").
		Where("user_id = ? AND is_default = ?", userID, true).
		First(&credential).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // No default set, use runner_local
		}
		return nil, err
	}
	return &credential, nil
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
	case user.CredentialTypeRunnerLocal:
		// No credentials to decrypt
		return result, nil

	case user.CredentialTypeOAuth:
		if credential.RepositoryProviderID != nil {
			token, err := s.GetDecryptedProviderToken(ctx, userID, *credential.RepositoryProviderID)
			if err != nil {
				return nil, err
			}
			result.Token = token
		}

	case user.CredentialTypePAT:
		if credential.PATEncrypted != nil && *credential.PATEncrypted != "" {
			if s.encryptionKey != "" {
				decrypted, err := crypto.DecryptWithKey(*credential.PATEncrypted, s.encryptionKey)
				if err != nil {
					return nil, err
				}
				result.Token = decrypted
			} else {
				result.Token = *credential.PATEncrypted
			}
		}

	case user.CredentialTypeSSHKey:
		if credential.PrivateKeyEncrypted != nil && *credential.PrivateKeyEncrypted != "" {
			if s.encryptionKey != "" {
				decrypted, err := crypto.DecryptWithKey(*credential.PrivateKeyEncrypted, s.encryptionKey)
				if err != nil {
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
func (s *Service) CreateCredentialFromProvider(ctx context.Context, userID, providerID int64) (*user.GitCredential, error) {
	provider, err := s.GetRepositoryProvider(ctx, userID, providerID)
	if err != nil {
		return nil, err
	}

	// Create a credential linked to this provider
	return s.CreateGitCredential(ctx, userID, &CreateGitCredentialRequest{
		Name:                 provider.Name + " (OAuth)",
		CredentialType:       user.CredentialTypeOAuth,
		RepositoryProviderID: &providerID,
	})
}
