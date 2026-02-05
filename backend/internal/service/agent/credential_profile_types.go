package agent

import (
	"context"
	"errors"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
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
}

// NewCredentialProfileService creates a new credential profile service
func NewCredentialProfileService(db *gorm.DB, agentTypeService AgentTypeProvider) *CredentialProfileService {
	return &CredentialProfileService{
		db:               db,
		agentTypeService: agentTypeService,
	}
}
