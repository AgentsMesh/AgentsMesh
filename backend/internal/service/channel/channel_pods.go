package channel

import (
	"context"
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	channelDomain "github.com/anthropics/agentsmesh/backend/internal/domain/channel"
)

type PodCreatorResolver interface {
	GetPodByKey(ctx context.Context, podKey string) (*agentpod.Pod, error)
}

func (s *Service) SetPodCreatorResolver(resolver PodCreatorResolver) {
	s.podCreatorResolver = resolver
}

func (s *Service) JoinChannel(ctx context.Context, channelID int64, podKey string) error {
	if err := s.repo.AddPodToChannel(ctx, channelID, podKey); err != nil {
		return err
	}
	s.ensurePodCreatorIsMember(ctx, channelID, podKey)
	return nil
}

func (s *Service) LeaveChannel(ctx context.Context, channelID int64, podKey string) error {
	return s.repo.RemovePodFromChannel(ctx, channelID, podKey)
}

func (s *Service) GetChannelPods(ctx context.Context, channelID int64) ([]*agentpod.Pod, error) {
	return s.repo.GetChannelPods(ctx, channelID)
}

func (s *Service) ensurePodCreatorIsMember(ctx context.Context, channelID int64, podKey string) {
	if s.podCreatorResolver == nil {
		return
	}
	pod, err := s.podCreatorResolver.GetPodByKey(ctx, podKey)
	if err != nil || pod == nil {
		slog.WarnContext(ctx, "failed to resolve pod creator for channel membership", "pod_key", podKey, "error", err)
		return
	}
	_ = s.repo.AddMemberWithRole(ctx, channelID, pod.CreatedByID, channelDomain.RoleMember)
}
