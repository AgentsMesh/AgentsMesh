package agent

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
)

// CreateCredentialProfile creates a new credential profile for a user
func (s *CredentialProfileService) CreateCredentialProfile(ctx context.Context, userID int64, params *CreateCredentialProfileParams) (*agent.UserAgentCredentialProfile, error) {
	// Verify agent exists
	if _, err := s.agentSvc.GetAgent(ctx, params.AgentSlug); err != nil {
		return nil, err
	}

	// Check if profile with same name exists
	exists, err := s.repo.NameExists(ctx, userID, params.AgentSlug, params.Name, nil)
	if err != nil {
		slog.Error("failed to check credential profile name existence", "user_id", userID, "agent_slug", params.AgentSlug, "error", err)
		return nil, err
	}
	if exists {
		return nil, ErrCredentialProfileExists
	}

	// If setting as default, unset other defaults for this agent
	if params.IsDefault {
		_ = s.repo.UnsetDefaults(ctx, userID, params.AgentSlug)
	}

	// Encrypt credentials if provided
	var encryptedCreds agent.EncryptedCredentials
	if !params.IsRunnerHost && params.Credentials != nil {
		encryptedCreds, err = s.encryptCredentials(params.Credentials)
		if err != nil {
			slog.Error("failed to encrypt credentials for new profile", "user_id", userID, "agent_slug", params.AgentSlug, "error", err)
			return nil, fmt.Errorf("encrypt credentials: %w", err)
		}
	}

	profile := &agent.UserAgentCredentialProfile{
		UserID:               userID,
		AgentSlug:          params.AgentSlug,
		Name:                 params.Name,
		Description:          params.Description,
		IsRunnerHost:         params.IsRunnerHost,
		CredentialsEncrypted: encryptedCreds,
		IsDefault:            params.IsDefault,
		IsActive:             true,
	}

	if err := s.repo.Create(ctx, profile); err != nil {
		slog.Error("failed to create credential profile", "user_id", userID, "agent_slug", params.AgentSlug, "error", err)
		return nil, err
	}

	slog.Info("credential profile created", "user_id", userID, "profile_id", profile.ID, "agent_slug", params.AgentSlug)
	// Reload with Agent
	return s.GetCredentialProfile(ctx, userID, profile.ID)
}

// GetCredentialProfile returns a credential profile by ID
func (s *CredentialProfileService) GetCredentialProfile(ctx context.Context, userID, profileID int64) (*agent.UserAgentCredentialProfile, error) {
	profile, err := s.repo.GetWithAgent(ctx, userID, profileID)
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
		slog.Error("failed to delete credential profile", "user_id", userID, "profile_id", profileID, "error", err)
		return err
	}
	if rowsAffected == 0 {
		return ErrCredentialProfileNotFound
	}
	slog.Info("credential profile deleted", "user_id", userID, "profile_id", profileID)
	return nil
}
