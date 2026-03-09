package infra

import (
	"context"
	"errors"

	"github.com/anthropics/agentsmesh/backend/internal/domain/notification"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Compile-time interface compliance check.
var _ notification.PreferenceRepository = (*notificationPreferenceRepo)(nil)

type notificationPreferenceRepo struct {
	db *gorm.DB
}

// NewNotificationPreferenceRepository creates a new GORM-backed PreferenceRepository.
func NewNotificationPreferenceRepository(db *gorm.DB) notification.PreferenceRepository {
	return &notificationPreferenceRepo{db: db}
}

func (r *notificationPreferenceRepo) GetPreference(ctx context.Context, userID int64, source string, entityID string) (*notification.PreferenceRecord, error) {
	var record notification.PreferenceRecord
	query := r.db.WithContext(ctx).Where("user_id = ? AND source = ? AND entity_id = ?", userID, source, entityID)

	if err := query.First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &record, nil
}

func (r *notificationPreferenceRepo) SetPreference(ctx context.Context, record *notification.PreferenceRecord) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "user_id"},
			{Name: "source"},
			{Name: "entity_id"},
		},
		DoUpdates: clause.AssignmentColumns([]string{"is_muted", "channels"}),
	}).Create(record).Error
}

func (r *notificationPreferenceRepo) ListPreferences(ctx context.Context, userID int64) ([]notification.PreferenceRecord, error) {
	var records []notification.PreferenceRecord
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}

func (r *notificationPreferenceRepo) DeletePreference(ctx context.Context, userID int64, source string, entityID string) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND source = ? AND entity_id = ?", userID, source, entityID).
		Delete(&notification.PreferenceRecord{}).Error
}
