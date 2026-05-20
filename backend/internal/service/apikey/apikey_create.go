package apikey

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	apikeyDomain "github.com/anthropics/agentsmesh/backend/internal/domain/apikey"
)

const (
	minExpiresIn  = 300
	maxExpiresIn  = 94608000
	maxNameLength = 255
)

func (s *Service) CreateAPIKey(ctx context.Context, req *CreateAPIKeyRequest) (*CreateAPIKeyResponse, error) {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return nil, ErrNameEmpty
	}
	if len(req.Name) > maxNameLength {
		return nil, ErrNameTooLong
	}

	if len(req.Scopes) == 0 {
		return nil, ErrScopesRequired
	}

	for _, scope := range req.Scopes {
		if !apikeyDomain.ValidateScope(scope) {
			return nil, fmt.Errorf("%w: %s", ErrInvalidScope, scope)
		}
	}

	if req.ExpiresIn != nil {
		if *req.ExpiresIn < minExpiresIn || *req.ExpiresIn > maxExpiresIn {
			return nil, ErrInvalidExpiresIn
		}
	}

	exists, err := s.repo.CheckDuplicateName(ctx, req.OrganizationID, req.Name, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to check duplicate name: %w", err)
	}
	if exists {
		return nil, ErrDuplicateKeyName
	}

	keyBytes := make([]byte, 40)
	if _, err := rand.Read(keyBytes); err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}
	rawKey := "amk_" + hex.EncodeToString(keyBytes)
	keyPrefix := rawKey[:12]

	hashBytes := sha256.Sum256([]byte(rawKey))
	keyHash := hex.EncodeToString(hashBytes[:])

	var expiresAt *time.Time
	if req.ExpiresIn != nil && *req.ExpiresIn > 0 {
		t := time.Now().Add(time.Duration(*req.ExpiresIn) * time.Second)
		expiresAt = &t
	}

	slug, err := s.EnsureUniqueSlug(ctx, req.OrganizationID, req.Name)
	if err != nil {
		// Non-fatal: leave slug NULL, backfill picks it up. Logs surface to
		// flag callers passing names that sanitize to nothing.
		slug = ""
	}

	key := &apikeyDomain.APIKey{
		OrganizationID: req.OrganizationID,
		Name:           req.Name,
		Description:    req.Description,
		KeyPrefix:      keyPrefix,
		KeyHash:        keyHash,
		Scopes:         apikeyDomain.ScopesFromStrings(req.Scopes),
		IsEnabled:      true,
		ExpiresAt:      expiresAt,
		CreatedBy:      req.CreatedBy,
	}
	if slug != "" {
		key.Slug = &slug
	}

	if err := s.repo.Create(ctx, key); err != nil {
		return nil, fmt.Errorf("failed to create api key: %w", err)
	}

	return &CreateAPIKeyResponse{
		APIKey: key,
		RawKey: rawKey,
	}, nil
}
