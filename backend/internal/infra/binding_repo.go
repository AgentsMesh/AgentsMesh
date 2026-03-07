package infra

import (
	"context"
	"errors"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
	"gorm.io/gorm"
)

// Compile-time interface compliance check.
var _ channel.BindingRepository = (*bindingRepository)(nil)

type bindingRepository struct {
	db *gorm.DB
}

// NewBindingRepository creates a new GORM-backed BindingRepository.
func NewBindingRepository(db *gorm.DB) channel.BindingRepository {
	return &bindingRepository{db: db}
}

func (r *bindingRepository) GetByID(ctx context.Context, bindingID int64) (*channel.PodBinding, error) {
	var binding channel.PodBinding
	if err := r.db.WithContext(ctx).First(&binding, bindingID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &binding, nil
}

func (r *bindingRepository) GetActive(ctx context.Context, initiatorPod, targetPod string) (*channel.PodBinding, error) {
	var binding channel.PodBinding
	err := r.db.WithContext(ctx).
		Where("initiator_pod = ? AND target_pod = ? AND status = ?",
			initiatorPod, targetPod, channel.BindingStatusActive).
		First(&binding).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &binding, nil
}

func (r *bindingRepository) GetExisting(ctx context.Context, initiatorPod, targetPod string) (*channel.PodBinding, error) {
	var binding channel.PodBinding
	err := r.db.WithContext(ctx).
		Where("initiator_pod = ? AND target_pod = ? AND status IN ?",
			initiatorPod, targetPod, []string{channel.BindingStatusActive, channel.BindingStatusPending}).
		First(&binding).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &binding, nil
}

func (r *bindingRepository) ListForPod(ctx context.Context, podKey string, status *string) ([]*channel.PodBinding, error) {
	query := r.db.WithContext(ctx).
		Where("initiator_pod = ? OR target_pod = ?", podKey, podKey)
	if status != nil {
		query = query.Where("status = ?", *status)
	}
	var bindings []*channel.PodBinding
	if err := query.Order("created_at DESC").Find(&bindings).Error; err != nil {
		return nil, err
	}
	return bindings, nil
}

func (r *bindingRepository) ListPending(ctx context.Context, targetPod string) ([]*channel.PodBinding, error) {
	var bindings []*channel.PodBinding
	err := r.db.WithContext(ctx).
		Where("target_pod = ? AND status = ?", targetPod, channel.BindingStatusPending).
		Order("created_at ASC").
		Find(&bindings).Error
	if err != nil {
		return nil, err
	}
	return bindings, nil
}

func (r *bindingRepository) Create(ctx context.Context, binding *channel.PodBinding) error {
	return r.db.WithContext(ctx).Create(binding).Error
}

func (r *bindingRepository) Save(ctx context.Context, binding *channel.PodBinding) error {
	return r.db.WithContext(ctx).Save(binding).Error
}

func (r *bindingRepository) MarkExpired(ctx context.Context, now time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Model(&channel.PodBinding{}).
		Where("status = ? AND expires_at IS NOT NULL AND expires_at < ?",
			channel.BindingStatusPending, now).
		Update("status", channel.BindingStatusExpired)
	return result.RowsAffected, result.Error
}
