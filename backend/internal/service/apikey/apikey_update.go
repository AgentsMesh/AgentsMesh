package apikey

import (
	"context"
	"fmt"
	"strings"
	"time"

	apikeyDomain "github.com/AgentsMesh/AgentsMesh/backend/internal/domain/apikey"
	"gorm.io/gorm"
)

// UpdateAPIKey updates an API key's metadata with organization ownership verification
func (s *Service) UpdateAPIKey(ctx context.Context, id int64, orgID int64, req *UpdateAPIKeyRequest) (*apikeyDomain.APIKey, error) {
	var key apikeyDomain.APIKey
	if err := s.db.WithContext(ctx).Where("id = ? AND organization_id = ?", id, orgID).First(&key).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrAPIKeyNotFound
		}
		return nil, fmt.Errorf("failed to get api key: %w", err)
	}

	updates := map[string]interface{}{
		"updated_at": time.Now(),
	}

	if req.Name != nil {
		trimmed := strings.TrimSpace(*req.Name)
		if trimmed == "" {
			return nil, ErrNameEmpty
		}
		if len(trimmed) > maxNameLength {
			return nil, ErrNameTooLong
		}
		req.Name = &trimmed
		// Check duplicate name within organization (excluding self)
		var count int64
		if err := s.db.WithContext(ctx).Model(&apikeyDomain.APIKey{}).
			Where("organization_id = ? AND name = ? AND id != ?", key.OrganizationID, *req.Name, id).
			Count(&count).Error; err != nil {
			return nil, fmt.Errorf("failed to check duplicate name: %w", err)
		}
		if count > 0 {
			return nil, ErrDuplicateKeyName
		}
		updates["name"] = *req.Name
	}

	if req.Description != nil {
		updates["description"] = *req.Description
	}

	if len(req.Scopes) > 0 {
		for _, scope := range req.Scopes {
			if !apikeyDomain.ValidateScope(scope) {
				return nil, fmt.Errorf("%w: %s", ErrInvalidScope, scope)
			}
		}
		updates["scopes"] = apikeyDomain.ScopesFromStrings(req.Scopes)
	}

	if req.IsEnabled != nil {
		updates["is_enabled"] = *req.IsEnabled
	}

	if err := s.db.WithContext(ctx).Model(&key).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to update api key: %w", err)
	}

	// Invalidate cache
	s.invalidateCache(ctx, key.KeyHash)

	// Reload with organization ownership
	if err := s.db.WithContext(ctx).Where("id = ? AND organization_id = ?", id, orgID).First(&key).Error; err != nil {
		return nil, fmt.Errorf("failed to reload api key: %w", err)
	}

	return &key, nil
}

// RevokeAPIKey disables an API key with organization ownership verification
func (s *Service) RevokeAPIKey(ctx context.Context, id int64, orgID int64) error {
	var key apikeyDomain.APIKey
	if err := s.db.WithContext(ctx).Where("id = ? AND organization_id = ?", id, orgID).First(&key).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrAPIKeyNotFound
		}
		return fmt.Errorf("failed to get api key: %w", err)
	}

	if err := s.db.WithContext(ctx).Model(&key).Updates(map[string]interface{}{
		"is_enabled": false,
		"updated_at": time.Now(),
	}).Error; err != nil {
		return fmt.Errorf("failed to revoke api key: %w", err)
	}

	// Invalidate cache
	s.invalidateCache(ctx, key.KeyHash)

	return nil
}

// DeleteAPIKey permanently deletes an API key with organization ownership verification
func (s *Service) DeleteAPIKey(ctx context.Context, id int64, orgID int64) error {
	var key apikeyDomain.APIKey
	if err := s.db.WithContext(ctx).Where("id = ? AND organization_id = ?", id, orgID).First(&key).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrAPIKeyNotFound
		}
		return fmt.Errorf("failed to get api key: %w", err)
	}

	if err := s.db.WithContext(ctx).Delete(&key).Error; err != nil {
		return fmt.Errorf("failed to delete api key: %w", err)
	}

	// Invalidate cache
	s.invalidateCache(ctx, key.KeyHash)

	return nil
}

// UpdateLastUsed updates the last_used_at timestamp (fire-and-forget, errors are logged)
func (s *Service) UpdateLastUsed(ctx context.Context, id int64) error {
	return s.db.WithContext(ctx).Model(&apikeyDomain.APIKey{}).
		Where("id = ?", id).
		Update("last_used_at", time.Now()).Error
}
