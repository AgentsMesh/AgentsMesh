package agent

import (
	"context"
	"errors"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	"github.com/anthropics/agentsmesh/backend/pkg/crypto"
	"gorm.io/gorm"
)

// Errors for CredentialProfileService
var (
	ErrCredentialProfileNotFound = errors.New("credential profile not found")
	ErrCredentialProfileExists   = errors.New("credential profile with this name already exists")
	ErrCredentialsRequired       = errors.New("required credentials missing")
)

// AgentTypeProvider provides agent type lookup for credential profile operations
type AgentTypeProvider interface {
	GetAgentType(ctx context.Context, id int64) (*agent.AgentType, error)
}

// CredentialProfileService handles user credential profile operations
type CredentialProfileService struct {
	db               *gorm.DB
	agentTypeService AgentTypeProvider
	encryptor        *crypto.Encryptor
}

// NewCredentialProfileService creates a new credential profile service
func NewCredentialProfileService(db *gorm.DB, agentTypeService AgentTypeProvider, encryptor *crypto.Encryptor) *CredentialProfileService {
	return &CredentialProfileService{
		db:               db,
		agentTypeService: agentTypeService,
		encryptor:        encryptor,
	}
}

// encryptCredentials encrypts a map of plaintext credentials
func (s *CredentialProfileService) encryptCredentials(creds map[string]string) (agent.EncryptedCredentials, error) {
	encrypted := make(agent.EncryptedCredentials, len(creds))
	for k, v := range creds {
		enc, err := s.encryptor.Encrypt(v)
		if err != nil {
			return nil, err
		}
		encrypted[k] = enc
	}
	return encrypted, nil
}

// decryptCredentials decrypts a map of encrypted credentials
func (s *CredentialProfileService) decryptCredentials(creds agent.EncryptedCredentials) (agent.EncryptedCredentials, error) {
	decrypted := make(agent.EncryptedCredentials, len(creds))
	for k, v := range creds {
		dec, err := s.encryptor.Decrypt(v)
		if err != nil {
			return nil, err
		}
		decrypted[k] = dec
	}
	return decrypted, nil
}

// ProfileToResponse converts a profile to API response, decrypting text field values
// in ConfiguredValues. Secret field values remain hidden (only field names in ConfiguredFields).
func (s *CredentialProfileService) ProfileToResponse(p *agent.UserAgentCredentialProfile) *agent.CredentialProfileResponse {
	resp := p.ToResponse()

	// Decrypt text field values in ConfiguredValues (they are stored encrypted)
	if resp.ConfiguredValues != nil {
		for k, v := range resp.ConfiguredValues {
			dec, err := s.encryptor.Decrypt(v)
			if err == nil {
				resp.ConfiguredValues[k] = dec
			}
			// If decryption fails (e.g., value was not encrypted), keep original
		}
	}

	return resp
}
