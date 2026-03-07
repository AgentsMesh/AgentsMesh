package agentpod

import (
	"encoding/json"
	"errors"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
)

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
			return errors.New("claude provider requires either api_key or auth_token")
		}
	case agentpod.AIProviderTypeOpenAI, agentpod.AIProviderTypeCodex:
		// OpenAI/Codex requires api_key
		if credentials["api_key"] == "" {
			return errors.New("OpenAI/Codex provider requires api_key")
		}
	case agentpod.AIProviderTypeGemini:
		// Gemini requires api_key
		if credentials["api_key"] == "" {
			return errors.New("gemini provider requires api_key")
		}
	}
	return nil
}
