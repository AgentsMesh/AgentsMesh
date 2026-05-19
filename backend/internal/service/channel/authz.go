package channel

import (
	"context"
	"errors"

	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
)

var (
	ErrNotMember      = errors.New("user is not a channel member")
	ErrChannelPrivate = errors.New("channel is private")
	ErrAlreadyMember  = errors.New("user is already a channel member")
	ErrNotCreator     = errors.New("only the channel creator can perform this action")
)

func (s *Service) requireMembership(ctx context.Context, channelID, userID int64) error {
	ok, err := s.repo.IsMember(ctx, channelID, userID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotMember
	}
	return nil
}

func (s *Service) requireCreatorRole(ctx context.Context, channelID, userID int64) error {
	role, err := s.repo.GetMemberRole(ctx, channelID, userID)
	if err != nil {
		return ErrNotMember
	}
	if role != channel.RoleCreator {
		return ErrNotCreator
	}
	return nil
}

func (s *Service) IsMember(ctx context.Context, channelID, userID int64) (bool, error) {
	return s.repo.IsMember(ctx, channelID, userID)
}

func (s *Service) validateOrgMembers(ctx context.Context, orgID int64, userIDs []int64) []int64 {
	if len(userIDs) == 0 || s.userLookup == nil {
		return userIDs
	}
	valid, err := s.userLookup.ValidateUserIDs(ctx, orgID, userIDs)
	if err != nil {
		return nil
	}
	return valid
}
