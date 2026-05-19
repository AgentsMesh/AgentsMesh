package channel

import (
	"context"
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
)

func (s *Service) TrackAccess(ctx context.Context, channelID int64, podKey *string, userID *int64) error {
	if err := s.repo.UpsertAccess(ctx, channelID, podKey, userID); err != nil {
		slog.ErrorContext(ctx, "failed to track channel access", "channel_id", channelID, "pod_key", podKey, "user_id", userID, "error", err)
		return err
	}
	return nil
}

func (s *Service) GetChannelsForPod(ctx context.Context, podKey string) ([]*channel.Channel, error) {
	return s.repo.GetChannelsForPod(ctx, podKey)
}

func (s *Service) HasAccessed(ctx context.Context, channelID int64, podKey string) (bool, error) {
	return s.repo.HasAccessed(ctx, channelID, podKey)
}

func (s *Service) GetAccessCount(ctx context.Context, channelID int64) (int64, error) {
	return s.repo.GetAccessCount(ctx, channelID)
}
