package admin

import (
	"context"
	"errors"
	"testing"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListUsers(t *testing.T) {
	t.Run("should list users with pagination", func(t *testing.T) {
		db := newMockDB()
		db.users[1] = &user.User{ID: 1, Email: "user1@example.com", IsActive: true}
		db.users[2] = &user.User{ID: 2, Email: "user2@example.com", IsActive: true}
		db.totalUsers = 2

		svc := NewService(db)
		result, err := svc.ListUsers(context.Background(), &UserListQuery{
			Page:     1,
			PageSize: 20,
		})

		require.NoError(t, err)
		assert.Equal(t, int64(2), result.Total)
		assert.Equal(t, 1, result.Page)
		assert.Equal(t, 20, result.PageSize)
	})

	t.Run("should handle empty result", func(t *testing.T) {
		db := newMockDB()
		db.totalUsers = 0

		svc := NewService(db)
		result, err := svc.ListUsers(context.Background(), &UserListQuery{
			Page:     1,
			PageSize: 20,
		})

		require.NoError(t, err)
		assert.Equal(t, int64(0), result.Total)
		assert.Empty(t, result.Data)
	})
}

func TestGetUser(t *testing.T) {
	t.Run("should return user when found", func(t *testing.T) {
		db := newMockDB()
		db.users[1] = &user.User{ID: 1, Email: "test@example.com"}

		svc := NewService(db)
		u, err := svc.GetUser(context.Background(), 1)

		require.NoError(t, err)
		assert.Equal(t, int64(1), u.ID)
		assert.Equal(t, "test@example.com", u.Email)
	})

	t.Run("should return error when user not found", func(t *testing.T) {
		db := newMockDB()

		svc := NewService(db)
		u, err := svc.GetUser(context.Background(), 999)

		assert.Error(t, err)
		assert.Equal(t, ErrUserNotFound, err)
		assert.Nil(t, u)
	})
}

func TestUpdateUser(t *testing.T) {
	t.Run("should update user successfully", func(t *testing.T) {
		db := newMockDB()
		oldName := "Old Name"
		db.users[1] = &user.User{ID: 1, Email: "old@example.com", Name: &oldName}

		svc := NewService(db)
		u, err := svc.UpdateUser(context.Background(), 1, map[string]interface{}{
			"name": "New Name",
		})

		require.NoError(t, err)
		assert.NotNil(t, u)
	})

	t.Run("should return error when user not found", func(t *testing.T) {
		db := newMockDB()

		svc := NewService(db)
		u, err := svc.UpdateUser(context.Background(), 999, map[string]interface{}{
			"name": "New Name",
		})

		assert.Error(t, err)
		assert.Equal(t, ErrUserNotFound, err)
		assert.Nil(t, u)
	})
}

func TestDisableUser(t *testing.T) {
	t.Run("should disable user successfully", func(t *testing.T) {
		db := newMockDB()
		db.users[1] = &user.User{ID: 1, IsActive: true}

		svc := NewService(db)
		u, err := svc.DisableUser(context.Background(), 1)

		require.NoError(t, err)
		assert.NotNil(t, u)
	})
}

func TestEnableUser(t *testing.T) {
	t.Run("should enable user successfully", func(t *testing.T) {
		db := newMockDB()
		db.users[1] = &user.User{ID: 1, IsActive: false}

		svc := NewService(db)
		u, err := svc.EnableUser(context.Background(), 1)

		require.NoError(t, err)
		assert.NotNil(t, u)
	})
}

func TestGrantAdmin(t *testing.T) {
	t.Run("should grant admin privileges successfully", func(t *testing.T) {
		db := newMockDB()
		db.users[1] = &user.User{ID: 1, IsSystemAdmin: false}

		svc := NewService(db)
		u, err := svc.GrantAdmin(context.Background(), 1)

		require.NoError(t, err)
		assert.NotNil(t, u)
	})
}

func TestRevokeAdmin(t *testing.T) {
	t.Run("should revoke admin privileges successfully", func(t *testing.T) {
		db := newMockDB()
		db.users[1] = &user.User{ID: 1, IsSystemAdmin: true}

		svc := NewService(db)
		u, err := svc.RevokeAdmin(context.Background(), 1, 2) // Admin 2 revoking user 1

		require.NoError(t, err)
		assert.NotNil(t, u)
	})

	t.Run("should prevent revoking own admin privileges", func(t *testing.T) {
		db := newMockDB()
		db.users[1] = &user.User{ID: 1, IsSystemAdmin: true}

		svc := NewService(db)
		u, err := svc.RevokeAdmin(context.Background(), 1, 1) // Admin 1 trying to revoke self

		assert.Error(t, err)
		assert.Equal(t, ErrCannotRevokeOwnAdmin, err)
		assert.Nil(t, u)
	})
}

func TestListUsers_WithFilters(t *testing.T) {
	t.Run("should filter by search term", func(t *testing.T) {
		db := newMockDB()
		db.users[1] = &user.User{ID: 1, Email: "test@example.com"}
		db.totalUsers = 1

		svc := NewService(db)
		result, err := svc.ListUsers(context.Background(), &UserListQuery{
			Page:     1,
			PageSize: 20,
			Search:   "test",
		})

		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("should filter by is_active", func(t *testing.T) {
		db := newMockDB()
		isActive := true

		svc := NewService(db)
		result, err := svc.ListUsers(context.Background(), &UserListQuery{
			Page:     1,
			PageSize: 20,
			IsActive: &isActive,
		})

		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("should filter by is_admin", func(t *testing.T) {
		db := newMockDB()
		isAdmin := true

		svc := NewService(db)
		result, err := svc.ListUsers(context.Background(), &UserListQuery{
			Page:     1,
			PageSize: 20,
			IsAdmin:  &isAdmin,
		})

		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("should return error when count fails", func(t *testing.T) {
		db := newMockDB()
		db.countErr = errors.New("count failed")

		svc := NewService(db)
		result, err := svc.ListUsers(context.Background(), &UserListQuery{
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
		result, err := svc.ListUsers(context.Background(), &UserListQuery{
			Page:     1,
			PageSize: 20,
		})

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestUpdateUser_ErrorPaths(t *testing.T) {
	t.Run("should return error when updates fail", func(t *testing.T) {
		db := newMockDB()
		db.users[1] = &user.User{ID: 1, Email: "test@example.com"}
		db.updatesErr = errors.New("update failed")

		svc := NewService(db)
		u, err := svc.UpdateUser(context.Background(), 1, map[string]interface{}{
			"name": "New Name",
		})

		assert.Error(t, err)
		assert.Nil(t, u)
	})

	t.Run("should return error when reload after update fails", func(t *testing.T) {
		db := newMockDB()
		db.users[1] = &user.User{ID: 1, Email: "test@example.com"}
		// Fail on the second First call (reload after update)
		db.firstErrAtCall = 2

		svc := NewService(db)
		u, err := svc.UpdateUser(context.Background(), 1, map[string]interface{}{
			"name": "New Name",
		})

		assert.Error(t, err)
		assert.Nil(t, u)
	})
}
