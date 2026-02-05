package agentpod

import (
	"errors"

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
