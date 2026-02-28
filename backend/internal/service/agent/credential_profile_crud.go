package agent

import (
	"context"
	"errors"
	"fmt"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	"gorm.io/gorm"
)

// CreateCredentialProfile creates a new credential profile for a user
func (s *CredentialProfileService) CreateCredentialProfile(ctx context.Context, userID int64, params *CreateCredentialProfileParams) (*agent.UserAgentCredentialProfile, error) {
	// Verify agent type exists
	if _, err := s.agentTypeService.GetAgentType(ctx, params.AgentTypeID); err != nil {
		return nil, err
	}

	// Check if profile with same name exists
	var existing agent.UserAgentCredentialProfile
	err := s.db.WithContext(ctx).
		Where("user_id = ? AND agent_type_id = ? AND name = ?", userID, params.AgentTypeID, params.Name).
		First(&existing).Error
	if err == nil {
		return nil, ErrCredentialProfileExists
	}

	// If setting as default, unset other defaults for this agent type
	if params.IsDefault {
		s.db.WithContext(ctx).Model(&agent.UserAgentCredentialProfile{}).
			Where("user_id = ? AND agent_type_id = ?", userID, params.AgentTypeID).
			Update("is_default", false)
	}

	// Encrypt credentials if provided
	var encryptedCreds agent.EncryptedCredentials
	if !params.IsRunnerHost && params.Credentials != nil {
		var err error
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

	if err := s.db.WithContext(ctx).Create(profile).Error; err != nil {
		return nil, err
	}

	// Reload with AgentType
	return s.GetCredentialProfile(ctx, userID, profile.ID)
}

// GetCredentialProfile returns a credential profile by ID
func (s *CredentialProfileService) GetCredentialProfile(ctx context.Context, userID, profileID int64) (*agent.UserAgentCredentialProfile, error) {
	var profile agent.UserAgentCredentialProfile
	err := s.db.WithContext(ctx).
		Preload("AgentType").
		Where("id = ? AND user_id = ?", profileID, userID).
		First(&profile).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCredentialProfileNotFound
		}
		return nil, err
	}
	return &profile, nil
}

// DeleteCredentialProfile deletes a credential profile
func (s *CredentialProfileService) DeleteCredentialProfile(ctx context.Context, userID, profileID int64) error {
	result := s.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", profileID, userID).
		Delete(&agent.UserAgentCredentialProfile{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrCredentialProfileNotFound
	}
	return nil
}
