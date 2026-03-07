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
	// minExpiresIn is the minimum allowed expiration (5 minutes)
	minExpiresIn = 300
	// maxExpiresIn is the maximum allowed expiration (3 years)
	maxExpiresIn = 94608000
	// maxNameLength is the maximum allowed API key name length
	maxNameLength = 255
)

// CreateAPIKey generates a new API key for an organization
func (s *Service) CreateAPIKey(ctx context.Context, req *CreateAPIKeyRequest) (*CreateAPIKeyResponse, error) {
	// Normalize and validate name
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return nil, ErrNameEmpty
	}
	if len(req.Name) > maxNameLength {
		return nil, ErrNameTooLong
	}

	// Validate at least one scope is provided
	if len(req.Scopes) == 0 {
		return nil, ErrScopesRequired
	}

	// Validate scopes
	for _, scope := range req.Scopes {
		if !apikeyDomain.ValidateScope(scope) {
			return nil, fmt.Errorf("%w: %s", ErrInvalidScope, scope)
		}
	}

	// Validate expires_in range
	if req.ExpiresIn != nil {
		if *req.ExpiresIn < minExpiresIn || *req.ExpiresIn > maxExpiresIn {
			return nil, ErrInvalidExpiresIn
		}
	}

	// Check duplicate name within organization
	exists, err := s.repo.CheckDuplicateName(ctx, req.OrganizationID, req.Name, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to check duplicate name: %w", err)
	}
	if exists {
		return nil, ErrDuplicateKeyName
	}

	// Generate random key: "amk_" + 40 bytes hex = 84 chars total
	keyBytes := make([]byte, 40)
	if _, err := rand.Read(keyBytes); err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}
	rawKey := "amk_" + hex.EncodeToString(keyBytes)
	keyPrefix := rawKey[:12] // "amk_" + 8 hex chars

	// SHA-256 hash for storage
	hashBytes := sha256.Sum256([]byte(rawKey))
	keyHash := hex.EncodeToString(hashBytes[:])

	// Handle expiration
	var expiresAt *time.Time
	if req.ExpiresIn != nil && *req.ExpiresIn > 0 {
		t := time.Now().Add(time.Duration(*req.ExpiresIn) * time.Second)
		expiresAt = &t
	}

	// Create record
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

	if err := s.repo.Create(ctx, key); err != nil {
		return nil, fmt.Errorf("failed to create api key: %w", err)
	}

	return &CreateAPIKeyResponse{
		APIKey: key,
		RawKey: rawKey,
	}, nil
}
