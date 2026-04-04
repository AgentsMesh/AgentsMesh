package infra

import (
	"context"
	"errors"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
	"gorm.io/gorm"
)

// Compile-time interface compliance check.
var _ channel.ChannelRepository = (*channelRepository)(nil)

// channelAccess is the GORM model for the channel_access table.
type channelAccess struct {
	ID         int64     `gorm:"primaryKey"`
	ChannelID  int64     `gorm:"not null;index"`
	PodKey     *string   `gorm:"size:100;index"`
	UserID     *int64    `gorm:"index"`
	LastAccess time.Time `gorm:"not null;default:now()"`
}

func (channelAccess) TableName() string { return "channel_access" }

// channelPod is the GORM model for the channel_pods table.
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

// NewChannelRepository creates a new GORM-backed ChannelRepository.
func NewChannelRepository(db *gorm.DB) channel.ChannelRepository {
	return &channelRepository{db: db}
}

// --- Channel CRUD ---

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

// --- Messages ---

func (r *channelRepository) CreateMessage(ctx context.Context, msg *channel.Message) error {
	return r.db.WithContext(ctx).Create(msg).Error
}

func (r *channelRepository) TouchChannel(ctx context.Context, channelID int64) error {
	return r.db.WithContext(ctx).Model(&channel.Channel{}).
		Where("id = ?", channelID).
		Update("updated_at", time.Now()).Error
}

func (r *channelRepository) GetMessages(ctx context.Context, channelID int64, before *time.Time, after *time.Time, limit int) ([]*channel.Message, error) {
	query := r.db.WithContext(ctx).Where("channel_id = ? AND is_deleted = FALSE", channelID)
	if before != nil {
		query = query.Where("created_at < ?", *before)
	}
	if after != nil {
		query = query.Where("created_at > ?", *after)
	}
	// After-only: return oldest-first so hasMore means "newer messages exist beyond the limit".
	// All other cases: newest-first so hasMore means "older messages exist" (load-more / scroll-up).
	// Use "id" as tiebreaker to guarantee stable ordering when created_at has identical values
	// (e.g., SQLite CURRENT_TIMESTAMP has only second-level precision).
	order := "created_at DESC, id DESC"
	if after != nil && before == nil {
		order = "created_at ASC, id ASC"
	}
	var messages []*channel.Message
	if err := query.
		Preload("SenderUser").
		Preload("SenderPodInfo").
		Preload("SenderPodInfo.Agent").
		Order(order).
		Limit(limit).
		Find(&messages).Error; err != nil {
		return nil, err
	}
	return messages, nil
}

func (r *channelRepository) GetMessagesMentioning(ctx context.Context, channelID int64, podKey string, limit int) ([]*channel.Message, bool, error) {
	var messages []*channel.Message
	// Require BOTH "mentioned_pods" key AND exact pod key value in metadata.
	// Using two LIKE conditions scopes the match to the mentioned_pods field, avoiding false
	// positives from pod keys appearing elsewhere in the JSON (e.g. reply_to, sender context).
	// Works cross-database (PostgreSQL JSONB cast to text and SQLite text no-op).
	// Text LIKE fallback: matches legacy "@podKey" mentions in content.
	podValuePattern := `%"` + podKey + `"%`
	textPattern := "%@" + podKey + "%"
	if err := r.db.WithContext(ctx).
		Where(`channel_id = ? AND is_deleted = FALSE AND ((CAST(metadata AS TEXT) LIKE '%mentioned_pods%' AND CAST(metadata AS TEXT) LIKE ?) OR content LIKE ?)`,
			channelID, podValuePattern, textPattern).
		Order("created_at DESC, id DESC").
		Limit(limit + 1).
		Find(&messages).Error; err != nil {
		return nil, false, err
	}
	hasMore := len(messages) > limit
	if hasMore {
		messages = messages[:limit]
	}
	return messages, hasMore, nil
}

func (r *channelRepository) GetRecentMessages(ctx context.Context, channelID int64, limit int) ([]*channel.Message, error) {
	var messages []*channel.Message
	if err := r.db.WithContext(ctx).
		Where("channel_id = ? AND is_deleted = FALSE", channelID).
		Order("created_at DESC, id DESC").
		Limit(limit).
		Find(&messages).Error; err != nil {
		return nil, err
	}
	return messages, nil
}
