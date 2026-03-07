package infra

import (
	"context"
	"errors"
	"strings"

	"github.com/anthropics/agentsmesh/backend/internal/domain/promocode"
	"gorm.io/gorm"
)

// promocodeRepo implements promocode.Repository using GORM
type promocodeRepo struct {
	db *gorm.DB
}

// NewPromocodeRepository creates a new promo code repository
func NewPromocodeRepository(db *gorm.DB) promocode.Repository {
	return &promocodeRepo{db: db}
}

func (r *promocodeRepo) Create(ctx context.Context, code *promocode.PromoCode) error {
	return r.db.WithContext(ctx).Create(code).Error
}

func (r *promocodeRepo) GetByID(ctx context.Context, id int64) (*promocode.PromoCode, error) {
	var code promocode.PromoCode
	if err := r.db.WithContext(ctx).First(&code, id).Error; err != nil {
		return nil, err
	}
	return &code, nil
}

func (r *promocodeRepo) GetByCode(ctx context.Context, code string) (*promocode.PromoCode, error) {
	var promoCode promocode.PromoCode
	if err := r.db.WithContext(ctx).Where("code = ?", strings.ToUpper(code)).First(&promoCode).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &promoCode, nil
}

func (r *promocodeRepo) List(ctx context.Context, filter *promocode.ListFilter) ([]*promocode.PromoCode, int64, error) {
	query := r.db.WithContext(ctx).Model(&promocode.PromoCode{})

	if filter.Type != nil {
		query = query.Where("type = ?", *filter.Type)
	}
	if filter.PlanName != nil {
		query = query.Where("plan_name = ?", *filter.PlanName)
	}
	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}
	if filter.Search != nil && *filter.Search != "" {
		search := "%" + *filter.Search + "%"
		query = query.Where("code ILIKE ? OR name ILIKE ?", search, search)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	offset := (filter.Page - 1) * filter.PageSize

	var codes []*promocode.PromoCode
	if err := query.Order("created_at DESC").Offset(offset).Limit(filter.PageSize).Find(&codes).Error; err != nil {
		return nil, 0, err
	}

	return codes, total, nil
}

func (r *promocodeRepo) Update(ctx context.Context, code *promocode.PromoCode) error {
	return r.db.WithContext(ctx).Save(code).Error
}

func (r *promocodeRepo) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&promocode.PromoCode{}, id).Error
}

func (r *promocodeRepo) IncrementUsedCount(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Model(&promocode.PromoCode{}).
		Where("id = ?", id).
		Update("used_count", gorm.Expr("used_count + 1")).Error
}

func (r *promocodeRepo) CreateRedemption(ctx context.Context, redemption *promocode.Redemption) error {
	return r.db.WithContext(ctx).Create(redemption).Error
}

func (r *promocodeRepo) GetRedemptionsByOrg(ctx context.Context, orgID int64) ([]*promocode.Redemption, error) {
	var redemptions []*promocode.Redemption
	if err := r.db.WithContext(ctx).
		Where("organization_id = ?", orgID).
		Preload("PromoCode").
		Order("created_at DESC").
		Find(&redemptions).Error; err != nil {
		return nil, err
	}
	return redemptions, nil
}

func (r *promocodeRepo) CountOrgRedemptionsForCode(ctx context.Context, orgID int64, codeID int64) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&promocode.Redemption{}).
		Where("organization_id = ? AND promo_code_id = ?", orgID, codeID).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// RedeemAtomic atomically creates a redemption, increments used count, and applies billing.
func (r *promocodeRepo) RedeemAtomic(ctx context.Context, params *promocode.RedeemAtomicParams) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Apply billing changes within this transaction
		if params.ApplyBilling != nil {
			if err := params.ApplyBilling(ctx, tx); err != nil {
				return err
			}
		}

		// Create redemption record
		if err := tx.Create(params.Redemption).Error; err != nil {
			return err
		}

		// Increment used count
		if err := tx.Model(&promocode.PromoCode{}).
			Where("id = ?", params.PromoCodeID).
			Update("used_count", gorm.Expr("used_count + 1")).Error; err != nil {
			return err
		}

		return nil
	})
}

// Compile-time interface compliance check
var _ promocode.Repository = (*promocodeRepo)(nil)
