package channel

import (
	"context"
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
	"github.com/anthropics/agentsmesh/backend/internal/infra/eventbus"
)

func (s *Service) JoinPublicChannel(ctx context.Context, channelID, userID int64) error {
	ch, err := s.GetChannel(ctx, channelID)
	if err != nil {
		return err
	}
	if !ch.IsPublic() {
		return ErrChannelPrivate
	}
	if err := s.repo.AddMemberWithRole(ctx, channelID, userID, channel.RoleMember); err != nil {
		return err
	}
	s.publishMemberEvent(ctx, ch.OrganizationID, channelID, userID, eventbus.EventChannelMemberAdded, channel.RoleMember)
	return nil
}

func (s *Service) InviteMembers(ctx context.Context, channelID, inviterUserID int64, memberIDs []int64) error {
	if err := s.requireMembership(ctx, channelID, inviterUserID); err != nil {
		return err
	}
	ch, err := s.GetChannel(ctx, channelID)
	if err != nil {
		return err
	}
	validIDs := s.validateOrgMembers(ctx, ch.OrganizationID, memberIDs)
	for _, uid := range validIDs {
		if err := s.repo.AddMemberWithRole(ctx, channelID, uid, channel.RoleMember); err != nil {
			slog.ErrorContext(ctx, "failed to invite member", "channel_id", channelID, "user_id", uid, "error", err)
			continue
		}
		s.publishMemberEvent(ctx, ch.OrganizationID, channelID, uid, eventbus.EventChannelMemberAdded, channel.RoleMember)
	}
	return nil
}

func (s *Service) LeaveUserChannel(ctx context.Context, channelID, userID int64) error {
	role, _ := s.repo.GetMemberRole(ctx, channelID, userID)
	if role == channel.RoleCreator {
		return ErrNotCreator // creator cannot abandon the channel
	}
	ch, err := s.GetChannel(ctx, channelID)
	if err != nil {
		return err
	}
	if err := s.repo.RemoveMember(ctx, channelID, userID); err != nil {
		return err
	}
	s.publishMemberEvent(ctx, ch.OrganizationID, channelID, userID, eventbus.EventChannelMemberRemoved, "")
	return nil
}

func (s *Service) RemoveMember(ctx context.Context, channelID, removerUserID, targetUserID int64) error {
	if removerUserID != targetUserID {
		if err := s.requireCreatorRole(ctx, channelID, removerUserID); err != nil {
			return err
		}
	}
	ch, err := s.GetChannel(ctx, channelID)
	if err != nil {
		return err
	}
	if err := s.repo.RemoveMember(ctx, channelID, targetUserID); err != nil {
		return err
	}
	s.publishMemberEvent(ctx, ch.OrganizationID, channelID, targetUserID, eventbus.EventChannelMemberRemoved, "")
	return nil
}

func (s *Service) SetMemberMuted(ctx context.Context, channelID, userID int64, muted bool) error {
	if err := s.requireMembership(ctx, channelID, userID); err != nil {
		return err
	}
	if err := s.repo.SetMemberMuted(ctx, channelID, userID, muted); err != nil {
		slog.ErrorContext(ctx, "failed to set member muted", "channel_id", channelID, "user_id", userID, "muted", muted, "error", err)
		return err
	}
	slog.InfoContext(ctx, "channel member muted updated", "channel_id", channelID, "user_id", userID, "muted", muted)
	return nil
}

func (s *Service) MarkRead(ctx context.Context, channelID, userID int64, messageID int64) error {
	ch, err := s.GetChannel(ctx, channelID)
	if err != nil {
		return err
	}
	if ch.IsPublic() {
		_ = s.repo.AddMemberWithRole(ctx, channelID, userID, channel.RoleMember)
	} else {
		if err := s.requireMembership(ctx, channelID, userID); err != nil {
			return err
		}
	}
	return s.repo.MarkRead(ctx, channelID, userID, messageID)
}

func (s *Service) GetUnreadCounts(ctx context.Context, userID int64) (map[int64]int64, error) {
	return s.repo.GetUnreadCounts(ctx, userID)
}

func (s *Service) GetMemberUserIDs(ctx context.Context, channelID int64) ([]int64, error) {
	return s.repo.GetMemberUserIDs(ctx, channelID)
}

func (s *Service) GetNonMutedMemberUserIDs(ctx context.Context, channelID int64) ([]int64, error) {
	return s.repo.GetNonMutedMemberUserIDs(ctx, channelID)
}

func (s *Service) ListMembers(ctx context.Context, channelID int64, limit, offset int) ([]channel.Member, int64, error) {
	return s.repo.GetMembers(ctx, channelID, limit, offset)
}
