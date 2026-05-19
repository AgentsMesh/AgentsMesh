package agent

import (
	"context"
	"errors"
	"fmt"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
)

func (s *CredentialProfileService) ListCredentialProfiles(ctx context.Context, userID int64) ([]*agent.CredentialProfilesByAgent, error) {
	profiles, err := s.repo.ListActiveWithAgent(ctx, userID)
	if err != nil {
		return nil, err
	}

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

	result := make([]*agent.CredentialProfilesByAgent, 0, len(groupedMap))
	for _, group := range groupedMap {
		result = append(result, group)
	}

	return result, nil
}

func (s *CredentialProfileService) ListCredentialProfilesForAgent(ctx context.Context, userID int64, agentSlug string) ([]*agent.UserAgentCredentialProfile, error) {
	return s.repo.ListByAgentSlug(ctx, userID, agentSlug)
}

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

func (s *CredentialProfileService) GetEffectiveCredentialsForPod(ctx context.Context, userID int64, agentSlug string, profileID *int64) (agent.EncryptedCredentials, bool, error) {
	if profileID != nil && *profileID == 0 {
		return nil, true, nil
	}

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

func (s *CredentialProfileService) ResolveCredentialsByName(ctx context.Context, userID int64, agentSlug, profileName string) (agent.EncryptedCredentials, bool, error) {
	if profileName == "runner_host" {
		return nil, true, nil
	}

	profile, err := s.repo.GetByName(ctx, userID, agentSlug, profileName)
	if err != nil {
		return nil, false, fmt.Errorf("lookup credential profile %q: %w", profileName, err)
	}
	if profile == nil {
		return nil, false, fmt.Errorf("credential profile %q not found for agent %s", profileName, agentSlug)
	}
	if profile.IsRunnerHost {
		return nil, true, nil
	}
	decrypted, err := s.decryptCredentials(profile.CredentialsEncrypted)
	if err != nil {
		return nil, false, fmt.Errorf("decrypt credentials for profile %q: %w", profileName, err)
	}
	return decrypted, false, nil
}
