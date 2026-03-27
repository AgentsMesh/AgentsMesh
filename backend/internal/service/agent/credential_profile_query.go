package agent

import (
	"context"
	"errors"
	"fmt"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
)

// ListCredentialProfiles returns all credential profiles for a user, grouped by agent
func (s *CredentialProfileService) ListCredentialProfiles(ctx context.Context, userID int64) ([]*agent.CredentialProfilesByAgent, error) {
	profiles, err := s.repo.ListActiveWithAgent(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Group by agent
	groupedMap := make(map[string]*agent.CredentialProfilesByAgent)
	for _, p := range profiles {
		group, exists := groupedMap[p.AgentSlug]
		if !exists {
			group = &agent.CredentialProfilesByAgent{
				AgentSlug: p.AgentSlug,
				Profiles:    make([]*agent.CredentialProfileResponse, 0),
			}
			if p.Agent != nil {
				group.AgentName = p.Agent.Name
			}
			groupedMap[p.AgentSlug] = group
		}
		group.Profiles = append(group.Profiles, s.ProfileToResponse(p))
	}

	// Convert map to slice
	result := make([]*agent.CredentialProfilesByAgent, 0, len(groupedMap))
	for _, group := range groupedMap {
		result = append(result, group)
	}

	return result, nil
}

// ListCredentialProfilesForAgent returns all credential profiles for a specific agent
func (s *CredentialProfileService) ListCredentialProfilesForAgent(ctx context.Context, userID int64, agentSlug string) ([]*agent.UserAgentCredentialProfile, error) {
	return s.repo.ListByAgentSlug(ctx, userID, agentSlug)
}

// GetDefaultCredentialProfile returns the default credential profile for a user and agent
func (s *CredentialProfileService) GetDefaultCredentialProfile(ctx context.Context, userID int64, agentSlug string) (*agent.UserAgentCredentialProfile, error) {
	profile, err := s.repo.GetDefault(ctx, userID, agentSlug)
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
func (s *CredentialProfileService) GetEffectiveCredentialsForPod(ctx context.Context, userID int64, agentSlug string, profileID *int64) (agent.EncryptedCredentials, bool, error) {
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
	profile, err := s.GetDefaultCredentialProfile(ctx, userID, agentSlug)
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
