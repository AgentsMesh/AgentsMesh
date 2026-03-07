package channel

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
)

// JoinChannel adds a pod to a channel
func (s *Service) JoinChannel(ctx context.Context, channelID int64, podKey string) error {
	return s.repo.AddPodToChannel(ctx, channelID, podKey)
}

// LeaveChannel removes a pod from a channel
func (s *Service) LeaveChannel(ctx context.Context, channelID int64, podKey string) error {
	return s.repo.RemovePodFromChannel(ctx, channelID, podKey)
}

// GetChannelPods returns pods joined to a channel
func (s *Service) GetChannelPods(ctx context.Context, channelID int64) ([]*agentpod.Pod, error) {
	return s.repo.GetChannelPods(ctx, channelID)
}
