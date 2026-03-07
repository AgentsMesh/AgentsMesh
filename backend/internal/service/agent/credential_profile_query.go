package agent

import (
	"context"
	"errors"
	"fmt"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
)

// ListCredentialProfiles returns all credential profiles for a user, grouped by agent type
func (s *CredentialProfileService) ListCredentialProfiles(ctx context.Context, userID int64) ([]*agent.CredentialProfilesByAgentType, error) {
	profiles, err := s.repo.ListActiveWithAgentType(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Group by agent type
	groupedMap := make(map[int64]*agent.CredentialProfilesByAgentType)
	for _, p := range profiles {
		group, exists := groupedMap[p.AgentTypeID]
		if !exists {
			group = &agent.CredentialProfilesByAgentType{
				AgentTypeID: p.AgentTypeID,
				Profiles:    make([]*agent.CredentialProfileResponse, 0),
			}
			if p.AgentType != nil {
				group.AgentTypeName = p.AgentType.Name
				group.AgentTypeSlug = p.AgentType.Slug
			}
			groupedMap[p.AgentTypeID] = group
		}
		group.Profiles = append(group.Profiles, s.ProfileToResponse(p))
	}

	// Convert map to slice
	result := make([]*agent.CredentialProfilesByAgentType, 0, len(groupedMap))
	for _, group := range groupedMap {
		result = append(result, group)
	}

	return result, nil
}

// ListCredentialProfilesForAgentType returns all credential profiles for a specific agent type
func (s *CredentialProfileService) ListCredentialProfilesForAgentType(ctx context.Context, userID, agentTypeID int64) ([]*agent.UserAgentCredentialProfile, error) {
	return s.repo.ListByAgentType(ctx, userID, agentTypeID)
}

// GetDefaultCredentialProfile returns the default credential profile for a user and agent type
func (s *CredentialProfileService) GetDefaultCredentialProfile(ctx context.Context, userID, agentTypeID int64) (*agent.UserAgentCredentialProfile, error) {
	profile, err := s.repo.GetDefault(ctx, userID, agentTypeID)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, ErrCredentialProfileNotFound
	}
	return profile, nil
}

// GetEffectiveCredentialsForPod returns the credentials to be injected for a pod.
// profileID semantics:
//   - nil (field absent): use user's default profile, fallback to RunnerHost if no default
//   - 0: explicit RunnerHost mode (use Runner's local environment, no credentials injected)
//   - >0: use specified credential profile ID
func (s *CredentialProfileService) GetEffectiveCredentialsForPod(ctx context.Context, userID, agentTypeID int64, profileID *int64) (agent.EncryptedCredentials, bool, error) {
	// 1. Explicit RunnerHost (profileID == 0)
	if profileID != nil && *profileID == 0 {
		return nil, true, nil
	}

	// 2. Specified profile (profileID > 0)
	if profileID != nil && *profileID > 0 {
		profile, err := s.GetCredentialProfile(ctx, userID, *profileID)
		if err != nil {
			return nil, false, err
		}
		if profile.IsRunnerHost {
			return nil, true, nil
		}
		decrypted, err := s.decryptCredentials(profile.CredentialsEncrypted)
		if err != nil {
			return nil, false, fmt.Errorf("decrypt credentials: %w", err)
		}
		return decrypted, false, nil
	}

	// 3. Not specified (profileID == nil) → use default profile, fallback to RunnerHost
	profile, err := s.GetDefaultCredentialProfile(ctx, userID, agentTypeID)
	if err != nil {
		if errors.Is(err, ErrCredentialProfileNotFound) {
			return nil, true, nil
		}
		return nil, false, err
	}
	if profile.IsRunnerHost {
		return nil, true, nil
	}
	decrypted, err := s.decryptCredentials(profile.CredentialsEncrypted)
	if err != nil {
		return nil, false, fmt.Errorf("decrypt credentials: %w", err)
	}
	return decrypted, false, nil
}
