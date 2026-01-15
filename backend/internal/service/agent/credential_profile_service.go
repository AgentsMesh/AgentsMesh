package agent

import (
	"context"
	"errors"

	"github.com/anthropics/agentmesh/backend/internal/domain/agent"
	"gorm.io/gorm"
)

// Errors for CredentialProfileService
var (
	ErrCredentialProfileNotFound = errors.New("credential profile not found")
	ErrCredentialProfileExists   = errors.New("credential profile with this name already exists")
	ErrCredentialsRequired       = errors.New("required credentials missing")
)

// AgentTypeProvider provides agent type lookup for credential profile operations
type AgentTypeProvider interface {
	GetAgentType(ctx context.Context, id int64) (*agent.AgentType, error)
}

// CredentialProfileService handles user credential profile operations
type CredentialProfileService struct {
	db               *gorm.DB
	agentTypeService AgentTypeProvider
}

// NewCredentialProfileService creates a new credential profile service
func NewCredentialProfileService(db *gorm.DB, agentTypeService AgentTypeProvider) *CredentialProfileService {
	return &CredentialProfileService{
		db:               db,
		agentTypeService: agentTypeService,
	}
}

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
		encryptedCreds = make(agent.EncryptedCredentials)
		for k, v := range params.Credentials {
			// TODO: Actually encrypt the value using encryption service
			encryptedCreds[k] = v
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
		encryptedCreds := make(agent.EncryptedCredentials)
		for k, v := range params.Credentials {
			// TODO: Actually encrypt the value using encryption service
			encryptedCreds[k] = v
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
		group.Profiles = append(group.Profiles, p.ToResponse())
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

	return profile.CredentialsEncrypted, false, nil
}

// SetUserCredentials sets user-level credentials for an agent (legacy method)
func (s *CredentialProfileService) SetUserCredentials(ctx context.Context, userID, agentTypeID int64, credentials agent.EncryptedCredentials) error {
	userCreds := &agent.UserAgentCredential{
		UserID:               userID,
		AgentTypeID:          agentTypeID,
		CredentialsEncrypted: credentials,
	}

	return s.db.WithContext(ctx).
		Where("user_id = ? AND agent_type_id = ?", userID, agentTypeID).
		Assign(userCreds).
		FirstOrCreate(userCreds).Error
}

// GetUserCredentials returns user-level credentials for an agent (legacy method)
func (s *CredentialProfileService) GetUserCredentials(ctx context.Context, userID, agentTypeID int64) (*agent.UserAgentCredential, error) {
	var userCreds agent.UserAgentCredential
	if err := s.db.WithContext(ctx).
		Where("user_id = ? AND agent_type_id = ?", userID, agentTypeID).
		First(&userCreds).Error; err != nil {
		return nil, err
	}
	return &userCreds, nil
}

// DeleteUserCredentials deletes user-level credentials for an agent (legacy method)
func (s *CredentialProfileService) DeleteUserCredentials(ctx context.Context, userID, agentTypeID int64) error {
	return s.db.WithContext(ctx).
		Where("user_id = ? AND agent_type_id = ?", userID, agentTypeID).
		Delete(&agent.UserAgentCredential{}).Error
}
