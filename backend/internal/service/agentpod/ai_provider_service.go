package agentpod

import (
	"errors"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/pkg/crypto"
)

var (
	ErrProviderNotFound    = errors.New("AI provider not found")
	ErrCredentialsNotFound = errors.New("credentials not found")
	ErrDecryptionFailed    = errors.New("failed to decrypt credentials")
	ErrInvalidCredentials  = errors.New("invalid credentials format")
)

// AIProviderService handles AI provider credential operations
type AIProviderService struct {
	repo      agentpod.AIProviderRepository
	encryptor *crypto.Encryptor
}

// NewAIProviderService creates a new AI provider service
func NewAIProviderService(repo agentpod.AIProviderRepository, encryptor *crypto.Encryptor) *AIProviderService {
	return &AIProviderService{
		repo:      repo,
		encryptor: encryptor,
	}
}
