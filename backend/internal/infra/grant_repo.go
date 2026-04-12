package infra

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/grant"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type grantRepo struct {
	db *gorm.DB
}

func NewGrantRepository(db *gorm.DB) grant.Repository {
	return &grantRepo{db: db}
}

func (r *grantRepo) Create(ctx context.Context, g *grant.ResourceGrant) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "organization_id"}, {Name: "resource_type"}, {Name: "resource_id"}, {Name: "user_id"}},
		DoNothing: true,
	}).Create(g).Error
}

func (r *grantRepo) Delete(ctx context.Context, resourceType, resourceID string, grantID int64) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND resource_type = ? AND resource_id = ?", grantID, resourceType, resourceID).
		Delete(&grant.ResourceGrant{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *grantRepo) ListByResource(ctx context.Context, resourceType, resourceID string) ([]*grant.ResourceGrant, error) {
	var grants []*grant.ResourceGrant
	err := r.db.WithContext(ctx).
		Where("resource_type = ? AND resource_id = ?", resourceType, resourceID).
		Preload("User").Preload("GrantedByUser").
		Order("created_at ASC").
		Find(&grants).Error
	return grants, err
}

func (r *grantRepo) GetGrantedUserIDs(ctx context.Context, resourceType, resourceID string) ([]int64, error) {
	var userIDs []int64
	err := r.db.WithContext(ctx).
		Model(&grant.ResourceGrant{}).
		Where("resource_type = ? AND resource_id = ?", resourceType, resourceID).
		Pluck("user_id", &userIDs).Error
	return userIDs, err
}

func (r *grantRepo) GetGrantedResourceIDs(ctx context.Context, resourceType string, userID int64, orgID int64) ([]string, error) {
	var resourceIDs []string
	err := r.db.WithContext(ctx).
		Model(&grant.ResourceGrant{}).
		Where("resource_type = ? AND user_id = ? AND organization_id = ?", resourceType, userID, orgID).
		Pluck("resource_id", &resourceIDs).Error
	return resourceIDs, err
}

func (r *grantRepo) DeleteByResource(ctx context.Context, resourceType, resourceID string) error {
	return r.db.WithContext(ctx).
		Where("resource_type = ? AND resource_id = ?", resourceType, resourceID).
		Delete(&grant.ResourceGrant{}).Error
}
