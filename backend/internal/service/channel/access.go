package channel

import (
	"context"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
)

// ChannelAccess tracks pod access to a channel
type ChannelAccess struct {
	ID         int64     `gorm:"primaryKey" json:"id"`
	ChannelID  int64     `gorm:"not null;index" json:"channel_id"`
	PodKey     *string   `gorm:"size:100;index" json:"pod_key,omitempty"`
	UserID     *int64    `gorm:"index" json:"user_id,omitempty"`
	LastAccess time.Time `gorm:"not null;default:now()" json:"last_access"`
}

func (ChannelAccess) TableName() string {
	return "channel_access"
}

// TrackAccess records a pod or user accessing a channel
func (s *Service) TrackAccess(ctx context.Context, channelID int64, podKey *string, userID *int64) error {
	access := &ChannelAccess{
		ChannelID:  channelID,
		PodKey:     podKey,
		UserID:     userID,
		LastAccess: time.Now(),
	}

	query := s.db.WithContext(ctx).Where("channel_id = ?", channelID)
	if podKey != nil {
		query = query.Where("pod_key = ?", *podKey)
	}
	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	}

	var existing ChannelAccess
	if err := query.First(&existing).Error; err == nil {
		return s.db.WithContext(ctx).Model(&existing).Update("last_access", time.Now()).Error
	}

	return s.db.WithContext(ctx).Create(access).Error
}

// GetChannelsForPod returns channels a pod has accessed
func (s *Service) GetChannelsForPod(ctx context.Context, podKey string) ([]*channel.Channel, error) {
	var accesses []ChannelAccess
	if err := s.db.WithContext(ctx).
		Where("pod_key = ?", podKey).
		Find(&accesses).Error; err != nil {
		return nil, err
	}

	if len(accesses) == 0 {
		return []*channel.Channel{}, nil
	}

	channelIDs := make([]int64, len(accesses))
	for i, a := range accesses {
		channelIDs[i] = a.ChannelID
	}

	var channels []*channel.Channel
	if err := s.db.WithContext(ctx).
		Where("id IN ?", channelIDs).
		Find(&channels).Error; err != nil {
		return nil, err
	}

	return channels, nil
}

// HasAccessed checks if a pod has accessed a channel
func (s *Service) HasAccessed(ctx context.Context, channelID int64, podKey string) (bool, error) {
	var count int64
	err := s.db.WithContext(ctx).Model(&ChannelAccess{}).
		Where("channel_id = ? AND pod_key = ?", channelID, podKey).
		Count(&count).Error
	return count > 0, err
}

// GetAccessCount returns the number of unique accessors for a channel
func (s *Service) GetAccessCount(ctx context.Context, channelID int64) (int64, error) {
	var count int64
	err := s.db.WithContext(ctx).Model(&ChannelAccess{}).
		Where("channel_id = ?", channelID).
		Count(&count).Error
	return count, err
}
