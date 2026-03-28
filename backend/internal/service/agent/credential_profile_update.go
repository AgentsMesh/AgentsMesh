package agent

import (
	"context"
	"fmt"
	"log/slog"

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
		exists, err := s.repo.NameExists(ctx, userID, profile.AgentSlug, *params.Name, &profileID)
		if err != nil {
			slog.Error("failed to check credential profile name uniqueness", "user_id", userID, "profile_id", profileID, "error", err)
			return nil, err
		}
		if exists {
			return nil, ErrCredentialProfileExists
		}
	}

	// If setting as default, unset other defaults
	if params.IsDefault != nil && *params.IsDefault && !profile.IsDefault {
		_ = s.repo.UnsetDefaults(ctx, userID, profile.AgentSlug)
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
			slog.Error("failed to encrypt credential profile credentials", "user_id", userID, "profile_id", profileID, "error", err)
			return nil, fmt.Errorf("encrypt credentials: %w", err)
		}
		updates["credentials_encrypted"] = encryptedCreds
	}

	if len(updates) > 0 {
		if err := s.repo.Update(ctx, profile, updates); err != nil {
			slog.Error("failed to update credential profile", "user_id", userID, "profile_id", profileID, "error", err)
			return nil, err
		}
	}

	slog.Info("credential profile updated", "user_id", userID, "profile_id", profileID)
	return s.GetCredentialProfile(ctx, userID, profileID)
}

// SetDefaultCredentialProfile sets a profile as the default for its agent
func (s *CredentialProfileService) SetDefaultCredentialProfile(ctx context.Context, userID, profileID int64) (*agent.UserAgentCredentialProfile, error) {
	profile, err := s.GetCredentialProfile(ctx, userID, profileID)
	if err != nil {
		return nil, err
	}

	// Unset other defaults
	_ = s.repo.UnsetDefaults(ctx, userID, profile.AgentSlug)

	// Set this as default
	if err := s.repo.SetDefault(ctx, profile); err != nil {
		slog.Error("failed to set default credential profile", "user_id", userID, "profile_id", profileID, "error", err)
		return nil, err
	}

	slog.Info("credential profile set as default", "user_id", userID, "profile_id", profileID, "agent_slug", profile.AgentSlug)
	return s.GetCredentialProfile(ctx, userID, profileID)
}
