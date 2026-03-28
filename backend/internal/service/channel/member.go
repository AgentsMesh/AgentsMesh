package channel

import (
	"context"
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
)

// EnsureMember ensures a user is a member of a channel (auto-join on interaction)
func (s *Service) EnsureMember(ctx context.Context, channelID, userID int64) error {
	return s.repo.UpsertMember(ctx, channelID, userID)
}

// SetMemberMuted sets the muted flag for a channel member
func (s *Service) SetMemberMuted(ctx context.Context, channelID, userID int64, muted bool) error {
	// Ensure membership first
	if err := s.repo.UpsertMember(ctx, channelID, userID); err != nil {
		return err
	}
	if err := s.repo.SetMemberMuted(ctx, channelID, userID, muted); err != nil {
		slog.Error("failed to set member muted", "channel_id", channelID, "user_id", userID, "muted", muted, "error", err)
		return err
	}
	slog.Info("channel member muted updated", "channel_id", channelID, "user_id", userID, "muted", muted)
	return nil
}

// MarkRead marks a channel as read up to a specific message ID
func (s *Service) MarkRead(ctx context.Context, channelID, userID int64, messageID int64) error {
	// Auto-join as member when marking read
	if err := s.repo.UpsertMember(ctx, channelID, userID); err != nil {
		return err
	}
	return s.repo.MarkRead(ctx, channelID, userID, messageID)
}

// GetUnreadCounts returns unread message counts for all channels the user is a member of
func (s *Service) GetUnreadCounts(ctx context.Context, userID int64) (map[int64]int64, error) {
	return s.repo.GetUnreadCounts(ctx, userID)
}

// GetMemberUserIDs returns user IDs of all members of a channel
func (s *Service) GetMemberUserIDs(ctx context.Context, channelID int64) ([]int64, error) {
	return s.repo.GetMemberUserIDs(ctx, channelID)
}

// GetNonMutedMemberUserIDs returns user IDs of members who have not muted this channel.
// Used by notification resolvers to respect channel-level mute settings.
func (s *Service) GetNonMutedMemberUserIDs(ctx context.Context, channelID int64) ([]int64, error) {
	return s.repo.GetNonMutedMemberUserIDs(ctx, channelID)
}

// ListMembers returns members of a channel with pagination
func (s *Service) ListMembers(ctx context.Context, channelID int64, limit, offset int) ([]channel.Member, int64, error) {
	return s.repo.GetMembers(ctx, channelID, limit, offset)
}
