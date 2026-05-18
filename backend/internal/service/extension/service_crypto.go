package extension

import (
	"fmt"
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
)

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
