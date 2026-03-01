package apikey

import (
	"context"
	"errors"
	"fmt"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/middleware"
)

// Compile-time check that MiddlewareAdapter implements APIKeyValidator
var _ middleware.APIKeyValidator = (*MiddlewareAdapter)(nil)

// MiddlewareAdapter adapts Service to the middleware.APIKeyValidator interface.
// It translates service-layer errors to middleware-layer errors so that
// errors.Is() works correctly in the middleware's handleAPIKeyError().
type MiddlewareAdapter struct {
	svc *Service
}

// NewMiddlewareAdapter creates an adapter for use in middleware
func NewMiddlewareAdapter(svc *Service) *MiddlewareAdapter {
	return &MiddlewareAdapter{svc: svc}
}

// ValidateKey validates an API key and returns the result in middleware format.
// Translates service sentinel errors to middleware sentinel errors.
func (a *MiddlewareAdapter) ValidateKey(ctx context.Context, rawKey string) (*middleware.APIKeyValidateResult, error) {
	result, err := a.svc.ValidateKey(ctx, rawKey)
	if err != nil {
		return nil, translateError(err)
	}

	return &middleware.APIKeyValidateResult{
		APIKeyID:       result.APIKeyID,
		OrganizationID: result.OrganizationID,
		CreatedBy:      result.CreatedBy,
		Scopes:         result.Scopes,
		KeyName:        result.KeyName,
	}, nil
}

// UpdateLastUsed delegates to the service
func (a *MiddlewareAdapter) UpdateLastUsed(ctx context.Context, id int64) error {
	return a.svc.UpdateLastUsed(ctx, id)
}

// translateError converts service-layer sentinel errors to middleware-layer sentinel errors.
// This is necessary because the middleware package cannot import the service package
// (it would create a circular dependency), so each has its own error definitions.
func translateError(err error) error {
	switch {
	case errors.Is(err, ErrAPIKeyNotFound):
		return fmt.Errorf("%w", middleware.ErrAPIKeyNotFound)
	case errors.Is(err, ErrAPIKeyDisabled):
		return fmt.Errorf("%w", middleware.ErrAPIKeyDisabled)
	case errors.Is(err, ErrAPIKeyExpired):
		return fmt.Errorf("%w", middleware.ErrAPIKeyExpired)
	default:
		return err
	}
}
