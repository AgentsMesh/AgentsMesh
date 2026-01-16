package agentpod

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/pkg/crypto"
	"gorm.io/gorm"
)

var (
	ErrProviderNotFound    = errors.New("AI provider not found")
	ErrCredentialsNotFound = errors.New("credentials not found")
	ErrDecryptionFailed    = errors.New("failed to decrypt credentials")
	ErrInvalidCredentials  = errors.New("invalid credentials format")
)

// AIProviderService handles AI provider credential operations
type AIProviderService struct {
	db        *gorm.DB
	encryptor *crypto.Encryptor
}

// NewAIProviderService creates a new AI provider service
func NewAIProviderService(db *gorm.DB, encryptor *crypto.Encryptor) *AIProviderService {
	return &AIProviderService{
		db:        db,
		encryptor: encryptor,
	}
}

// GetUserDefaultCredentials returns the default credentials for a user and provider type
func (s *AIProviderService) GetUserDefaultCredentials(ctx context.Context, userID int64, providerType string) (map[string]string, error) {
	var provider agentpod.UserAIProvider
	err := s.db.WithContext(ctx).
		Where("user_id = ? AND provider_type = ? AND is_default = ? AND is_enabled = ?",
			userID, providerType, true, true).
		First(&provider).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProviderNotFound
		}
		return nil, err
	}

	return s.decryptCredentials(provider.EncryptedCredentials)
}

// GetAIProviderEnvVars returns AI provider credentials as environment variables for a user
// This retrieves the user's default provider credentials and formats them for PTY injection
func (s *AIProviderService) GetAIProviderEnvVars(ctx context.Context, userID int64) (map[string]string, error) {
	// Try to get default Claude credentials first
	credentials, err := s.GetUserDefaultCredentials(ctx, userID, agentpod.AIProviderTypeClaude)
	if err != nil {
		if errors.Is(err, ErrProviderNotFound) {
			return nil, nil // No credentials configured
		}
		return nil, err
	}

	return s.formatEnvVars(agentpod.AIProviderTypeClaude, credentials), nil
}

// GetAIProviderEnvVarsByID returns AI provider credentials as environment variables by provider ID
func (s *AIProviderService) GetAIProviderEnvVarsByID(ctx context.Context, providerID int64) (map[string]string, error) {
	var provider agentpod.UserAIProvider
	err := s.db.WithContext(ctx).
		Where("id = ? AND is_enabled = ?", providerID, true).
		First(&provider).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProviderNotFound
		}
		return nil, err
	}

	credentials, err := s.decryptCredentials(provider.EncryptedCredentials)
	if err != nil {
		return nil, err
	}

	return s.formatEnvVars(provider.ProviderType, credentials), nil
}

// GetUserProviders returns all AI providers for a user
func (s *AIProviderService) GetUserProviders(ctx context.Context, userID int64) ([]*agentpod.UserAIProvider, error) {
	var providers []*agentpod.UserAIProvider
	err := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("provider_type, name").
		Find(&providers).Error
	if err != nil {
		return nil, err
	}
	return providers, nil
}

// GetUserProvidersByType returns AI providers for a user filtered by type
func (s *AIProviderService) GetUserProvidersByType(ctx context.Context, userID int64, providerType string) ([]*agentpod.UserAIProvider, error) {
	var providers []*agentpod.UserAIProvider
	err := s.db.WithContext(ctx).
		Where("user_id = ? AND provider_type = ?", userID, providerType).
		Order("is_default DESC, name").
		Find(&providers).Error
	if err != nil {
		return nil, err
	}
	return providers, nil
}

// CreateUserProvider creates a new AI provider for a user
func (s *AIProviderService) CreateUserProvider(ctx context.Context, userID int64, providerType, name string, credentials map[string]string, isDefault bool) (*agentpod.UserAIProvider, error) {
	// Encrypt credentials
	encrypted, err := s.encryptCredentials(credentials)
	if err != nil {
		return nil, err
	}

	provider := &agentpod.UserAIProvider{
		UserID:               userID,
		ProviderType:         providerType,
		Name:                 name,
		IsDefault:            isDefault,
		IsEnabled:            true,
		EncryptedCredentials: encrypted,
	}

	// If this is set as default, clear other defaults for this provider type
	if isDefault {
		if err := s.clearDefaultProvider(ctx, userID, providerType); err != nil {
			return nil, err
		}
	}

	if err := s.db.WithContext(ctx).Create(provider).Error; err != nil {
		return nil, err
	}

	return provider, nil
}

