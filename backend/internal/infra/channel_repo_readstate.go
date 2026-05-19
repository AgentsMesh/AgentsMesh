package infra

import (
	"context"
	"time"
)

func (r *channelRepository) MarkRead(ctx context.Context, channelID, userID int64, messageID int64) error {
	return r.db.WithContext(ctx).Exec(`
		INSERT INTO channel_read_states (channel_id, user_id, last_read_message_id, last_read_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT (channel_id, user_id) DO UPDATE
		SET last_read_message_id = EXCLUDED.last_read_message_id,
		    last_read_at = EXCLUDED.last_read_at
		WHERE channel_read_states.last_read_message_id IS NULL
		   OR EXCLUDED.last_read_message_id > channel_read_states.last_read_message_id
	`, channelID, userID, messageID, time.Now()).Error
}

func (r *channelRepository) GetUnreadCounts(ctx context.Context, userID int64) (map[int64]int64, error) {
	type result struct {
		ChannelID int64 `gorm:"column:channel_id"`
		Count     int64 `gorm:"column:count"`
	}

	var results []result
	err := r.db.WithContext(ctx).Raw(`
		SELECT cm.channel_id,
			(SELECT COUNT(*) FROM channel_messages msg
			 WHERE msg.channel_id = cm.channel_id
			   AND msg.is_deleted = FALSE
			   AND msg.id > COALESCE(crs.last_read_message_id, 0)) as count
		FROM channel_members cm
		LEFT JOIN channel_read_states crs
			ON crs.channel_id = cm.channel_id AND crs.user_id = cm.user_id
		WHERE cm.user_id = ?
	`, userID).Scan(&results).Error

	if err != nil {
		return nil, err
	}

	counts := make(map[int64]int64, len(results))
	for _, r := range results {
		if r.Count > 0 {
			counts[r.ChannelID] = r.Count
		}
	}
	return counts, nil
}
