package agent

import (
	"context"
	"errors"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	"github.com/anthropics/agentsmesh/backend/pkg/crypto"
)

var (
	ErrCredentialProfileNotFound = errors.New("credential profile not found")
	ErrCredentialProfileExists   = errors.New("credential profile with this name already exists")
	ErrCredentialsRequired       = errors.New("required credentials missing")
)

type AgentProvider interface {
	GetAgent(ctx context.Context, slug string) (*agent.Agent, error)
}

type CredentialProfileService struct {
	repo         agent.CredentialProfileRepository
	agentSvc     AgentProvider
	encryptor        *crypto.Encryptor
}

func NewCredentialProfileService(repo agent.CredentialProfileRepository, agentSvc AgentProvider, encryptor *crypto.Encryptor) *CredentialProfileService {
	return &CredentialProfileService{
		repo:      repo,
		agentSvc:  agentSvc,
		encryptor: encryptor,
	}
}

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

func (s *CredentialProfileService) ProfileToResponse(p *agent.UserAgentCredentialProfile) *agent.CredentialProfileResponse {
	resp := p.ToResponse()

	if resp.ConfiguredValues != nil {
		for k, v := range resp.ConfiguredValues {
			dec, err := s.encryptor.Decrypt(v)
			if err == nil {
				resp.ConfiguredValues[k] = dec
			}
		}
	}

	return resp
}
