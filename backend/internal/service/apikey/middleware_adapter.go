package apikey

import (
	"context"
	"errors"
	"fmt"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
)

var _ middleware.APIKeyValidator = (*MiddlewareAdapter)(nil)

type MiddlewareAdapter struct {
	svc *Service
}

func NewMiddlewareAdapter(svc *Service) *MiddlewareAdapter {
	return &MiddlewareAdapter{svc: svc}
}

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

func (a *MiddlewareAdapter) UpdateLastUsed(ctx context.Context, id int64) error {
	return a.svc.UpdateLastUsed(ctx, id)
}

// translateError remaps service sentinels → middleware sentinels — middleware can't
// import service (would create import cycle).
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
