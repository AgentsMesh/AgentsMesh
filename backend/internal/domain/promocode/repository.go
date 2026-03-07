package promocode

import (
	"context"
	"errors"
)

// Domain-level repository errors
var (
	ErrNotFound = errors.New("promo code not found")
)

// RedeemAtomicParams holds parameters for an atomic promo code redemption
type RedeemAtomicParams struct {
	Redemption  *Redemption
	PromoCodeID int64
	// ApplyBilling is called within the transaction to apply billing changes.
	// The tx parameter is an implementation-specific transaction handle.
	ApplyBilling func(ctx context.Context, tx interface{}) error
}

// Repository defines the interface for promo code data access
type Repository interface {
	// PromoCode CRUD
	Create(ctx context.Context, code *PromoCode) error
	GetByID(ctx context.Context, id int64) (*PromoCode, error)
	GetByCode(ctx context.Context, code string) (*PromoCode, error)
	List(ctx context.Context, filter *ListFilter) ([]*PromoCode, int64, error)
	Update(ctx context.Context, code *PromoCode) error
	Delete(ctx context.Context, id int64) error
	IncrementUsedCount(ctx context.Context, id int64) error

	// Redemption
	CreateRedemption(ctx context.Context, redemption *Redemption) error
	GetRedemptionsByOrg(ctx context.Context, orgID int64) ([]*Redemption, error)
	CountOrgRedemptionsForCode(ctx context.Context, orgID int64, codeID int64) (int64, error)

	// RedeemAtomic atomically creates a redemption record, increments the promo code's
	// used count, and calls ApplyBilling within the same transaction.
	RedeemAtomic(ctx context.Context, params *RedeemAtomicParams) error
}
