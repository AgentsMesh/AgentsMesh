package infra

import (
	"context"
	"encoding/json"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
	"gorm.io/gorm"
)

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
	podKeyJSON, _ := json.Marshal(podKey)
	podPattern := `%` + string(podKeyJSON) + `%`
	if err := r.db.WithContext(ctx).
		Where("channel_id = ? AND is_deleted = FALSE AND CAST(mentions AS TEXT) LIKE ?", channelID, podPattern).
		Preload("SenderUser").
		Preload("SenderPodInfo").
		Preload("SenderPodInfo.Agent").
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
		Preload("SenderUser").
		Preload("SenderPodInfo").
		Preload("SenderPodInfo.Agent").
		Order("created_at DESC, id DESC").
		Limit(limit).
		Find(&messages).Error; err != nil {
		return nil, err
	}
	return messages, nil
}

func (r *channelRepository) GetMessageByID(ctx context.Context, messageID int64) (*channel.Message, error) {
	var msg channel.Message
	if err := r.db.WithContext(ctx).
		Preload("SenderUser").
		Preload("SenderPodInfo").
		Preload("SenderPodInfo.Agent").
		First(&msg, messageID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &msg, nil
}

func (r *channelRepository) UpdateMessage(ctx context.Context, messageID int64, body string, content *channel.MessageContent, mentions channel.MessageMentions) error {
	return r.db.WithContext(ctx).
		Model(&channel.Message{}).
		Where("id = ?", messageID).
		Updates(map[string]interface{}{
			"body":      body,
			"content":   content,
			"mentions":  mentions,
			"edited_at": time.Now(),
		}).Error
}

func (r *channelRepository) UpdateMessageMentions(ctx context.Context, messageID int64, mentions channel.MessageMentions) error {
	return r.db.WithContext(ctx).
		Model(&channel.Message{}).
		Where("id = ?", messageID).
		Update("mentions", mentions).Error
}

func (r *channelRepository) SoftDeleteMessage(ctx context.Context, messageID int64) error {
	return r.db.WithContext(ctx).
		Model(&channel.Message{}).
		Where("id = ?", messageID).
		Updates(map[string]interface{}{
			"is_deleted": true,
			"body":       "[deleted]",
			"content":    nil,
			"mentions":   channel.MessageMentions{},
		}).Error
}

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

func (r *channelRepository) SearchMessages(ctx context.Context, channelID int64, query string, limit int) ([]*channel.Message, error) {
	var messages []*channel.Message
	likePattern := "%" + query + "%"
	if err := r.db.WithContext(ctx).
		Where("channel_id = ? AND is_deleted = FALSE AND body LIKE ?", channelID, likePattern).
		Preload("SenderUser").
		Preload("SenderPodInfo").
		Preload("SenderPodInfo.Agent").
		Order("created_at DESC, id DESC").
		Limit(limit).
		Find(&messages).Error; err != nil {
		return nil, err
	}
	return messages, nil
}

func (r *channelRepository) SaveMessageEdit(ctx context.Context, edit *channel.MessageEdit) error {
	return r.db.WithContext(ctx).Create(edit).Error
}

func (r *channelRepository) GetMessageEdits(ctx context.Context, messageID int64) ([]*channel.MessageEdit, error) {
	var edits []*channel.MessageEdit
	if err := r.db.WithContext(ctx).
		Where("message_id = ?", messageID).
		Order("created_at ASC").
		Find(&edits).Error; err != nil {
		return nil, err
	}
	return edits, nil
}