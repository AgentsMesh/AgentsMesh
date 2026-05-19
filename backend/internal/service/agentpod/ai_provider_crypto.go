package agentpod

import (
	"encoding/json"
	"errors"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
)

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
		if err := json.Unmarshal([]byte(encrypted), &credentials); err != nil {
			return nil, ErrInvalidCredentials
		}
	}

	return credentials, nil
}

func (s *AIProviderService) encryptCredentials(credentials map[string]string) (string, error) {
	jsonBytes, err := json.Marshal(credentials)
	if err != nil {
		return "", err
	}

	if s.encryptor != nil {
		return s.encryptor.Encrypt(string(jsonBytes))
	}

	return string(jsonBytes), nil
}

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

func (s *AIProviderService) ValidateCredentials(providerType string, credentials map[string]string) error {
	switch providerType {
	case agentpod.AIProviderTypeClaude:
		if credentials["api_key"] == "" && credentials["auth_token"] == "" {
			return errors.New("claude provider requires either api_key or auth_token")
		}
	case agentpod.AIProviderTypeOpenAI, agentpod.AIProviderTypeCodex:
		if credentials["api_key"] == "" {
			return errors.New("OpenAI/Codex provider requires api_key")
		}
	case agentpod.AIProviderTypeGemini:
		if credentials["api_key"] == "" {
			return errors.New("gemini provider requires api_key")
		}
	}
	return nil
}
