package infra

import (
	"context"

	"gorm.io/gorm"
)

// --- Cleanup (application-layer referential integrity, replaces FK CASCADE) ---

// DeleteWithCleanup deletes a channel and all associated data in a single transaction.
// Order: child tables first, then the channel itself.
func (r *channelRepository) DeleteWithCleanup(ctx context.Context, channelID int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Child tables (order doesn't matter since no FK)
		tx.Exec("DELETE FROM channel_messages WHERE channel_id = ?", channelID)
		tx.Exec("DELETE FROM channel_members WHERE channel_id = ?", channelID)
		tx.Exec("DELETE FROM channel_read_states WHERE channel_id = ?", channelID)
		tx.Exec("DELETE FROM channel_pods WHERE channel_id = ?", channelID)
		tx.Exec("DELETE FROM channel_access WHERE channel_id = ?", channelID)
		tx.Exec("DELETE FROM pod_bindings WHERE channel_id = ?", channelID)
		tx.Exec("DELETE FROM notification_preferences WHERE source IN ('channel:message','channel:mention') AND entity_id = ?", channelID)

		// Finally delete the channel itself
		return tx.Exec("DELETE FROM channels WHERE id = ?", channelID).Error
	})
}

// DeleteChannelsByOrg deletes all channels and associated data for an organization.
// Used when deleting an organization (replaces FK CASCADE on channels.organization_id).
func (r *channelRepository) DeleteChannelsByOrg(ctx context.Context, orgID int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Subquery for channel IDs in this org
		subq := "SELECT id FROM channels WHERE organization_id = ?"

		tx.Exec("DELETE FROM channel_messages WHERE channel_id IN ("+subq+")", orgID)
		tx.Exec("DELETE FROM channel_members WHERE channel_id IN ("+subq+")", orgID)
		tx.Exec("DELETE FROM channel_read_states WHERE channel_id IN ("+subq+")", orgID)
		tx.Exec("DELETE FROM channel_pods WHERE channel_id IN ("+subq+")", orgID)
		tx.Exec("DELETE FROM channel_access WHERE channel_id IN ("+subq+")", orgID)
		tx.Exec("DELETE FROM pod_bindings WHERE channel_id IN ("+subq+")", orgID)

		// Delete the channels themselves
		return tx.Exec("DELETE FROM channels WHERE organization_id = ?", orgID).Error
	})
}

// CleanupUserReferences removes all channel-related data for a deleted user.
// Messages are preserved but sender_user_id is nullified (messages belong to the channel, not the user).
func (r *channelRepository) CleanupUserReferences(ctx context.Context, userID int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		tx.Exec("DELETE FROM channel_members WHERE user_id = ?", userID)
		tx.Exec("DELETE FROM channel_read_states WHERE user_id = ?", userID)
		tx.Exec("DELETE FROM channel_access WHERE user_id = ?", userID)
		tx.Exec("DELETE FROM notification_preferences WHERE user_id = ?", userID)
		// Nullify sender references — messages are retained for channel history
		tx.Exec("UPDATE channel_messages SET sender_user_id = NULL WHERE sender_user_id = ?", userID)
		// Nullify creator references on channels
		tx.Exec("UPDATE channels SET created_by_user_id = NULL WHERE created_by_user_id = ?", userID)
		return nil
	})
}
