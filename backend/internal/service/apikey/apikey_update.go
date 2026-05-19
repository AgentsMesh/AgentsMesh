package apikey

import (
	"errors"
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	apikeyDomain "github.com/anthropics/agentsmesh/backend/internal/domain/apikey"
)

func (s *Service) UpdateAPIKey(ctx context.Context, id int64, orgID int64, req *UpdateAPIKeyRequest) (*apikeyDomain.APIKey, error) {
	key, err := s.repo.GetByID(ctx, id, orgID)
	if err != nil {
		if errors.Is(err, apikeyDomain.ErrNotFound) {
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
		exists, err := s.repo.CheckDuplicateName(ctx, key.OrganizationID, *req.Name, &id)
		if err != nil {
			return nil, fmt.Errorf("failed to check duplicate name: %w", err)
		}
		if exists {
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

	if err := s.repo.Update(ctx, key, updates); err != nil {
		slog.ErrorContext(ctx, "failed to update API key", "api_key_id", id, "org_id", orgID, "error", err)
		return nil, fmt.Errorf("failed to update api key: %w", err)
	}

	slog.InfoContext(ctx, "API key updated", "api_key_id", id, "org_id", orgID)
	s.invalidateCache(ctx, key.KeyHash)

	key, err = s.repo.GetByID(ctx, id, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to reload api key: %w", err)
	}

	return key, nil
}

func (s *Service) RevokeAPIKey(ctx context.Context, id int64, orgID int64) error {
	key, err := s.repo.GetByID(ctx, id, orgID)
	if err != nil {
		if errors.Is(err, apikeyDomain.ErrNotFound) {
			return ErrAPIKeyNotFound
		}
		return fmt.Errorf("failed to get api key: %w", err)
	}

	if err := s.repo.Update(ctx, key, map[string]interface{}{
		"is_enabled": false,
		"updated_at": time.Now(),
	}); err != nil {
		slog.ErrorContext(ctx, "failed to revoke API key", "api_key_id", id, "org_id", orgID, "error", err)
		return fmt.Errorf("failed to revoke api key: %w", err)
	}

	slog.InfoContext(ctx, "API key revoked", "api_key_id", id, "org_id", orgID)
	s.invalidateCache(ctx, key.KeyHash)

	return nil
}

func (s *Service) DeleteAPIKey(ctx context.Context, id int64, orgID int64) error {
	key, err := s.repo.GetByID(ctx, id, orgID)
	if err != nil {
		if errors.Is(err, apikeyDomain.ErrNotFound) {
			return ErrAPIKeyNotFound
		}
		return fmt.Errorf("failed to get api key: %w", err)
	}

	if err := s.repo.Delete(ctx, key); err != nil {
		slog.ErrorContext(ctx, "failed to delete API key", "api_key_id", id, "org_id", orgID, "error", err)
		return fmt.Errorf("failed to delete api key: %w", err)
	}

	slog.InfoContext(ctx, "API key deleted", "api_key_id", id, "org_id", orgID)
	s.invalidateCache(ctx, key.KeyHash)

	return nil
}

func (s *Service) UpdateLastUsed(ctx context.Context, id int64) error {
	return s.repo.UpdateLastUsed(ctx, id)
}
