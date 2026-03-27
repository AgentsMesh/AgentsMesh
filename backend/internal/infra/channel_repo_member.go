package infra

import (
	"context"
	"encoding/json"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// --- Members ---

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

// --- Read States ---

func (r *channelRepository) MarkRead(ctx context.Context, channelID, userID int64, messageID int64) error {
	state := channel.ReadState{
		ChannelID:         channelID,
		UserID:            userID,
		LastReadMessageID: &messageID,
		LastReadAt:        time.Now(),
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "channel_id"}, {Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"last_read_message_id", "last_read_at"}),
	}).Create(&state).Error
}

func (r *channelRepository) GetUnreadCounts(ctx context.Context, userID int64) (map[int64]int64, error) {
	type result struct {
		ChannelID int64 `gorm:"column:channel_id"`
		Count     int64 `gorm:"column:count"`
	}

	var results []result
	// For each channel the user is a member of, count messages with ID > last_read_message_id
	err := r.db.WithContext(ctx).Raw(`
		SELECT cm.channel_id, COUNT(msg.id) as count
		FROM channel_members cm
		LEFT JOIN channel_read_states crs
			ON crs.channel_id = cm.channel_id AND crs.user_id = cm.user_id
		JOIN channel_messages msg
			ON msg.channel_id = cm.channel_id
			AND msg.is_deleted = FALSE
			AND (crs.last_read_message_id IS NULL OR msg.id > crs.last_read_message_id)
		WHERE cm.user_id = ?
		GROUP BY cm.channel_id
		HAVING COUNT(msg.id) > 0
	`, userID).Scan(&results).Error

	if err != nil {
		return nil, err
	}

	counts := make(map[int64]int64, len(results))
	for _, r := range results {
		counts[r.ChannelID] = r.Count
	}
	return counts, nil
}

// --- Message Operations ---

func (r *channelRepository) GetMessageByID(ctx context.Context, messageID int64) (*channel.Message, error) {
	var msg channel.Message
	if err := r.db.WithContext(ctx).First(&msg, messageID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &msg, nil
}

func (r *channelRepository) UpdateMessageContent(ctx context.Context, messageID int64, content string) error {
	return r.db.WithContext(ctx).
		Model(&channel.Message{}).
		Where("id = ?", messageID).
		Updates(map[string]interface{}{
			"content":   content,
			"edited_at": time.Now(),
		}).Error
}

func (r *channelRepository) SoftDeleteMessage(ctx context.Context, messageID int64) error {
	return r.db.WithContext(ctx).
		Model(&channel.Message{}).
		Where("id = ?", messageID).
		Updates(map[string]interface{}{
			"is_deleted": true,
			"content":    "[deleted]",
		}).Error
}

// --- Message Metadata ---

func (r *channelRepository) UpdateMessageMetadata(ctx context.Context, messageID int64, metadata map[string]interface{}) error {
	jsonData, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	// Use PostgreSQL jsonb merge: COALESCE(metadata, '{}') || new_data
	return r.db.WithContext(ctx).Exec(
		`UPDATE channel_messages SET metadata = COALESCE(metadata, '{}'::jsonb) || ?::jsonb WHERE id = ?`,
		string(jsonData), messageID,
	).Error
}

// --- Cursor Pagination ---

func (r *channelRepository) GetMessagesBefore(ctx context.Context, channelID int64, beforeID int64, limit int) ([]*channel.Message, error) {
	var messages []*channel.Message
	query := r.db.WithContext(ctx).
		Where("channel_id = ? AND id < ? AND is_deleted = FALSE", channelID, beforeID).
		Preload("SenderUser").
		Preload("SenderPodInfo").
		Preload("SenderPodInfo.Agent").
		Order("id DESC").
		Limit(limit)

	if err := query.Find(&messages).Error; err != nil {
		return nil, err
	}
	return messages, nil
}

// --- User Lookup for MentionValidatorHook ---

// channelUserLookup implements channel.UserLookup using GORM.
// Searches for usernames within an organization's member list.
type channelUserLookup struct {
	db *gorm.DB
}

// NewChannelUserLookup creates a UserLookup implementation.
func NewChannelUserLookup(db *gorm.DB) *channelUserLookup {
	return &channelUserLookup{db: db}
}

func (l *channelUserLookup) GetUsersByUsernames(ctx context.Context, orgID int64, usernames []string) (map[string]int64, error) {
	if len(usernames) == 0 {
		return nil, nil
	}

	type row struct {
		Username string `gorm:"column:username"`
		UserID   int64  `gorm:"column:id"`
	}

	var rows []row
	err := l.db.WithContext(ctx).Raw(`
		SELECT u.username, u.id
		FROM users u
		JOIN organization_members om ON om.user_id = u.id
		WHERE om.organization_id = ? AND u.username IN ?
	`, orgID, usernames).Scan(&rows).Error

	if err != nil {
		return nil, err
	}

	result := make(map[string]int64, len(rows))
	for _, r := range rows {
		result[r.Username] = r.UserID
	}
	return result, nil
}

func (l *channelUserLookup) ValidateUserIDs(ctx context.Context, orgID int64, userIDs []int64) ([]int64, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}

	var validIDs []int64
	err := l.db.WithContext(ctx).Raw(`
		SELECT u.id
		FROM users u
		JOIN organization_members om ON om.user_id = u.id
		WHERE om.organization_id = ? AND u.id IN ?
	`, orgID, userIDs).Pluck("id", &validIDs).Error

	if err != nil {
		return nil, err
	}
	return validIDs, nil
}

// --- Pod Lookup for MentionValidatorHook ---

// channelPodLookup implements channel.PodLookup using GORM.
type channelPodLookup struct {
	db *gorm.DB
}

// NewChannelPodLookup creates a PodLookup implementation.
func NewChannelPodLookup(db *gorm.DB) *channelPodLookup {
	return &channelPodLookup{db: db}
}

func (l *channelPodLookup) GetPodsByKeys(ctx context.Context, orgID int64, podKeys []string) ([]string, error) {
	if len(podKeys) == 0 {
		return nil, nil
	}

	var validKeys []string
	err := l.db.WithContext(ctx).
		Table("pods").
		Where("organization_id = ? AND pod_key IN ?", orgID, podKeys).
		Pluck("pod_key", &validKeys).Error

	if err != nil {
		return nil, err
	}
	return validKeys, nil
}
