package middleware

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

func LogAction(db *gorm.DB, action AuditAction, opts *LogActionOptions) error {
	log := &AuditLog{
		Action:       string(action),
		ResourceType: opts.ResourceType,
		ActorType:    opts.ActorType,
		StatusCode:   opts.StatusCode,
		CreatedAt:    time.Now(),
	}

	if opts.OrganizationID > 0 {
		log.OrganizationID = &opts.OrganizationID
	}
	if opts.ActorID > 0 {
		log.ActorID = &opts.ActorID
	}
	if opts.ResourceID > 0 {
		log.ResourceID = &opts.ResourceID
	}
	if opts.IPAddress != "" {
		log.IPAddress = &opts.IPAddress
	}
	if opts.UserAgent != "" {
		log.UserAgent = &opts.UserAgent
	}
	if opts.Details != nil {
		detailsJSON, _ := json.Marshal(opts.Details)
		log.Details = detailsJSON
	}

	return db.Create(log).Error
}

func QueryAuditLogs(db *gorm.DB, filter *AuditLogFilter) ([]AuditLog, int64, error) {
	var logs []AuditLog
	var total int64

	query := db.Model(&AuditLog{})

	if filter.OrganizationID > 0 {
		query = query.Where("organization_id = ?", filter.OrganizationID)
	}
	if filter.ActorID > 0 {
		query = query.Where("actor_id = ?", filter.ActorID)
	}
	if filter.Action != "" {
		query = query.Where("action = ?", filter.Action)
	}
	if filter.ResourceType != "" {
		query = query.Where("resource_type = ?", filter.ResourceType)
	}
	if filter.ResourceID > 0 {
		query = query.Where("resource_id = ?", filter.ResourceID)
	}
	if !filter.StartTime.IsZero() {
		query = query.Where("created_at >= ?", filter.StartTime)
	}
	if !filter.EndTime.IsZero() {
		query = query.Where("created_at <= ?", filter.EndTime)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}
	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}

	query = query.Order("created_at DESC")

	if err := query.Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}
