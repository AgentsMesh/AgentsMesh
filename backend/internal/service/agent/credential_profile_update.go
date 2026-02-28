package agent

import (
	"context"
	"fmt"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
)

// UpdateCredentialProfile updates an existing credential profile
func (s *CredentialProfileService) UpdateCredentialProfile(ctx context.Context, userID, profileID int64, params *UpdateCredentialProfileParams) (*agent.UserAgentCredentialProfile, error) {
	profile, err := s.GetCredentialProfile(ctx, userID, profileID)
	if err != nil {
		return nil, err
	}

	// Check name uniqueness if changing
	if params.Name != nil && *params.Name != profile.Name {
		var existing agent.UserAgentCredentialProfile
		err := s.db.WithContext(ctx).
			Where("user_id = ? AND agent_type_id = ? AND name = ? AND id != ?", userID, profile.AgentTypeID, *params.Name, profileID).
			First(&existing).Error
		if err == nil {
			return nil, ErrCredentialProfileExists
		}
	}

	// If setting as default, unset other defaults
	if params.IsDefault != nil && *params.IsDefault && !profile.IsDefault {
		s.db.WithContext(ctx).Model(&agent.UserAgentCredentialProfile{}).
			Where("user_id = ? AND agent_type_id = ? AND id != ?", userID, profile.AgentTypeID, profileID).
			Update("is_default", false)
	}

	// Build updates
	updates := make(map[string]interface{})
	if params.Name != nil {
		updates["name"] = *params.Name
	}
	if params.Description != nil {
		updates["description"] = *params.Description
	}
	if params.IsRunnerHost != nil {
		updates["is_runner_host"] = *params.IsRunnerHost
		if *params.IsRunnerHost {
			// Clear credentials when switching to RunnerHost
			updates["credentials_encrypted"] = nil
		}
	}
	if params.IsDefault != nil {
		updates["is_default"] = *params.IsDefault
	}
	if params.IsActive != nil {
		updates["is_active"] = *params.IsActive
	}

	// Update credentials if provided
	if params.Credentials != nil {
		encryptedCreds, err := s.encryptCredentials(params.Credentials)
		if err != nil {
			return nil, fmt.Errorf("encrypt credentials: %w", err)
		}
		updates["credentials_encrypted"] = encryptedCreds
	}

	if len(updates) > 0 {
		if err := s.db.WithContext(ctx).Model(profile).Updates(updates).Error; err != nil {
			return nil, err
		}
	}

	return s.GetCredentialProfile(ctx, userID, profileID)
}

// SetDefaultCredentialProfile sets a profile as the default for its agent type
func (s *CredentialProfileService) SetDefaultCredentialProfile(ctx context.Context, userID, profileID int64) (*agent.UserAgentCredentialProfile, error) {
	profile, err := s.GetCredentialProfile(ctx, userID, profileID)
	if err != nil {
		return nil, err
	}

	// Unset other defaults
	s.db.WithContext(ctx).Model(&agent.UserAgentCredentialProfile{}).
		Where("user_id = ? AND agent_type_id = ? AND id != ?", userID, profile.AgentTypeID, profileID).
		Update("is_default", false)

	// Set this as default
	if err := s.db.WithContext(ctx).Model(profile).Update("is_default", true).Error; err != nil {
		return nil, err
	}

	return s.GetCredentialProfile(ctx, userID, profileID)
}
