package infra

import (
	"context"
	"errors"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (r *channelRepository) UpsertMember(ctx context.Context, channelID, userID int64) error {
	member := channel.Member{
		ChannelID: channelID,
		UserID:    userID,
		JoinedAt:  time.Now(),
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "channel_id"}, {Name: "user_id"}},
		DoNothing: true,
	}).Create(&member).Error
}

func (r *channelRepository) AddMemberWithRole(ctx context.Context, channelID, userID int64, role string) error {
	member := channel.Member{
		ChannelID: channelID,
		UserID:    userID,
		Role:      role,
		JoinedAt:  time.Now(),
	}
	result := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "channel_id"}, {Name: "user_id"}},
		DoNothing: true,
	}).Create(&member)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		r.initReadStateToLatest(ctx, channelID, userID)
	}
	return nil
}

func (r *channelRepository) initReadStateToLatest(ctx context.Context, channelID, userID int64) {
	var maxID *int64
	if err := r.db.WithContext(ctx).
		Model(&channel.Message{}).
		Where("channel_id = ? AND is_deleted = FALSE", channelID).
		Select("MAX(id)").Scan(&maxID).Error; err != nil {
		return
	}
	if maxID != nil {
		_ = r.MarkRead(ctx, channelID, userID, *maxID)
	}
}

func (r *channelRepository) IsMember(ctx context.Context, channelID, userID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&channel.Member{}).
		Where("channel_id = ? AND user_id = ?", channelID, userID).
		Count(&count).Error
	return count > 0, err
}

func (r *channelRepository) RemoveMember(ctx context.Context, channelID, userID int64) error {
	return r.db.WithContext(ctx).
		Where("channel_id = ? AND user_id = ?", channelID, userID).
		Delete(&channel.Member{}).Error
}

func (r *channelRepository) GetMemberFlags(ctx context.Context, channelID, userID int64) (isMuted, isPinned bool, err error) {
	var m channel.Member
	err = r.db.WithContext(ctx).
		Select("is_muted", "is_pinned").
		Where("channel_id = ? AND user_id = ?", channelID, userID).
		Take(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, false, nil
	}
	return m.IsMuted, m.IsPinned, err
}

func (r *channelRepository) GetMemberRole(ctx context.Context, channelID, userID int64) (string, error) {
	var role string
	err := r.db.WithContext(ctx).
		Model(&channel.Member{}).
		Where("channel_id = ? AND user_id = ?", channelID, userID).
		Pluck("role", &role).Error
	return role, err
}

func (r *channelRepository) GetMembers(ctx context.Context, channelID int64, limit, offset int) ([]channel.Member, int64, error) {
	var members []channel.Member
	var total int64
	q := r.db.WithContext(ctx).Where("channel_id = ?", channelID)
	if err := q.Model(&channel.Member{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if limit > 0 {
		q = q.Limit(limit).Offset(offset)
	}
	if err := q.Find(&members).Error; err != nil {
		return nil, 0, err
	}
	return members, total, nil
}

func (r *channelRepository) GetMemberUserIDs(ctx context.Context, channelID int64) ([]int64, error) {
	var userIDs []int64
	err := r.db.WithContext(ctx).
		Model(&channel.Member{}).
		Where("channel_id = ?", channelID).
		Pluck("user_id", &userIDs).Error
	return userIDs, err
}

func (r *channelRepository) GetNonMutedMemberUserIDs(ctx context.Context, channelID int64) ([]int64, error) {
	var userIDs []int64
	err := r.db.WithContext(ctx).
		Model(&channel.Member{}).
		Where("channel_id = ? AND is_muted = FALSE", channelID).
		Pluck("user_id", &userIDs).Error
	return userIDs, err
}

func (r *channelRepository) SetMemberMuted(ctx context.Context, channelID, userID int64, muted bool) error {
	return r.db.WithContext(ctx).
		Model(&channel.Member{}).
		Where("channel_id = ? AND user_id = ?", channelID, userID).
		Update("is_muted", muted).Error
}

func (r *channelRepository) SetMemberPinned(ctx context.Context, channelID, userID int64, pinned bool) error {
	return r.db.WithContext(ctx).
		Model(&channel.Member{}).
		Where("channel_id = ? AND user_id = ?", channelID, userID).
		Update("is_pinned", pinned).Error
}
