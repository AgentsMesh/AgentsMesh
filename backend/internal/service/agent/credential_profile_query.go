package agent

import (
	"context"
	"errors"
	"fmt"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	"gorm.io/gorm"
)

// ListCredentialProfiles returns all credential profiles for a user, grouped by agent type
func (s *CredentialProfileService) ListCredentialProfiles(ctx context.Context, userID int64) ([]*agent.CredentialProfilesByAgentType, error) {
	var profiles []*agent.UserAgentCredentialProfile
	err := s.db.WithContext(ctx).
		Preload("AgentType").
		Where("user_id = ? AND is_active = ?", userID, true).
		Order("agent_type_id, is_default DESC, name").
		Find(&profiles).Error
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
	var profiles []*agent.UserAgentCredentialProfile
	err := s.db.WithContext(ctx).
		Preload("AgentType").
		Where("user_id = ? AND agent_type_id = ? AND is_active = ?", userID, agentTypeID, true).
		Order("is_default DESC, name").
		Find(&profiles).Error
	return profiles, err
}

// GetDefaultCredentialProfile returns the default credential profile for a user and agent type
func (s *CredentialProfileService) GetDefaultCredentialProfile(ctx context.Context, userID, agentTypeID int64) (*agent.UserAgentCredentialProfile, error) {
	var profile agent.UserAgentCredentialProfile
	err := s.db.WithContext(ctx).
		Preload("AgentType").
		Where("user_id = ? AND agent_type_id = ? AND is_default = ? AND is_active = ?", userID, agentTypeID, true, true).
		First(&profile).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCredentialProfileNotFound
		}
		return nil, err
	}
	return &profile, nil
}

// GetEffectiveCredentialsForPod returns the credentials to be injected for a pod
// Returns nil if using RunnerHost mode
func (s *CredentialProfileService) GetEffectiveCredentialsForPod(ctx context.Context, userID, agentTypeID int64, profileID *int64) (agent.EncryptedCredentials, bool, error) {
	var profile *agent.UserAgentCredentialProfile
	var err error

	if profileID != nil && *profileID > 0 {
		// Use specified profile
		profile, err = s.GetCredentialProfile(ctx, userID, *profileID)
		if err != nil {
			return nil, false, err
		}
	} else {
		// Use default profile
		profile, err = s.GetDefaultCredentialProfile(ctx, userID, agentTypeID)
		if err != nil {
			if errors.Is(err, ErrCredentialProfileNotFound) {
				// No default profile, use RunnerHost mode
				return nil, true, nil
			}
			return nil, false, err
		}
	}

	if profile.IsRunnerHost {
		return nil, true, nil
	}

	// Decrypt credentials before returning to caller
	decrypted, err := s.decryptCredentials(profile.CredentialsEncrypted)
	if err != nil {
		return nil, false, fmt.Errorf("decrypt credentials: %w", err)
	}
	return decrypted, false, nil
}
