package user

import (
	"context"
	"errors"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/user"
	"github.com/AgentsMesh/AgentsMesh/backend/pkg/crypto"
	"gorm.io/gorm"
)

// CreateGitCredential creates a new Git credential for a user
func (s *Service) CreateGitCredential(ctx context.Context, userID int64, req *CreateGitCredentialRequest) (*user.GitCredential, error) {
	// Validate credential type
	if !user.IsValidCredentialType(req.CredentialType) {
		return nil, ErrInvalidCredentialType
	}

	// Check if credential with same name already exists
	var existing user.GitCredential
	err := s.db.WithContext(ctx).
		Where("user_id = ? AND name = ?", userID, req.Name).
		First(&existing).Error
	if err == nil {
		return nil, ErrCredentialAlreadyExists
	}

	credential := &user.GitCredential{
		UserID:         userID,
		Name:           req.Name,
		CredentialType: req.CredentialType,
		IsDefault:      false,
	}

	// Type-specific validation and processing
	if err := s.processCredentialType(ctx, userID, credential, req); err != nil {
		return nil, err
	}

	// Set optional host pattern
	if req.HostPattern != "" {
		credential.HostPattern = &req.HostPattern
	}

	if err := s.db.WithContext(ctx).Create(credential).Error; err != nil {
		return nil, err
	}

	return credential, nil
}

// processCredentialType handles type-specific validation and field setting
func (s *Service) processCredentialType(ctx context.Context, userID int64, credential *user.GitCredential, req *CreateGitCredentialRequest) error {
	switch req.CredentialType {
	case user.CredentialTypeRunnerLocal:
		// No additional fields needed for runner_local
		return nil

	case user.CredentialTypeOAuth:
		if req.RepositoryProviderID == nil {
			return ErrProviderIDRequired
		}
		// Verify the provider exists and belongs to the user
		_, err := s.GetRepositoryProvider(ctx, userID, *req.RepositoryProviderID)
		if err != nil {
			return err
		}
		credential.RepositoryProviderID = req.RepositoryProviderID

	case user.CredentialTypePAT:
		if req.PAT == "" {
			return errors.New("PAT is required for pat type")
		}
		// Encrypt PAT
		if s.encryptionKey != "" {
			encrypted, err := crypto.EncryptWithKey(req.PAT, s.encryptionKey)
			if err != nil {
				return err
			}
			credential.PATEncrypted = &encrypted
		} else {
			credential.PATEncrypted = &req.PAT
		}

	case user.CredentialTypeSSHKey:
		if req.PrivateKey == "" {
			return errors.New("private key is required for ssh_key type")
		}

		// Parse and validate SSH key
		privateKey, publicKey, fingerprint, err := parseSSHKey(req.PrivateKey, req.PublicKey)
		if err != nil {
			return err
		}

		credential.PublicKey = &publicKey
		credential.Fingerprint = &fingerprint

		// Encrypt private key
		if s.encryptionKey != "" {
			encrypted, err := crypto.EncryptWithKey(privateKey, s.encryptionKey)
			if err != nil {
				return err
			}
			credential.PrivateKeyEncrypted = &encrypted
		} else {
			credential.PrivateKeyEncrypted = &privateKey
		}
	}

	return nil
}

// GetGitCredential returns a Git credential by ID
func (s *Service) GetGitCredential(ctx context.Context, userID, credentialID int64) (*user.GitCredential, error) {
	var credential user.GitCredential
	err := s.db.WithContext(ctx).
		Preload("RepositoryProvider").
		Where("id = ? AND user_id = ?", credentialID, userID).
		First(&credential).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCredentialNotFound
		}
		return nil, err
	}
	return &credential, nil
}

// ListGitCredentials returns all Git credentials for a user
func (s *Service) ListGitCredentials(ctx context.Context, userID int64) ([]*user.GitCredential, error) {
	var credentials []*user.GitCredential
	err := s.db.WithContext(ctx).
		Preload("RepositoryProvider").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&credentials).Error
	return credentials, err
}

// UpdateGitCredential updates a Git credential
func (s *Service) UpdateGitCredential(ctx context.Context, userID, credentialID int64, req *UpdateGitCredentialRequest) (*user.GitCredential, error) {
	// Verify ownership
	credential, err := s.GetGitCredential(ctx, userID, credentialID)
	if err != nil {
		return nil, err
	}

	updates := make(map[string]interface{})

	if req.Name != nil && *req.Name != "" {
		// Check if new name conflicts
		var existing user.GitCredential
		err := s.db.WithContext(ctx).
			Where("user_id = ? AND name = ? AND id != ?", userID, *req.Name, credentialID).
			First(&existing).Error
		if err == nil {
			return nil, ErrCredentialAlreadyExists
		}
		updates["name"] = *req.Name
	}

	if req.HostPattern != nil {
		if *req.HostPattern == "" {
			updates["host_pattern"] = nil
		} else {
			updates["host_pattern"] = *req.HostPattern
		}
	}

	// Type-specific updates
	if err := s.applyCredentialTypeUpdates(credential, req, updates); err != nil {
		return nil, err
	}

	if len(updates) == 0 {
		return credential, nil
	}

	if err := s.db.WithContext(ctx).Model(credential).Updates(updates).Error; err != nil {
		return nil, err
	}

	return s.GetGitCredential(ctx, userID, credentialID)
}

// applyCredentialTypeUpdates applies type-specific updates to the updates map
func (s *Service) applyCredentialTypeUpdates(credential *user.GitCredential, req *UpdateGitCredentialRequest, updates map[string]interface{}) error {
	if req.PAT != nil && credential.CredentialType == user.CredentialTypePAT {
		if *req.PAT == "" {
			updates["pat_encrypted"] = nil
		} else if s.encryptionKey != "" {
			encrypted, err := crypto.EncryptWithKey(*req.PAT, s.encryptionKey)
			if err != nil {
				return err
			}
			updates["pat_encrypted"] = encrypted
		} else {
			updates["pat_encrypted"] = *req.PAT
		}
	}

	if req.PrivateKey != nil && credential.CredentialType == user.CredentialTypeSSHKey {
		if *req.PrivateKey == "" {
			updates["private_key_encrypted"] = nil
			updates["public_key"] = nil
			updates["fingerprint"] = nil
		} else {
			privateKey, publicKey, fingerprint, err := parseSSHKey(*req.PrivateKey, "")
			if err != nil {
				return err
			}

			updates["public_key"] = publicKey
			updates["fingerprint"] = fingerprint

			if s.encryptionKey != "" {
				encrypted, err := crypto.EncryptWithKey(privateKey, s.encryptionKey)
				if err != nil {
					return err
				}
				updates["private_key_encrypted"] = encrypted
			} else {
				updates["private_key_encrypted"] = privateKey
			}
		}
	}

	return nil
}

// DeleteGitCredential deletes a Git credential
func (s *Service) DeleteGitCredential(ctx context.Context, userID, credentialID int64) error {
	// First check if this is the default credential
	var credential user.GitCredential
	err := s.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", credentialID, userID).
		First(&credential).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrCredentialNotFound
		}
		return err
	}

	// If this is the default, clear user's default credential reference
	if credential.IsDefault {
		if err := s.db.WithContext(ctx).
			Model(&user.User{}).
			Where("id = ? AND default_git_credential_id = ?", userID, credentialID).
			Update("default_git_credential_id", nil).Error; err != nil {
			return err
		}
	}

	result := s.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", credentialID, userID).
		Delete(&user.GitCredential{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrCredentialNotFound
	}
	return nil
}