// UpdateUserProvider updates an existing AI provider
func (s *AIProviderService) UpdateUserProvider(ctx context.Context, providerID int64, name string, credentials map[string]string, isDefault, isEnabled bool) (*agentpod.UserAIProvider, error) {
	var provider agentpod.UserAIProvider
	if err := s.db.WithContext(ctx).First(&provider, providerID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProviderNotFound
		}
		return nil, err
	}

	// Encrypt new credentials if provided
	if len(credentials) > 0 {
		encrypted, err := s.encryptCredentials(credentials)
		if err != nil {
			return nil, err
		}
		provider.EncryptedCredentials = encrypted
	}

	provider.Name = name
	provider.IsEnabled = isEnabled

	// Handle default flag
	if isDefault && !provider.IsDefault {
		if err := s.clearDefaultProvider(ctx, provider.UserID, provider.ProviderType); err != nil {
			return nil, err
		}
		provider.IsDefault = true
	} else if !isDefault {
		provider.IsDefault = false
	}

	if err := s.db.WithContext(ctx).Save(&provider).Error; err != nil {
		return nil, err
	}

	return &provider, nil
}

// DeleteUserProvider deletes an AI provider
func (s *AIProviderService) DeleteUserProvider(ctx context.Context, providerID int64) error {
	return s.db.WithContext(ctx).Delete(&agentpod.UserAIProvider{}, providerID).Error
}

// SetDefaultProvider sets a provider as the default for its type
func (s *AIProviderService) SetDefaultProvider(ctx context.Context, providerID int64) error {
	var provider agentpod.UserAIProvider
	if err := s.db.WithContext(ctx).First(&provider, providerID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrProviderNotFound
		}
		return err
	}

	// Clear other defaults
	if err := s.clearDefaultProvider(ctx, provider.UserID, provider.ProviderType); err != nil {
		return err
	}

	// Set this one as default
	return s.db.WithContext(ctx).Model(&provider).Update("is_default", true).Error
}

// GetProviderCredentials returns decrypted credentials for a provider
// This should only be used when the credentials need to be displayed/edited
func (s *AIProviderService) GetProviderCredentials(ctx context.Context, providerID int64) (map[string]string, error) {
	var provider agentpod.UserAIProvider
	if err := s.db.WithContext(ctx).First(&provider, providerID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProviderNotFound
		}
		return nil, err
	}

	return s.decryptCredentials(provider.EncryptedCredentials)
}

// clearDefaultProvider clears the default flag for all providers of a type
func (s *AIProviderService) clearDefaultProvider(ctx context.Context, userID int64, providerType string) error {
	return s.db.WithContext(ctx).
		Model(&agentpod.UserAIProvider{}).
		Where("user_id = ? AND provider_type = ?", userID, providerType).
		Update("is_default", false).Error
}

// decryptCredentials decrypts stored credentials
func (s *AIProviderService) decryptCredentials(encrypted string) (map[string]string, error) {
	if encrypted == "" {
		return nil, ErrCredentialsNotFound
	}

	var credentials map[string]string

	if s.encryptor != nil {
		decrypted, err := s.encryptor.Decrypt(encrypted)
		if err != nil {
			return nil, ErrDecryptionFailed
		}
		if err := json.Unmarshal([]byte(decrypted), &credentials); err != nil {
			return nil, ErrInvalidCredentials
		}
	} else {
		// Development mode: credentials stored as plain JSON
		if err := json.Unmarshal([]byte(encrypted), &credentials); err != nil {
			return nil, ErrInvalidCredentials
		}
	}

	return credentials, nil
}

// encryptCredentials encrypts credentials for storage
func (s *AIProviderService) encryptCredentials(credentials map[string]string) (string, error) {
	jsonBytes, err := json.Marshal(credentials)
	if err != nil {
		return "", err
	}

	if s.encryptor != nil {
		return s.encryptor.Encrypt(string(jsonBytes))
	}

	// Development mode: store as plain JSON
	return string(jsonBytes), nil
}

// formatEnvVars formats credentials as environment variables based on provider type
func (s *AIProviderService) formatEnvVars(providerType string, credentials map[string]string) map[string]string {
	envVars := make(map[string]string)

	mapping, ok := agentpod.ProviderEnvVarMapping[providerType]
	if !ok {
		return envVars
	}

	for credKey, envKey := range mapping {
		if value, exists := credentials[credKey]; exists && value != "" {
			envVars[envKey] = value
		}
	}

	return envVars
}

// ValidateCredentials validates credentials for a provider type
func (s *AIProviderService) ValidateCredentials(providerType string, credentials map[string]string) error {
	switch providerType {
	case agentpod.AIProviderTypeClaude:
		// Claude requires either api_key or auth_token
		if credentials["api_key"] == "" && credentials["auth_token"] == "" {
			return errors.New("Claude provider requires either api_key or auth_token")
		}
	case agentpod.AIProviderTypeOpenAI, agentpod.AIProviderTypeCodex:
		// OpenAI/Codex requires api_key
		if credentials["api_key"] == "" {
			return errors.New("OpenAI/Codex provider requires api_key")
		}
	case agentpod.AIProviderTypeGemini:
		// Gemini requires api_key
		if credentials["api_key"] == "" {
			return errors.New("Gemini provider requires api_key")
		}
	}
	return nil
}
