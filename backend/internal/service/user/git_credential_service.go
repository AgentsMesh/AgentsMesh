package user

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
	"github.com/anthropics/agentsmesh/backend/pkg/crypto"
	"golang.org/x/crypto/ssh"
	"gorm.io/gorm"
)

var (
	ErrCredentialNotFound      = errors.New("git credential not found")
	ErrCredentialAlreadyExists = errors.New("git credential already exists with this name")
	ErrInvalidCredentialType   = errors.New("invalid credential type")
	ErrInvalidSSHKey           = errors.New("invalid SSH key format")
	ErrProviderIDRequired      = errors.New("repository_provider_id is required for oauth type")
)

// CreateGitCredentialRequest represents a request to create a Git credential
type CreateGitCredentialRequest struct {
	Name                 string
	CredentialType       string // runner_local, oauth, pat, ssh_key
	RepositoryProviderID *int64 // Required for oauth type
	PAT                  string // For pat type
	PublicKey            string // For ssh_key type (can be generated)
	PrivateKey           string // For ssh_key type
	HostPattern          string // Optional host pattern
}

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
	switch req.CredentialType {
	case user.CredentialTypeRunnerLocal:
		// No additional fields needed for runner_local
		break

	case user.CredentialTypeOAuth:
		if req.RepositoryProviderID == nil {
			return nil, ErrProviderIDRequired
		}
		// Verify the provider exists and belongs to the user
		_, err := s.GetRepositoryProvider(ctx, userID, *req.RepositoryProviderID)
		if err != nil {
			return nil, err
		}
		credential.RepositoryProviderID = req.RepositoryProviderID

	case user.CredentialTypePAT:
		if req.PAT == "" {
			return nil, errors.New("PAT is required for pat type")
		}
		// Encrypt PAT
		if s.encryptionKey != "" {
			encrypted, err := crypto.EncryptWithKey(req.PAT, s.encryptionKey)
			if err != nil {
				return nil, err
			}
			credential.PATEncrypted = &encrypted
		} else {
			credential.PATEncrypted = &req.PAT
		}

	case user.CredentialTypeSSHKey:
		if req.PrivateKey == "" {
			return nil, errors.New("private key is required for ssh_key type")
		}

		// Parse and validate SSH key
		privateKey, publicKey, fingerprint, err := parseSSHKey(req.PrivateKey, req.PublicKey)
		if err != nil {
			return nil, err
		}

		credential.PublicKey = &publicKey
		credential.Fingerprint = &fingerprint

		// Encrypt private key
		if s.encryptionKey != "" {
			encrypted, err := crypto.EncryptWithKey(privateKey, s.encryptionKey)
			if err != nil {
				return nil, err
			}
			credential.PrivateKeyEncrypted = &encrypted
		} else {
			credential.PrivateKeyEncrypted = &privateKey
		}
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

// parseSSHKey validates and parses an SSH key, returns private key, public key, and fingerprint
func parseSSHKey(privateKeyPEM, publicKeyStr string) (string, string, string, error) {
	// Parse the private key
	signer, err := ssh.ParsePrivateKey([]byte(privateKeyPEM))
	if err != nil {
		return "", "", "", ErrInvalidSSHKey
	}

	// Get public key from private key
	pubKey := signer.PublicKey()
	publicKey := string(ssh.MarshalAuthorizedKey(pubKey))
	publicKey = strings.TrimSpace(publicKey)

	// Calculate fingerprint (SHA256)
	hash := sha256.Sum256(pubKey.Marshal())
	fingerprint := "SHA256:" + hex.EncodeToString(hash[:])

	return privateKeyPEM, publicKey, fingerprint, nil
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

// UpdateGitCredentialRequest represents a request to update a Git credential
type UpdateGitCredentialRequest struct {
	Name        *string
	PAT         *string // For pat type
	PrivateKey  *string // For ssh_key type
	HostPattern *string
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
	if req.PAT != nil && credential.CredentialType == user.CredentialTypePAT {
		if *req.PAT == "" {
			updates["pat_encrypted"] = nil
		} else if s.encryptionKey != "" {
			encrypted, err := crypto.EncryptWithKey(*req.PAT, s.encryptionKey)
			if err != nil {
				return nil, err
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
				return nil, err
			}

			updates["public_key"] = publicKey
			updates["fingerprint"] = fingerprint

			if s.encryptionKey != "" {
				encrypted, err := crypto.EncryptWithKey(privateKey, s.encryptionKey)
				if err != nil {
					return nil, err
				}
				updates["private_key_encrypted"] = encrypted
			} else {
				updates["private_key_encrypted"] = privateKey
			}
		}
	}

	if len(updates) == 0 {
		return credential, nil
	}

	if err := s.db.WithContext(ctx).Model(credential).Updates(updates).Error; err != nil {
		return nil, err
	}

	return s.GetGitCredential(ctx, userID, credentialID)
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

// DecryptedCredential holds decrypted credential information
type DecryptedCredential struct {
	Type          string // runner_local, oauth, pat, ssh_key
	Token         string // For oauth and pat types
	SSHPrivateKey string // For ssh_key type
	SSHPublicKey  string // For ssh_key type
}

// GenerateSSHKeyPair generates a new SSH key pair
func GenerateSSHKeyPair() (privateKey, publicKey string, err error) {
	// Generate ED25519 key (more secure and shorter than RSA)
	pubKey, privKey, err := generateED25519Key()
	if err != nil {
		return "", "", err
	}
	return privKey, pubKey, nil
}

// generateED25519Key generates an ED25519 SSH key pair
func generateED25519Key() (publicKey, privateKey string, err error) {
	// Generate random seed
	seed := make([]byte, 32)
	if _, err := rand.Read(seed); err != nil {
		return "", "", err
	}

	// For simplicity, we'll return an error asking user to provide their own key
	// Full ED25519 key generation would require additional dependencies
	return "", "", errors.New("SSH key generation not implemented - please provide your own key")
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
