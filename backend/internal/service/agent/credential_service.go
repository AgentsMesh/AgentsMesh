package agent

import (
	"context"
	"errors"

	"github.com/anthropics/agentmesh/backend/internal/domain/agent"
	"github.com/anthropics/agentmesh/backend/pkg/crypto"
	"gorm.io/gorm"
)

var (
	ErrCredentialsNotFound = errors.New("credentials not found")
	ErrDecryptionFailed    = errors.New("failed to decrypt credentials")
)

// CredentialService handles AI provider credential operations
type CredentialService struct {
	db        *gorm.DB
	encryptor *crypto.Encryptor
}

// NewCredentialService creates a new credential service
func NewCredentialService(db *gorm.DB, encryptor *crypto.Encryptor) *CredentialService {
	return &CredentialService{
		db:        db,
		encryptor: encryptor,
	}
}

// ClaudeCredentials represents decrypted Claude credentials
type ClaudeCredentials struct {
	BaseURL   string `json:"base_url,omitempty"`
	AuthToken string `json:"auth_token,omitempty"`
	APIKey    string `json:"api_key,omitempty"`
}

// GetUserCredentials returns decrypted credentials for a user's specific agent type
func (s *CredentialService) GetUserCredentials(ctx context.Context, userID, agentTypeID int64) (map[string]string, error) {
	var cred agent.UserAgentCredential
	err := s.db.WithContext(ctx).
		Where("user_id = ? AND agent_type_id = ?", userID, agentTypeID).
		First(&cred).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCredentialsNotFound
		}
		return nil, err
	}

	if cred.CredentialsEncrypted == nil {
		return nil, ErrCredentialsNotFound
	}

	return s.decryptCredentialsMap(cred.CredentialsEncrypted)
}


// SetUserCredentials encrypts and stores credentials for a user
func (s *CredentialService) SetUserCredentials(ctx context.Context, userID, agentTypeID int64, credentials map[string]string) error {
	encrypted, err := s.encryptCredentialsMap(credentials)
	if err != nil {
		return err
	}

	cred := agent.UserAgentCredential{
		UserID:               userID,
		AgentTypeID:          agentTypeID,
		CredentialsEncrypted: encrypted,
	}

	// Upsert
	return s.db.WithContext(ctx).
		Where("user_id = ? AND agent_type_id = ?", userID, agentTypeID).
		Assign(agent.UserAgentCredential{CredentialsEncrypted: encrypted}).
		FirstOrCreate(&cred).Error
}


// DeleteUserCredentials removes credentials for a user
func (s *CredentialService) DeleteUserCredentials(ctx context.Context, userID, agentTypeID int64) error {
	return s.db.WithContext(ctx).
		Where("user_id = ? AND agent_type_id = ?", userID, agentTypeID).
		Delete(&agent.UserAgentCredential{}).Error
}

// decryptCredentialsMap decrypts the stored credentials from EncryptedCredentials type
// For this implementation, credentials are stored as-is in the map (individual values may be encrypted)
func (s *CredentialService) decryptCredentialsMap(encrypted agent.EncryptedCredentials) (map[string]string, error) {
	if encrypted == nil {
		return nil, ErrCredentialsNotFound
	}

	// If encryptor is configured, decrypt each value
	if s.encryptor != nil {
		result := make(map[string]string)
		for key, value := range encrypted {
			decrypted, err := s.encryptor.Decrypt(value)
			if err != nil {
				// If decryption fails, try using raw value (for backward compatibility)
				result[key] = value
			} else {
				result[key] = decrypted
			}
		}
		return result, nil
	}

	// Return as-is if no encryptor
	return map[string]string(encrypted), nil
}

// encryptCredentialsMap encrypts credentials to EncryptedCredentials type
func (s *CredentialService) encryptCredentialsMap(credentials map[string]string) (agent.EncryptedCredentials, error) {
	if s.encryptor == nil {
		// If no encryptor, store as plain map (for development)
		return agent.EncryptedCredentials(credentials), nil
	}

	// Encrypt each value
	result := make(agent.EncryptedCredentials)
	for key, value := range credentials {
		encrypted, err := s.encryptor.Encrypt(value)
		if err != nil {
			return nil, err
		}
		result[key] = encrypted
	}

	return result, nil
}
