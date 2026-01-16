package channel

import (
	"context"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
)

// ChannelPod represents a pod joined to a channel
type ChannelPod struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	ChannelID int64     `gorm:"not null;index" json:"channel_id"`
	PodKey    string    `gorm:"size:100;not null" json:"pod_key"`
	JoinedAt  time.Time `gorm:"not null;default:now()" json:"joined_at"`
}

func (ChannelPod) TableName() string {
	return "channel_pods"
}

// JoinChannel adds a pod to a channel
func (s *Service) JoinChannel(ctx context.Context, channelID int64, podKey string) error {
	cp := &ChannelPod{
		ChannelID: channelID,
		PodKey:    podKey,
		JoinedAt:  time.Now(),
	}
	return s.db.WithContext(ctx).Create(cp).Error
}

// LeaveChannel removes a pod from a channel
func (s *Service) LeaveChannel(ctx context.Context, channelID int64, podKey string) error {
	return s.db.WithContext(ctx).
		Where("channel_id = ? AND pod_key = ?", channelID, podKey).
		Delete(&ChannelPod{}).Error
}

// GetChannelPods returns pods joined to a channel
func (s *Service) GetChannelPods(ctx context.Context, channelID int64) ([]*agentpod.Pod, error) {
	var channelPods []ChannelPod
	if err := s.db.WithContext(ctx).
		Where("channel_id = ?", channelID).
		Find(&channelPods).Error; err != nil {
		return nil, err
	}

	if len(channelPods) == 0 {
		return []*agentpod.Pod{}, nil
	}

	podKeys := make([]string, len(channelPods))
	for i, cp := range channelPods {
		podKeys[i] = cp.PodKey
	}

	var pods []*agentpod.Pod
	if err := s.db.WithContext(ctx).
		Where("pod_key IN ?", podKeys).
		Find(&pods).Error; err != nil {
		return nil, err
	}

	return pods, nil
}
