package agent

import (
	"context"
	"fmt"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
)

// CreateCredentialProfile creates a new credential profile for a user
func (s *CredentialProfileService) CreateCredentialProfile(ctx context.Context, userID int64, params *CreateCredentialProfileParams) (*agent.UserAgentCredentialProfile, error) {
	// Verify agent type exists
	if _, err := s.agentTypeService.GetAgentType(ctx, params.AgentTypeID); err != nil {
		return nil, err
	}

	// Check if profile with same name exists
	exists, err := s.repo.NameExists(ctx, userID, params.AgentTypeID, params.Name, nil)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrCredentialProfileExists
	}

	// If setting as default, unset other defaults for this agent type
	if params.IsDefault {
		_ = s.repo.UnsetDefaults(ctx, userID, params.AgentTypeID)
	}

	// Encrypt credentials if provided
	var encryptedCreds agent.EncryptedCredentials
	if !params.IsRunnerHost && params.Credentials != nil {
		encryptedCreds, err = s.encryptCredentials(params.Credentials)
		if err != nil {
			return nil, fmt.Errorf("encrypt credentials: %w", err)
		}
	}

	profile := &agent.UserAgentCredentialProfile{
		UserID:               userID,
		AgentTypeID:          params.AgentTypeID,
		Name:                 params.Name,
		Description:          params.Description,
		IsRunnerHost:         params.IsRunnerHost,
		CredentialsEncrypted: encryptedCreds,
		IsDefault:            params.IsDefault,
		IsActive:             true,
	}

	if err := s.repo.Create(ctx, profile); err != nil {
		return nil, err
	}

	// Reload with AgentType
	return s.GetCredentialProfile(ctx, userID, profile.ID)
}

// GetCredentialProfile returns a credential profile by ID
func (s *CredentialProfileService) GetCredentialProfile(ctx context.Context, userID, profileID int64) (*agent.UserAgentCredentialProfile, error) {
	profile, err := s.repo.GetWithAgentType(ctx, userID, profileID)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, ErrCredentialProfileNotFound
	}
	return profile, nil
}

// DeleteCredentialProfile deletes a credential profile
func (s *CredentialProfileService) DeleteCredentialProfile(ctx context.Context, userID, profileID int64) error {
	rowsAffected, err := s.repo.Delete(ctx, userID, profileID)
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrCredentialProfileNotFound
	}
	return nil
}
