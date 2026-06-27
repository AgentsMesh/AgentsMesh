package infra

import (
	"context"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
)

// Mention containment has no portable SQL form: production Postgres uses
// precise GIN-indexed JSONB ops (idx_channel_messages_mentions_pod); the
// SQLite unit-test harness uses JSON1 (json_each/json_extract). Both count
// unread messages that @-mention the member directly or via @channel.
const mentionPredPostgres = `msg.mentions @> jsonb_build_object('users', jsonb_build_array(cm.user_id)) ` +
	`OR (msg.mentions->>'channel')::boolean = TRUE`

const mentionPredSQLite = `EXISTS (SELECT 1 FROM json_each(msg.mentions, '$.users') WHERE value = cm.user_id) ` +
	`OR json_extract(msg.mentions, '$.channel') = 1`

func (r *channelRepository) MarkRead(ctx context.Context, channelID, userID int64, messageID int64) error {
	// Monotonic via CASE (portable max — SQLite has no GREATEST) so the cursor
	// never rewinds, while manually_unread is cleared unconditionally: reading
	// a channel always dismisses a "mark unread" even if no new messages.
	return r.db.WithContext(ctx).Exec(`
		INSERT INTO channel_read_states (channel_id, user_id, last_read_message_id, last_read_at, manually_unread)
		VALUES (?, ?, ?, ?, FALSE)
		ON CONFLICT (channel_id, user_id) DO UPDATE
		SET last_read_message_id = CASE
		        WHEN EXCLUDED.last_read_message_id > COALESCE(channel_read_states.last_read_message_id, 0)
		        THEN EXCLUDED.last_read_message_id
		        ELSE COALESCE(channel_read_states.last_read_message_id, 0)
		    END,
		    last_read_at = EXCLUDED.last_read_at,
		    manually_unread = FALSE
	`, channelID, userID, messageID, time.Now()).Error
}

func (r *channelRepository) SetManuallyUnread(ctx context.Context, channelID, userID int64) error {
	// Seed the cursor to the latest message on a first-time row: a never-opened
	// channel marked unread then shows the sticky dot (unread=0 + manually_unread)
	// rather than unread=COUNT(all), and the divider hook gets a real cursor
	// instead of a NULL→0 it cannot anchor on. Existing rows keep their cursor.
	return r.db.WithContext(ctx).Exec(`
		INSERT INTO channel_read_states (channel_id, user_id, last_read_message_id, manually_unread, last_read_at)
		VALUES (?, ?, (SELECT COALESCE(MAX(id), 0) FROM channel_messages WHERE channel_id = ? AND is_deleted = FALSE), TRUE, ?)
		ON CONFLICT (channel_id, user_id) DO UPDATE
		SET manually_unread = TRUE
	`, channelID, userID, channelID, time.Now()).Error
}

func (r *channelRepository) GetReadByUserIDs(ctx context.Context, channelID, messageID int64) ([]int64, error) {
	var ids []int64
	// Join current membership so receipts never list users who have since left.
	err := r.db.WithContext(ctx).Raw(`
		SELECT crs.user_id FROM channel_read_states crs
		JOIN channel_members cm ON cm.channel_id = crs.channel_id AND cm.user_id = crs.user_id
		WHERE crs.channel_id = ? AND crs.last_read_message_id >= ?
	`, channelID, messageID).Scan(&ids).Error
	return ids, err
}

func (r *channelRepository) GetChannelSummaries(ctx context.Context, userID int64) (map[int64]channel.ChannelSummary, error) {
	type result struct {
		ChannelID      int64 `gorm:"column:channel_id"`
		Unread         int64 `gorm:"column:unread"`
		Mention        int64 `gorm:"column:mention"`
		LastRead       int64 `gorm:"column:last_read"`
		ManuallyUnread bool  `gorm:"column:manually_unread"`
	}

	mentionPred := mentionPredPostgres
	if r.db.Name() == "sqlite" {
		mentionPred = mentionPredSQLite
	}

	// Two correlated COUNT(*) subqueries, not one JOIN+GROUP BY over
	// channel_messages: the subqueries probe the (channel_id, id) WHERE NOT
	// is_deleted partial index as index-range counts, whereas a GROUP BY fans
	// out one row per (member × message) and materializes every message in
	// every joined channel before aggregating — far worse for busy channels.
	var results []result
	err := r.db.WithContext(ctx).Raw(`
		SELECT cm.channel_id,
			(SELECT COUNT(*) FROM channel_messages msg
			 WHERE msg.channel_id = cm.channel_id AND msg.is_deleted = FALSE
			   AND msg.id > COALESCE(crs.last_read_message_id, 0)) AS unread,
			(SELECT COUNT(*) FROM channel_messages msg
			 WHERE msg.channel_id = cm.channel_id AND msg.is_deleted = FALSE
			   AND msg.id > COALESCE(crs.last_read_message_id, 0)
			   AND (`+mentionPred+`)) AS mention,
			COALESCE(crs.last_read_message_id, 0) AS last_read,
			COALESCE(crs.manually_unread, FALSE) AS manually_unread
		FROM channel_members cm
		LEFT JOIN channel_read_states crs
			ON crs.channel_id = cm.channel_id AND crs.user_id = cm.user_id
		WHERE cm.user_id = ?
	`, userID).Scan(&results).Error

	if err != nil {
		return nil, err
	}

	out := make(map[int64]channel.ChannelSummary, len(results))
	for _, row := range results {
		out[row.ChannelID] = channel.ChannelSummary{
			Unread:         row.Unread,
			Mention:        row.Mention,
			LastRead:       row.LastRead,
			ManuallyUnread: row.ManuallyUnread,
		}
	}
	return out, nil
}
