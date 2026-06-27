package infra

import (
	"context"
	"errors"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
	"gorm.io/gorm"
)

var _ channel.ChannelRepository = (*channelRepository)(nil)

type channelAccess struct {
	ID         int64     `gorm:"primaryKey"`
	ChannelID  int64     `gorm:"not null;index"`
	PodKey     *string   `gorm:"size:100;index"`
	UserID     *int64    `gorm:"index"`
	LastAccess time.Time `gorm:"not null;default:now()"`
}

func (channelAccess) TableName() string { return "channel_access" }

type channelPod struct {
	ID        int64     `gorm:"primaryKey"`
	ChannelID int64     `gorm:"not null;index"`
	PodKey    string    `gorm:"size:100;not null"`
	JoinedAt  time.Time `gorm:"not null;default:now()"`
}

func (channelPod) TableName() string { return "channel_pods" }

type channelRepository struct {
	db *gorm.DB
}

func NewChannelRepository(db *gorm.DB) channel.ChannelRepository {
	return &channelRepository{db: db}
}

func (r *channelRepository) GetByID(ctx context.Context, channelID int64) (*channel.Channel, error) {
	var ch channel.Channel
	if err := r.db.WithContext(ctx).First(&ch, channelID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &ch, nil
}

func (r *channelRepository) GetByOrgAndName(ctx context.Context, orgID int64, name string) (*channel.Channel, error) {
	var ch channel.Channel
	err := r.db.WithContext(ctx).
		Where("organization_id = ? AND name = ?", orgID, name).
		First(&ch).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &ch, nil
}

func (r *channelRepository) SlugExists(ctx context.Context, orgID int64, slug string) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&channel.Channel{}).
		Where("organization_id = ? AND slug = ?", orgID, slug).
		Count(&n).Error
	return n > 0, err
}

func (r *channelRepository) GetByOrgAndSlug(ctx context.Context, orgID int64, slug string) (*channel.Channel, error) {
	var ch channel.Channel
	err := r.db.WithContext(ctx).
		Where("organization_id = ? AND slug = ?", orgID, slug).
		First(&ch).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &ch, nil
}

func (r *channelRepository) Create(ctx context.Context, ch *channel.Channel) error {
	return r.db.WithContext(ctx).Create(ch).Error
}

func (r *channelRepository) ListByOrg(ctx context.Context, orgID int64, filter *channel.ChannelListFilter) ([]*channel.Channel, int64, error) {
	query := r.db.WithContext(ctx).Model(&channel.Channel{}).Where("organization_id = ?", orgID)

	if !filter.IncludeArchived {
		query = query.Where("is_archived = ?", false)
	}
	if filter.RepositoryID != nil {
		query = query.Where("repository_id = ?", *filter.RepositoryID)
	}
	if filter.TicketID != nil {
		query = query.Where("ticket_id = ?", *filter.TicketID)
	}

	var total int64
	query.Count(&total)

	var channels []*channel.Channel
	if err := query.
		Order("updated_at DESC").
		Limit(filter.Limit).
		Offset(filter.Offset).
		Find(&channels).Error; err != nil {
		return nil, 0, err
	}
	return channels, total, nil
}

func (r *channelRepository) UpdateFields(ctx context.Context, channelID int64, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&channel.Channel{}).Where("id = ?", channelID).Updates(updates).Error
}

func (r *channelRepository) SetArchived(ctx context.Context, channelID int64, archived bool) error {
	return r.db.WithContext(ctx).Model(&channel.Channel{}).
		Where("id = ?", channelID).
		Update("is_archived", archived).Error
}

func (r *channelRepository) GetByTicketID(ctx context.Context, ticketID int64) ([]*channel.Channel, error) {
	var channels []*channel.Channel
	if err := r.db.WithContext(ctx).Where("ticket_id = ?", ticketID).Find(&channels).Error; err != nil {
		return nil, err
	}
	return channels, nil
}

func (r *channelRepository) ListVisibleForUser(ctx context.Context, orgID, userID int64, filter *channel.ChannelListFilter) ([]*channel.Channel, int64, error) {
	type row struct {
		channel.Channel
		IsMember    bool  `gorm:"column:is_member"`
		MemberCount int64 `gorm:"column:member_count"`
		IsMuted     bool  `gorm:"column:is_muted"`
		IsPinned    bool  `gorm:"column:is_pinned"`
	}

	baseWhere := "c.organization_id = ?"
	args := []interface{}{orgID}

	if !filter.IncludeArchived {
		baseWhere += " AND c.is_archived = FALSE"
	}
	if filter.RepositoryID != nil {
		baseWhere += " AND c.repository_id = ?"
		args = append(args, *filter.RepositoryID)
	}
	if filter.TicketID != nil {
		baseWhere += " AND c.ticket_id = ?"
		args = append(args, *filter.TicketID)
	}
	if filter.Visibility != nil {
		baseWhere += " AND c.visibility = ?"
		args = append(args, *filter.Visibility)
	}

	visibilityClause := " AND (c.visibility = 'public' OR EXISTS (SELECT 1 FROM channel_members cm3 WHERE cm3.channel_id = c.id AND cm3.user_id = ?))"
	args = append(args, userID)

	fullWhere := baseWhere + visibilityClause

	var total int64
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	r.db.WithContext(ctx).Raw("SELECT COUNT(*) FROM channels c WHERE "+fullWhere, countArgs...).Scan(&total)

	// A single LEFT JOIN on the caller's membership row supplies is_member /
	// is_muted / is_pinned at once — avoids one correlated subquery per per-user
	// flag (member_count stays a subquery; it counts all members, not the row).
	selectSQL := `SELECT c.*,
		cm_u.user_id IS NOT NULL as is_member,
		(SELECT COUNT(*) FROM channel_members cm2 WHERE cm2.channel_id = c.id) as member_count,
		COALESCE(cm_u.is_muted, FALSE) as is_muted,
		COALESCE(cm_u.is_pinned, FALSE) as is_pinned
		FROM channels c
		LEFT JOIN channel_members cm_u ON cm_u.channel_id = c.id AND cm_u.user_id = ?
		WHERE ` + fullWhere + " ORDER BY c.updated_at DESC"

	selectArgs := append([]interface{}{userID}, args...)
	if filter.Limit > 0 {
		selectSQL += " LIMIT ? OFFSET ?"
		selectArgs = append(selectArgs, filter.Limit, filter.Offset)
	}

	var rows []row
	if err := r.db.WithContext(ctx).Raw(selectSQL, selectArgs...).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}

	channels := make([]*channel.Channel, len(rows))
	for i := range rows {
		ch := rows[i].Channel
		ch.IsMember = rows[i].IsMember
		ch.MemberCount = rows[i].MemberCount
		ch.IsMuted = rows[i].IsMuted
		ch.IsPinned = rows[i].IsPinned
		channels[i] = &ch
	}
	return channels, total, nil
}
