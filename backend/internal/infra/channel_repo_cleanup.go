package infra

import (
	"context"

	"gorm.io/gorm"
)

func (r *channelRepository) DeleteWithCleanup(ctx context.Context, channelID int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		tx.Exec("DELETE FROM channel_messages WHERE channel_id = ?", channelID)
		tx.Exec("DELETE FROM channel_members WHERE channel_id = ?", channelID)
		tx.Exec("DELETE FROM channel_read_states WHERE channel_id = ?", channelID)
		tx.Exec("DELETE FROM channel_pods WHERE channel_id = ?", channelID)
		tx.Exec("DELETE FROM channel_access WHERE channel_id = ?", channelID)
		tx.Exec("DELETE FROM pod_bindings WHERE channel_id = ?", channelID)
		tx.Exec("DELETE FROM notification_preferences WHERE source IN ('channel:message','channel:mention') AND entity_id = ?", channelID)

		return tx.Exec("DELETE FROM channels WHERE id = ?", channelID).Error
	})
}

func (r *channelRepository) DeleteChannelsByOrg(ctx context.Context, orgID int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		subq := "SELECT id FROM channels WHERE organization_id = ?"

		tx.Exec("DELETE FROM channel_messages WHERE channel_id IN ("+subq+")", orgID)
		tx.Exec("DELETE FROM channel_members WHERE channel_id IN ("+subq+")", orgID)
		tx.Exec("DELETE FROM channel_read_states WHERE channel_id IN ("+subq+")", orgID)
		tx.Exec("DELETE FROM channel_pods WHERE channel_id IN ("+subq+")", orgID)
		tx.Exec("DELETE FROM channel_access WHERE channel_id IN ("+subq+")", orgID)
		tx.Exec("DELETE FROM pod_bindings WHERE channel_id IN ("+subq+")", orgID)

		return tx.Exec("DELETE FROM channels WHERE organization_id = ?", orgID).Error
	})
}

func (r *channelRepository) CleanupUserReferences(ctx context.Context, userID int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		tx.Exec("DELETE FROM channel_members WHERE user_id = ?", userID)
		tx.Exec("DELETE FROM channel_read_states WHERE user_id = ?", userID)
		tx.Exec("DELETE FROM channel_access WHERE user_id = ?", userID)
		tx.Exec("DELETE FROM notification_preferences WHERE user_id = ?", userID)
		tx.Exec("UPDATE channel_messages SET sender_user_id = NULL WHERE sender_user_id = ?", userID)
		tx.Exec("UPDATE channels SET created_by_user_id = NULL WHERE created_by_user_id = ?", userID)
		return nil
	})
}
