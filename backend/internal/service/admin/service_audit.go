package admin

import (
	"context"
	"net"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/admin"
)

// LogAction records an admin action in the audit log
func (s *Service) LogAction(ctx context.Context, entry *admin.AuditLogEntry) error {
	log, err := entry.ToAuditLog()
	if err != nil {
		return err
	}
	return s.db.Create(log)
}

// LogActionFromContext creates and logs an audit entry from common parameters
func (s *Service) LogActionFromContext(
	ctx context.Context,
	adminUserID int64,
	action admin.AuditAction,
	targetType admin.TargetType,
	targetID int64,
	oldData interface{},
	newData interface{},
	ipAddress string,
	userAgent string,
) error {
	var ip net.IP
	if ipAddress != "" {
		ip = net.ParseIP(ipAddress)
	}

	entry := &admin.AuditLogEntry{
		AdminUserID: adminUserID,
		Action:      action,
		TargetType:  targetType,
		TargetID:    targetID,
		OldData:     oldData,
		NewData:     newData,
		IPAddress:   ip,
		UserAgent:   userAgent,
	}

	return s.LogAction(ctx, entry)
}

// GetAuditLogs retrieves audit logs with filtering and pagination
func (s *Service) GetAuditLogs(ctx context.Context, query *admin.AuditLogQuery) (*admin.AuditLogListResponse, error) {
	db := s.db.Model(&admin.AuditLog{})

	// Apply filters
	if query.AdminUserID != nil {
		db = db.Where("admin_user_id = ?", *query.AdminUserID)
	}
	if query.Action != nil {
		db = db.Where("action = ?", *query.Action)
	}
	if query.TargetType != nil {
		db = db.Where("target_type = ?", *query.TargetType)
	}
	if query.TargetID != nil {
		db = db.Where("target_id = ?", *query.TargetID)
	}
	if query.StartTime != nil {
		db = db.Where("created_at >= ?", *query.StartTime)
	}
	if query.EndTime != nil {
		db = db.Where("created_at <= ?", *query.EndTime)
	}

	// Count total
	var total int64
	if err := db.Count(&total); err != nil {
		return nil, err
	}

	// Apply pagination using helper
	p := normalizePagination(query.Page, query.PageSize, total)

	var logs []admin.AuditLog
	if err := db.
		Preload("AdminUser").
		Order("created_at DESC").
		Limit(p.PageSize).
		Offset(p.Offset).
		Find(&logs); err != nil {
		return nil, err
	}

	return &admin.AuditLogListResponse{
		Data:       logs,
		Total:      total,
		Page:       p.Page,
		PageSize:   p.PageSize,
		TotalPages: p.TotalPages,
	}, nil
}
