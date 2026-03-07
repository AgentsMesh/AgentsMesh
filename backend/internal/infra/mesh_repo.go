package infra

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
	"github.com/anthropics/agentsmesh/backend/internal/domain/mesh"
	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	"gorm.io/gorm"
)

// Compile-time interface compliance check.
var _ mesh.MeshRepository = (*meshRepository)(nil)

type meshRepository struct{ db *gorm.DB }

// NewMeshRepository creates a new GORM-backed MeshRepository.
func NewMeshRepository(db *gorm.DB) mesh.MeshRepository {
	return &meshRepository{db: db}
}

func (r *meshRepository) ListEnabledRunners(ctx context.Context, orgID int64) ([]*runner.Runner, error) {
	var runners []*runner.Runner
	err := r.db.WithContext(ctx).
		Where("organization_id = ? AND is_enabled = ?", orgID, true).
		Find(&runners).Error
	return runners, err
}

func (r *meshRepository) GetChannelPodKeys(ctx context.Context, channelID int64) ([]string, error) {
	var pods []mesh.ChannelPod
	if err := r.db.WithContext(ctx).
		Where("channel_id = ?", channelID).
		Find(&pods).Error; err != nil {
		return nil, err
	}
	keys := make([]string, len(pods))
	for i, cp := range pods {
		keys[i] = cp.PodKey
	}
	return keys, nil
}

func (r *meshRepository) CountChannelMessages(ctx context.Context, channelID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&channel.Message{}).
		Where("channel_id = ?", channelID).
		Count(&count).Error
	return count, err
}

func (r *meshRepository) ListPodsByTicketIDs(ctx context.Context, ticketIDs []int64) ([]*agentpod.Pod, error) {
	var pods []*agentpod.Pod
	err := r.db.WithContext(ctx).
		Where("ticket_id IN ?", ticketIDs).
		Find(&pods).Error
	return pods, err
}

func (r *meshRepository) CreateChannelPod(ctx context.Context, cp *mesh.ChannelPod) error {
	return r.db.WithContext(ctx).Create(cp).Error
}

func (r *meshRepository) DeleteChannelPod(ctx context.Context, channelID int64, podKey string) error {
	return r.db.WithContext(ctx).
		Where("channel_id = ? AND pod_key = ?", channelID, podKey).
		Delete(&mesh.ChannelPod{}).Error
}

func (r *meshRepository) CreateChannelAccess(ctx context.Context, access *mesh.ChannelAccess) error {
	return r.db.WithContext(ctx).Create(access).Error
}
