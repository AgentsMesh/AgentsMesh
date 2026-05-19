package infra

import (
	"context"

	"gorm.io/gorm"
)

type channelUserLookup struct {
	db *gorm.DB
}

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

type channelPodLookup struct {
	db *gorm.DB
}

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
