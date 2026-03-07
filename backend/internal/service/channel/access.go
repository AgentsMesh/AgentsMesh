package channel

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
)

// TrackAccess records a pod or user accessing a channel
func (s *Service) TrackAccess(ctx context.Context, channelID int64, podKey *string, userID *int64) error {
	return s.repo.UpsertAccess(ctx, channelID, podKey, userID)
}

// GetChannelsForPod returns channels a pod has accessed
func (s *Service) GetChannelsForPod(ctx context.Context, podKey string) ([]*channel.Channel, error) {
	return s.repo.GetChannelsForPod(ctx, podKey)
}

// HasAccessed checks if a pod has accessed a channel
func (s *Service) HasAccessed(ctx context.Context, channelID int64, podKey string) (bool, error) {
	return s.repo.HasAccessed(ctx, channelID, podKey)
}

// GetAccessCount returns the number of unique accessors for a channel
func (s *Service) GetAccessCount(ctx context.Context, channelID int64) (int64, error) {
	return s.repo.GetAccessCount(ctx, channelID)
}
