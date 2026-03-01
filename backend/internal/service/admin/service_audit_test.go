package admin

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/admin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// unserializable is a type that cannot be JSON marshaled
type unserializable struct {
	Channel chan int `json:"channel"` // channels cannot be marshaled
}

func TestLogAction(t *testing.T) {
	t.Run("should log action successfully", func(t *testing.T) {
		db := newMockDB()

		svc := NewService(db)
		err := svc.LogAction(context.Background(), &admin.AuditLogEntry{
			AdminUserID: 1,
			Action:      admin.AuditActionUserView,
			TargetType:  admin.TargetTypeUser,
			TargetID:    2,
		})

		require.NoError(t, err)
		assert.Len(t, db.auditLogs, 1)
		assert.Equal(t, admin.AuditActionUserView, db.auditLogs[0].Action)
	})

	t.Run("should return error when create fails", func(t *testing.T) {
		db := newMockDB()
		db.createErr = errors.New("database error")

		svc := NewService(db)
		err := svc.LogAction(context.Background(), &admin.AuditLogEntry{
			AdminUserID: 1,
			Action:      admin.AuditActionUserView,
			TargetType:  admin.TargetTypeUser,
			TargetID:    2,
		})

		assert.Error(t, err)
	})

	t.Run("should return error when OldData cannot be marshaled", func(t *testing.T) {
		db := newMockDB()

		svc := NewService(db)
		err := svc.LogAction(context.Background(), &admin.AuditLogEntry{
			AdminUserID: 1,
			Action:      admin.AuditActionUserView,
			TargetType:  admin.TargetTypeUser,
			TargetID:    2,
			OldData:     unserializable{Channel: make(chan int)}, // Cannot be JSON marshaled
		})

		assert.Error(t, err)
	})

	t.Run("should return error when NewData cannot be marshaled", func(t *testing.T) {
		db := newMockDB()

		svc := NewService(db)
		err := svc.LogAction(context.Background(), &admin.AuditLogEntry{
			AdminUserID: 1,
			Action:      admin.AuditActionUserView,
			TargetType:  admin.TargetTypeUser,
			TargetID:    2,
			NewData:     unserializable{Channel: make(chan int)}, // Cannot be JSON marshaled
		})

		assert.Error(t, err)
	})
}

func TestLogActionFromContext(t *testing.T) {
	t.Run("should log action with all parameters", func(t *testing.T) {
		db := newMockDB()

		svc := NewService(db)
		err := svc.LogActionFromContext(
			context.Background(),
			1,
			admin.AuditActionUserDisable,
			admin.TargetTypeUser,
			2,
			map[string]interface{}{"is_active": true},
			map[string]interface{}{"is_active": false},
			"192.168.1.1",
			"Mozilla/5.0",
		)

		require.NoError(t, err)
		assert.Len(t, db.auditLogs, 1)
		assert.Equal(t, admin.AuditActionUserDisable, db.auditLogs[0].Action)
	})
}

func TestGetAuditLogs(t *testing.T) {
	t.Run("should return audit logs with pagination", func(t *testing.T) {
		db := newMockDB()
		now := time.Now()
		db.auditLogs = []admin.AuditLog{
			{ID: 1, AdminUserID: 1, Action: admin.AuditActionUserView, TargetType: admin.TargetTypeUser, TargetID: 2, CreatedAt: now},
			{ID: 2, AdminUserID: 1, Action: admin.AuditActionUserDisable, TargetType: admin.TargetTypeUser, TargetID: 3, CreatedAt: now},
		}
		db.totalUsers = 2 // Using totalUsers as a proxy for audit log count in mock

		svc := NewService(db)
		result, err := svc.GetAuditLogs(context.Background(), &admin.AuditLogQuery{
			Page:     1,
			PageSize: 20,
		})

		require.NoError(t, err)
		assert.Len(t, result.Data, 2)
	})

	t.Run("should filter by admin user ID", func(t *testing.T) {
		db := newMockDB()
		adminUserID := int64(1)

		svc := NewService(db)
		result, err := svc.GetAuditLogs(context.Background(), &admin.AuditLogQuery{
			Page:        1,
			PageSize:    20,
			AdminUserID: &adminUserID,
		})

		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("should filter by action", func(t *testing.T) {
		db := newMockDB()
		action := admin.AuditActionUserView

		svc := NewService(db)
		result, err := svc.GetAuditLogs(context.Background(), &admin.AuditLogQuery{
			Page:     1,
			PageSize: 20,
			Action:   &action,
		})

		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("should filter by target type and ID", func(t *testing.T) {
		db := newMockDB()
		targetType := admin.TargetTypeUser
		targetID := int64(2)

		svc := NewService(db)
		result, err := svc.GetAuditLogs(context.Background(), &admin.AuditLogQuery{
			Page:       1,
			PageSize:   20,
			TargetType: &targetType,
			TargetID:   &targetID,
		})

		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("should filter by date range", func(t *testing.T) {
		db := newMockDB()
		now := time.Now()
		startTime := now.AddDate(0, 0, -7)
		endTime := now

		svc := NewService(db)
		result, err := svc.GetAuditLogs(context.Background(), &admin.AuditLogQuery{
			Page:      1,
			PageSize:  20,
			StartTime: &startTime,
			EndTime:   &endTime,
		})

		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("should return error when count fails", func(t *testing.T) {
		db := newMockDB()
		db.countErr = errors.New("count failed")

		svc := NewService(db)
		result, err := svc.GetAuditLogs(context.Background(), &admin.AuditLogQuery{
			Page:     1,
			PageSize: 20,
		})

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("should return error when find fails", func(t *testing.T) {
		db := newMockDB()
		db.findErr = errors.New("find failed")

		svc := NewService(db)
		result, err := svc.GetAuditLogs(context.Background(), &admin.AuditLogQuery{
			Page:     1,
			PageSize: 20,
		})

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}
