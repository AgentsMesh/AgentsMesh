package agent

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
)

func (s *CredentialProfileService) UpdateCredentialProfile(ctx context.Context, userID, profileID int64, params *UpdateCredentialProfileParams) (*agent.UserAgentCredentialProfile, error) {
	profile, err := s.GetCredentialProfile(ctx, userID, profileID)
	if err != nil {
		return nil, err
	}

	if params.Name != nil && *params.Name != profile.Name {
		exists, err := s.repo.NameExists(ctx, userID, profile.AgentSlug, *params.Name, &profileID)
		if err != nil {
			slog.ErrorContext(ctx, "failed to check credential profile name uniqueness", "user_id", userID, "profile_id", profileID, "error", err)
			return nil, err
		}
		if exists {
			return nil, ErrCredentialProfileExists
		}
	}

	if params.IsDefault != nil && *params.IsDefault && !profile.IsDefault {
		_ = s.repo.UnsetDefaults(ctx, userID, profile.AgentSlug)
	}

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
			updates["credentials_encrypted"] = nil
		}
	}
	if params.IsDefault != nil {
		updates["is_default"] = *params.IsDefault
	}
	if params.IsActive != nil {
		updates["is_active"] = *params.IsActive
	}

	if params.Credentials != nil {
		encryptedCreds, err := s.encryptCredentials(params.Credentials)
		if err != nil {
			slog.ErrorContext(ctx, "failed to encrypt credential profile credentials", "user_id", userID, "profile_id", profileID, "error", err)
			return nil, fmt.Errorf("encrypt credentials: %w", err)
		}
		updates["credentials_encrypted"] = encryptedCreds
	}

	if len(updates) > 0 {
		if err := s.repo.Update(ctx, profile, updates); err != nil {
			slog.ErrorContext(ctx, "failed to update credential profile", "user_id", userID, "profile_id", profileID, "error", err)
			return nil, err
		}
	}

	slog.InfoContext(ctx, "credential profile updated", "user_id", userID, "profile_id", profileID)
	return s.GetCredentialProfile(ctx, userID, profileID)
}

func (s *CredentialProfileService) SetDefaultCredentialProfile(ctx context.Context, userID, profileID int64) (*agent.UserAgentCredentialProfile, error) {
	profile, err := s.GetCredentialProfile(ctx, userID, profileID)
	if err != nil {
		return nil, err
	}

	_ = s.repo.UnsetDefaults(ctx, userID, profile.AgentSlug)

	if err := s.repo.SetDefault(ctx, profile); err != nil {
		slog.ErrorContext(ctx, "failed to set default credential profile", "user_id", userID, "profile_id", profileID, "error", err)
		return nil, err
	}

	slog.InfoContext(ctx, "credential profile set as default", "user_id", userID, "profile_id", profileID, "agent_slug", profile.AgentSlug)
	return s.GetCredentialProfile(ctx, userID, profileID)
}
