package infra

import (
	"context"
	"errors"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
	"gorm.io/gorm"
)

// --- Access tracking ---

func (r *channelRepository) UpsertAccess(ctx context.Context, channelID int64, podKey *string, userID *int64) error {
	query := r.db.WithContext(ctx).Where("channel_id = ?", channelID)
	if podKey != nil {
		query = query.Where("pod_key = ?", *podKey)
	}
	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	}

	var existing channelAccess
	if err := query.First(&existing).Error; err == nil {
		return r.db.WithContext(ctx).Model(&existing).Update("last_access", time.Now()).Error
	}

	return r.db.WithContext(ctx).Create(&channelAccess{
		ChannelID: channelID,
		PodKey:    podKey,
		UserID:    userID,
	}).Error
}

func (r *channelRepository) GetChannelsForPod(ctx context.Context, podKey string) ([]*channel.Channel, error) {
	var accesses []channelAccess
	if err := r.db.WithContext(ctx).Where("pod_key = ?", podKey).Find(&accesses).Error; err != nil {
		return nil, err
	}
	if len(accesses) == 0 {
		return []*channel.Channel{}, nil
	}

	ids := make([]int64, len(accesses))
	for i, a := range accesses {
		ids[i] = a.ChannelID
	}

	var channels []*channel.Channel
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&channels).Error; err != nil {
		return nil, err
	}
	return channels, nil
}

func (r *channelRepository) HasAccessed(ctx context.Context, channelID int64, podKey string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&channelAccess{}).
		Where("channel_id = ? AND pod_key = ?", channelID, podKey).
		Count(&count).Error
	return count > 0, err
}

func (r *channelRepository) GetAccessCount(ctx context.Context, channelID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&channelAccess{}).
		Where("channel_id = ?", channelID).
		Count(&count).Error
	return count, err
}

// --- Channel Pods ---

func (r *channelRepository) AddPodToChannel(ctx context.Context, channelID int64, podKey string) error {
	return r.db.WithContext(ctx).Create(&channelPod{
		ChannelID: channelID,
		PodKey:    podKey,
	}).Error
}

func (r *channelRepository) RemovePodFromChannel(ctx context.Context, channelID int64, podKey string) error {
	return r.db.WithContext(ctx).
		Where("channel_id = ? AND pod_key = ?", channelID, podKey).
		Delete(&channelPod{}).Error
}

func (r *channelRepository) GetChannelPods(ctx context.Context, channelID int64) ([]*agentpod.Pod, error) {
	var cps []channelPod
	if err := r.db.WithContext(ctx).Where("channel_id = ?", channelID).Find(&cps).Error; err != nil {
		return nil, err
	}
	if len(cps) == 0 {
		return []*agentpod.Pod{}, nil
	}

	keys := make([]string, len(cps))
	for i, cp := range cps {
		keys[i] = cp.PodKey
	}

	var pods []*agentpod.Pod
	if err := r.db.WithContext(ctx).Where("pod_key IN ?", keys).Find(&pods).Error; err != nil {
		return nil, err
	}
	return pods, nil
}

// --- Bindings (channel-level) ---

func (r *channelRepository) CreateBinding(ctx context.Context, binding *channel.PodBinding) error {
	return r.db.WithContext(ctx).Create(binding).Error
}

func (r *channelRepository) GetBindingByID(ctx context.Context, bindingID int64) (*channel.PodBinding, error) {
	var binding channel.PodBinding
	if err := r.db.WithContext(ctx).First(&binding, bindingID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &binding, nil
}

func (r *channelRepository) GetBindingByPods(ctx context.Context, initiator, target string) (*channel.PodBinding, error) {
	var binding channel.PodBinding
	err := r.db.WithContext(ctx).
		Where("initiator_pod = ? AND target_pod = ?", initiator, target).
		First(&binding).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &binding, nil
}

func (r *channelRepository) ListBindingsForPod(ctx context.Context, podKey string) ([]*channel.PodBinding, error) {
	var bindings []*channel.PodBinding
	if err := r.db.WithContext(ctx).
		Where("initiator_pod = ? OR target_pod = ?", podKey, podKey).
		Find(&bindings).Error; err != nil {
		return nil, err
	}
	return bindings, nil
}

func (r *channelRepository) UpdateBindingFields(ctx context.Context, bindingID int64, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&channel.PodBinding{}).
		Where("id = ?", bindingID).
		Updates(updates).Error
}
