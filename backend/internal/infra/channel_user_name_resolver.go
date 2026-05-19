package infra

import (
	"context"

	"gorm.io/gorm"
)

type channelUserNameResolver struct {
	db *gorm.DB
}

func NewChannelUserNameResolver(db *gorm.DB) *channelUserNameResolver {
	return &channelUserNameResolver{db: db}
}

func (r *channelUserNameResolver) GetUsername(ctx context.Context, userID int64) (string, error) {
	var result struct {
		Username string
		Name     string
	}
	err := r.db.WithContext(ctx).
		Table("users").
		Select("username, name").
		Where("id = ?", userID).
		Scan(&result).Error
	if err != nil {
		return "", err
	}
	if result.Name != "" {
		return result.Name, nil
	}
	return result.Username, nil
}
