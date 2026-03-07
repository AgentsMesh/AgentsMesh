package infra

import (
	"context"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	"gorm.io/gorm"
)

// Compile-time interface check
var _ agent.MessageRepository = (*agentMessageRepo)(nil)

type agentMessageRepo struct {
	db *gorm.DB
}

// NewAgentMessageRepository creates a new GORM-based agent message repository
func NewAgentMessageRepository(db *gorm.DB) agent.MessageRepository {
	return &agentMessageRepo{db: db}
}

func (r *agentMessageRepo) Create(ctx context.Context, message *agent.AgentMessage) error {
	return r.db.WithContext(ctx).Create(message).Error
}

func (r *agentMessageRepo) GetByID(ctx context.Context, id int64) (*agent.AgentMessage, error) {
	var message agent.AgentMessage
	if err := r.db.WithContext(ctx).First(&message, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &message, nil
}

func (r *agentMessageRepo) Save(ctx context.Context, message *agent.AgentMessage) error {
	return r.db.WithContext(ctx).Save(message).Error
}

func (r *agentMessageRepo) Delete(ctx context.Context, message *agent.AgentMessage) error {
	return r.db.WithContext(ctx).Delete(message).Error
}

func (r *agentMessageRepo) UpdateStatus(ctx context.Context, messageID int64, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&agent.AgentMessage{}).
		Where("id = ?", messageID).
		Updates(updates).Error
}

func (r *agentMessageRepo) MarkAllRead(ctx context.Context, podKey string) (int64, error) {
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&agent.AgentMessage{}).
		Where("receiver_pod = ? AND status IN ?", podKey,
			[]string{agent.MessageStatusPending, agent.MessageStatusDelivered}).
		Updates(map[string]interface{}{
			"status":  agent.MessageStatusRead,
			"read_at": now,
		})
	return result.RowsAffected, result.Error
}

func (r *agentMessageRepo) GetMessages(ctx context.Context, podKey string, unreadOnly bool, messageTypes []string, limit, offset int) ([]*agent.AgentMessage, error) {
	query := r.db.WithContext(ctx).Where("receiver_pod = ?", podKey)

	if unreadOnly {
		query = query.Where("status IN ?", []string{agent.MessageStatusPending, agent.MessageStatusDelivered})
	}
	if len(messageTypes) > 0 {
		query = query.Where("message_type IN ?", messageTypes)
	}

	var messages []*agent.AgentMessage
	err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&messages).Error
	return messages, err
}

func (r *agentMessageRepo) GetUnreadMessages(ctx context.Context, podKey string, limit int) ([]*agent.AgentMessage, error) {
	var messages []*agent.AgentMessage
	err := r.db.WithContext(ctx).
		Where("receiver_pod = ? AND status IN ?", podKey,
			[]string{agent.MessageStatusPending, agent.MessageStatusDelivered}).
		Order("created_at ASC").
		Limit(limit).
		Find(&messages).Error
	return messages, err
}

func (r *agentMessageRepo) GetUnreadCount(ctx context.Context, podKey string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&agent.AgentMessage{}).
		Where("receiver_pod = ? AND status IN ?", podKey,
			[]string{agent.MessageStatusPending, agent.MessageStatusDelivered}).
		Count(&count).Error
	return count, err
}

func (r *agentMessageRepo) GetConversation(ctx context.Context, correlationID string, limit int) ([]*agent.AgentMessage, error) {
	var messages []*agent.AgentMessage
	err := r.db.WithContext(ctx).
		Where("correlation_id = ?", correlationID).
		Order("created_at ASC").Limit(limit).
		Find(&messages).Error
	return messages, err
}

func (r *agentMessageRepo) GetReplies(ctx context.Context, parentMessageID int64) ([]*agent.AgentMessage, error) {
	var replies []*agent.AgentMessage
	err := r.db.WithContext(ctx).
		Where("parent_message_id = ?", parentMessageID).
		Order("created_at ASC").
		Find(&replies).Error
	return replies, err
}

func (r *agentMessageRepo) GetSentMessages(ctx context.Context, podKey string, limit, offset int) ([]*agent.AgentMessage, error) {
	var messages []*agent.AgentMessage
	err := r.db.WithContext(ctx).
		Where("sender_pod = ?", podKey).
		Order("created_at DESC").Limit(limit).Offset(offset).
		Find(&messages).Error
	return messages, err
}

func (r *agentMessageRepo) GetMessagesBetween(ctx context.Context, podA, podB string, limit int) ([]*agent.AgentMessage, error) {
	var messages []*agent.AgentMessage
	err := r.db.WithContext(ctx).
		Where("(sender_pod = ? AND receiver_pod = ?) OR (sender_pod = ? AND receiver_pod = ?)",
			podA, podB, podB, podA).
		Order("created_at ASC").Limit(limit).
		Find(&messages).Error
	return messages, err
}

func (r *agentMessageRepo) GetPendingRetries(ctx context.Context, before time.Time, limit int) ([]*agent.AgentMessage, error) {
	var messages []*agent.AgentMessage
	err := r.db.WithContext(ctx).
		Where("status = ? AND next_retry_at IS NOT NULL AND next_retry_at <= ?",
			agent.MessageStatusFailed, before).
		Order("next_retry_at ASC").Limit(limit).
		Find(&messages).Error
	return messages, err
}

func (r *agentMessageRepo) CreateDeadLetter(ctx context.Context, entry *agent.DeadLetterEntry) error {
	return r.db.WithContext(ctx).Create(entry).Error
}

func (r *agentMessageRepo) GetDeadLetters(ctx context.Context, limit, offset int) ([]*agent.DeadLetterEntry, error) {
	var entries []*agent.DeadLetterEntry
	err := r.db.WithContext(ctx).
		Preload("OriginalMessage").
		Order("moved_at DESC").Limit(limit).Offset(offset).
		Find(&entries).Error
	return entries, err
}

func (r *agentMessageRepo) GetDeadLetterWithMessage(ctx context.Context, id int64) (*agent.DeadLetterEntry, error) {
	var entry agent.DeadLetterEntry
	err := r.db.WithContext(ctx).
		Preload("OriginalMessage").
		First(&entry, id).Error
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

func (r *agentMessageRepo) SaveDeadLetter(ctx context.Context, entry *agent.DeadLetterEntry) error {
	return r.db.WithContext(ctx).Save(entry).Error
}

func (r *agentMessageRepo) CleanupExpiredDeadLetters(ctx context.Context, olderThan time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("moved_at < ?", olderThan).
		Delete(&agent.DeadLetterEntry{})
	return result.RowsAffected, result.Error
}
