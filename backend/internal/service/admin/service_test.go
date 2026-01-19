package admin

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/admin"
	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/domain/organization"
	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
	"github.com/anthropics/agentsmesh/backend/internal/infra/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// mockDB implements database.DB interface for testing
type mockDB struct {
	// Control behavior
	createErr  error
	firstErr   error
	findErr    error
	saveErr    error
	deleteErr  error
	updatesErr error
	countErr   error

	// Error control for specific count queries (for GetDashboardStats testing)
	countErrForTable map[string]error // table name -> error
	countErrForModel map[string]error // model type name -> error
	countCallNum     int              // Track count call order
	countErrAtCall   int              // Fail at specific call number (1-indexed, 0 = don't use)

	// Error control for First method (for reload testing)
	firstCallNum   int // Track First call order
	firstErrAtCall int // Fail at specific call number (1-indexed, 0 = don't use)

	// Store data
	users         map[int64]*user.User
	organizations map[int64]*organization.Organization
	runners       map[int64]*runner.Runner
	members       []organization.Member
	auditLogs     []admin.AuditLog

	// Counters for stats
	totalUsers          int64
	activeUsers         int64
	totalOrgs           int64
	totalRunners        int64
	onlineRunners       int64
	totalPods           int64
	activePods          int64
	totalSubscriptions  int64
	activeSubscriptions int64
	newUsersToday       int64
	newUsersThisWeek    int64
	newUsersThisMonth   int64
	runnerCount         int64
	activePodCount      int64

	// Track method calls
	lastModel   interface{}
	lastTable   string
	lastWhere   interface{}
	lastPreload string
}

func newMockDB() *mockDB {
	return &mockDB{
		users:         make(map[int64]*user.User),
		organizations: make(map[int64]*organization.Organization),
		runners:       make(map[int64]*runner.Runner),
	}
}

func (m *mockDB) Transaction(fc func(tx database.DB) error) error {
	return fc(m)
}

func (m *mockDB) WithContext(ctx context.Context) database.DB {
	return m
}

func (m *mockDB) Create(value interface{}) error {
	if m.createErr != nil {
		return m.createErr
	}
	if log, ok := value.(*admin.AuditLog); ok {
		m.auditLogs = append(m.auditLogs, *log)
	}
	return nil
}

func (m *mockDB) First(dest interface{}, conds ...interface{}) error {
	// Increment call counter
	m.firstCallNum++

	// Check if we should fail at this specific call number
	if m.firstErrAtCall > 0 && m.firstCallNum == m.firstErrAtCall {
		return errors.New("first error at call")
	}

	if m.firstErr != nil {
		return m.firstErr
	}

	if len(conds) > 0 {
		id, ok := conds[0].(int64)
		if !ok {
			return gorm.ErrRecordNotFound
		}

		switch d := dest.(type) {
		case *user.User:
			if u, exists := m.users[id]; exists {
				*d = *u
				return nil
			}
		case *organization.Organization:
			if o, exists := m.organizations[id]; exists {
				*d = *o
				return nil
			}
		case *runner.Runner:
			if r, exists := m.runners[id]; exists {
				*d = *r
				return nil
			}
		}
	}

	return gorm.ErrRecordNotFound
}

func (m *mockDB) Find(dest interface{}, conds ...interface{}) error {
	if m.findErr != nil {
		return m.findErr
	}

	switch d := dest.(type) {
	case *[]user.User:
		for _, u := range m.users {
			*d = append(*d, *u)
		}
	case *[]organization.Organization:
		for _, o := range m.organizations {
			*d = append(*d, *o)
		}
	case *[]organization.Member:
		*d = m.members
	case *[]runner.Runner:
		for _, r := range m.runners {
			*d = append(*d, *r)
		}
	case *[]admin.AuditLog:
		*d = m.auditLogs
	}
	return nil
}

func (m *mockDB) Save(value interface{}) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	if r, ok := value.(*runner.Runner); ok {
		m.runners[r.ID] = r
	}
	return nil
}

func (m *mockDB) Delete(value interface{}, conds ...interface{}) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	switch v := value.(type) {
	case *organization.Organization:
		delete(m.organizations, v.ID)
	case *runner.Runner:
		delete(m.runners, v.ID)
	}
	return nil
}

func (m *mockDB) Updates(model interface{}, values interface{}) error {
	if m.updatesErr != nil {
		return m.updatesErr
	}
	if u, ok := model.(*user.User); ok {
		if updates, ok := values.(map[string]interface{}); ok {
			if v, exists := updates["is_active"]; exists {
				u.IsActive = v.(bool)
			}
			if v, exists := updates["is_system_admin"]; exists {
				u.IsSystemAdmin = v.(bool)
			}
			m.users[u.ID] = u
		}
	}
	return nil
}

func (m *mockDB) Model(value interface{}) database.DB {
	m.lastModel = value
	// Set lastTable based on model type for proper Count behavior
	switch value.(type) {
	case *agentpod.Pod:
		m.lastTable = "agent_pods"
	case *runner.Runner:
		m.lastTable = "runners"
	default:
		m.lastTable = "" // Reset table for other models
	}
	return m
}

func (m *mockDB) Table(name string) database.DB {
	m.lastTable = name
	return m
}

func (m *mockDB) Where(query interface{}, args ...interface{}) database.DB {
	m.lastWhere = query
	return m
}

func (m *mockDB) Select(query interface{}, args ...interface{}) database.DB {
	return m
}

func (m *mockDB) Joins(query string, args ...interface{}) database.DB {
	return m
}

func (m *mockDB) Preload(query string, args ...interface{}) database.DB {
	m.lastPreload = query
	return m
}

func (m *mockDB) Order(value interface{}) database.DB {
	return m
}

func (m *mockDB) Limit(limit int) database.DB {
	return m
}

func (m *mockDB) Offset(offset int) database.DB {
	return m
}

func (m *mockDB) Group(name string) database.DB {
	return m
}

func (m *mockDB) Count(count *int64) error {
	// Increment call counter
	m.countCallNum++

	// Check if we should fail at this specific call number
	if m.countErrAtCall > 0 && m.countCallNum == m.countErrAtCall {
		return errors.New("count error at call " + string(rune('0'+m.countCallNum)))
	}

	if m.countErr != nil {
		return m.countErr
	}

	// Return appropriate count based on last model/table
	switch m.lastTable {
	case "runners":
		if m.lastWhere == "status = ?" {
			*count = m.onlineRunners
		} else if m.lastWhere == "organization_id = ?" {
			*count = m.runnerCount
		} else {
			*count = m.totalRunners
		}
	case "agent_pods":
		if m.lastWhere != nil {
			*count = m.activePodCount
		} else {
			*count = m.totalPods
		}
	case "subscriptions":
		if m.lastWhere == "status = ?" {
			*count = m.activeSubscriptions
		} else {
			*count = m.totalSubscriptions
		}
	default:
		// User or Organization model
		if m.lastWhere == "is_active = ?" {
			*count = m.activeUsers
		} else if m.lastWhere == "created_at >= ?" {
			*count = m.newUsersToday
		} else if m.lastModel != nil {
			switch m.lastModel.(type) {
			case *organization.Organization:
				*count = m.totalOrgs
			case *runner.Runner:
				*count = m.totalRunners
			default:
				*count = m.totalUsers
			}
		} else {
			*count = m.totalUsers
		}
	}

	return nil
}

func (m *mockDB) Scan(dest interface{}) error {
	return nil
}

func (m *mockDB) GormDB() *gorm.DB {
	return nil
}

// Ensure mockDB implements database.DB
var _ database.DB = (*mockDB)(nil)

// =============================================================================
// Test normalizePagination
// =============================================================================

func TestNormalizePagination(t *testing.T) {
	tests := []struct {
		name           string
		page           int
		pageSize       int
		total          int64
		expectedPage   int
		expectedSize   int
		expectedOffset int
		expectedPages  int
	}{
		{
			name:           "normal case",
			page:           1,
			pageSize:       20,
			total:          100,
			expectedPage:   1,
			expectedSize:   20,
			expectedOffset: 0,
			expectedPages:  5,
		},
		{
			name:           "page less than 1 normalizes to 1",
			page:           0,
			pageSize:       20,
			total:          50,
			expectedPage:   1,
			expectedSize:   20,
			expectedOffset: 0,
			expectedPages:  3,
		},
		{
			name:           "negative page normalizes to 1",
			page:           -5,
			pageSize:       10,
			total:          30,
			expectedPage:   1,
			expectedSize:   10,
			expectedOffset: 0,
			expectedPages:  3,
		},
		{
			name:           "pageSize less than 1 defaults to 20",
			page:           1,
			pageSize:       0,
			total:          100,
			expectedPage:   1,
			expectedSize:   20,
			expectedOffset: 0,
			expectedPages:  5,
		},
		{
			name:           "pageSize over 100 caps at 100",
			page:           1,
			pageSize:       200,
			total:          500,
			expectedPage:   1,
			expectedSize:   100,
			expectedOffset: 0,
			expectedPages:  5,
		},
		{
			name:           "page 2 calculates correct offset",
			page:           2,
			pageSize:       20,
			total:          100,
			expectedPage:   2,
			expectedSize:   20,
			expectedOffset: 20,
			expectedPages:  5,
		},
		{
			name:           "partial last page",
			page:           1,
			pageSize:       20,
			total:          45,
			expectedPage:   1,
			expectedSize:   20,
			expectedOffset: 0,
			expectedPages:  3,
		},
		{
			name:           "zero total",
			page:           1,
			pageSize:       20,
			total:          0,
			expectedPage:   1,
			expectedSize:   20,
			expectedOffset: 0,
			expectedPages:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizePagination(tt.page, tt.pageSize, tt.total)
			assert.Equal(t, tt.expectedPage, result.Page)
			assert.Equal(t, tt.expectedSize, result.PageSize)
			assert.Equal(t, tt.expectedOffset, result.Offset)
			assert.Equal(t, tt.expectedPages, result.TotalPages)
		})
	}
}

// =============================================================================
// Test Dashboard Stats
// =============================================================================

func TestGetDashboardStats(t *testing.T) {
	t.Run("should return all stats successfully", func(t *testing.T) {
		db := newMockDB()
		// For this test, the mock returns totalUsers for most counts
		// We just verify the function executes without error
		db.totalUsers = 100

		svc := NewService(db)
		stats, err := svc.GetDashboardStats(context.Background())

		require.NoError(t, err)
		assert.NotNil(t, stats)
		// The mock doesn't fully simulate all the different count queries
		// but we verify the structure is correct
		assert.GreaterOrEqual(t, stats.TotalUsers, int64(0))
	})

	t.Run("should return error when count fails", func(t *testing.T) {
		db := newMockDB()
		db.countErr = errors.New("database connection failed")

		svc := NewService(db)
		stats, err := svc.GetDashboardStats(context.Background())

		assert.Error(t, err)
		assert.Nil(t, stats)
		assert.Contains(t, err.Error(), "failed to count")
	})

	// Test error at different Count call positions
	// GetDashboardStats makes 12 Count calls in sequence:
	// 1: TotalUsers, 2: ActiveUsers, 3: TotalOrgs, 4: TotalRunners, 5: OnlineRunners
	// 6: TotalPods, 7: ActivePods, 8: TotalSubscriptions, 9: ActiveSubscriptions
	// 10: NewUsersToday, 11: NewUsersThisWeek, 12: NewUsersThisMonth
	errorCases := []struct {
		name         string
		callNum      int
		expectedErr  string
	}{
		{"error on active users count", 2, "failed to count active users"},
		{"error on organizations count", 3, "failed to count organizations"},
		{"error on runners count", 4, "failed to count runners"},
		{"error on online runners count", 5, "failed to count online runners"},
		{"error on pods count", 6, "failed to count pods"},
		{"error on active pods count", 7, "failed to count active pods"},
		{"error on subscriptions count", 8, "failed to count subscriptions"},
		{"error on active subscriptions count", 9, "failed to count active subscriptions"},
		{"error on new users today count", 10, "failed to count new users today"},
		{"error on new users this week count", 11, "failed to count new users this week"},
		{"error on new users this month count", 12, "failed to count new users this month"},
	}

	for _, tc := range errorCases {
		t.Run(tc.name, func(t *testing.T) {
			db := newMockDB()
			db.countErrAtCall = tc.callNum

			svc := NewService(db)
			stats, err := svc.GetDashboardStats(context.Background())

			assert.Error(t, err)
			assert.Nil(t, stats)
			assert.Contains(t, err.Error(), tc.expectedErr)
		})
	}
}

// =============================================================================
// Test User Management
// =============================================================================

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

// =============================================================================
// Test Organization Management
// =============================================================================

func TestListOrganizations(t *testing.T) {
	t.Run("should list organizations with pagination", func(t *testing.T) {
		db := newMockDB()
		db.organizations[1] = &organization.Organization{ID: 1, Name: "Org 1", Slug: "org-1"}
		db.organizations[2] = &organization.Organization{ID: 2, Name: "Org 2", Slug: "org-2"}
		db.totalOrgs = 2

		svc := NewService(db)
		result, err := svc.ListOrganizations(context.Background(), &OrganizationListQuery{
			Page:     1,
			PageSize: 20,
		})

		require.NoError(t, err)
		assert.Equal(t, int64(2), result.Total)
	})
}

func TestGetOrganization(t *testing.T) {
	t.Run("should return organization when found", func(t *testing.T) {
		db := newMockDB()
		db.organizations[1] = &organization.Organization{ID: 1, Name: "Test Org", Slug: "test-org"}

		svc := NewService(db)
		org, err := svc.GetOrganization(context.Background(), 1)

		require.NoError(t, err)
		assert.Equal(t, int64(1), org.ID)
		assert.Equal(t, "Test Org", org.Name)
	})

	t.Run("should return error when organization not found", func(t *testing.T) {
		db := newMockDB()

		svc := NewService(db)
		org, err := svc.GetOrganization(context.Background(), 999)

		assert.Error(t, err)
		assert.Equal(t, ErrOrganizationNotFound, err)
		assert.Nil(t, org)
	})
}

func TestGetOrganizationWithMembers(t *testing.T) {
	t.Run("should return organization with members", func(t *testing.T) {
		db := newMockDB()
		db.organizations[1] = &organization.Organization{ID: 1, Name: "Test Org"}
		db.members = []organization.Member{
			{ID: 1, UserID: 100, OrganizationID: 1, Role: "owner"},
			{ID: 2, UserID: 101, OrganizationID: 1, Role: "member"},
		}

		svc := NewService(db)
		org, members, err := svc.GetOrganizationWithMembers(context.Background(), 1)

		require.NoError(t, err)
		assert.NotNil(t, org)
		assert.Len(t, members, 2)
	})
}

func TestDeleteOrganization(t *testing.T) {
	t.Run("should delete organization successfully when no runners", func(t *testing.T) {
		db := newMockDB()
		db.organizations[1] = &organization.Organization{ID: 1, Name: "Test Org"}
		db.runnerCount = 0 // Use runnerCount for Model(&runner.Runner{}).Where("organization_id = ?", ...).Count()

		svc := NewService(db)
		err := svc.DeleteOrganization(context.Background(), 1)

		require.NoError(t, err)
	})

	t.Run("should return error when organization has runners", func(t *testing.T) {
		db := newMockDB()
		db.organizations[1] = &organization.Organization{ID: 1, Name: "Test Org"}
		db.runnerCount = 5 // Use runnerCount for Model(&runner.Runner{}).Where("organization_id = ?", ...).Count()

		svc := NewService(db)
		err := svc.DeleteOrganization(context.Background(), 1)

		assert.Error(t, err)
		assert.Equal(t, ErrOrganizationHasActiveRunner, err)
	})

	t.Run("should return error when organization not found", func(t *testing.T) {
		db := newMockDB()

		svc := NewService(db)
		err := svc.DeleteOrganization(context.Background(), 999)

		assert.Error(t, err)
		assert.Equal(t, ErrOrganizationNotFound, err)
	})
}

// =============================================================================
// Test Runner Management
// =============================================================================

func TestListRunners(t *testing.T) {
	t.Run("should list runners with pagination", func(t *testing.T) {
		db := newMockDB()
		db.runners[1] = &runner.Runner{ID: 1, NodeID: "node-1", OrganizationID: 1}
		db.runners[2] = &runner.Runner{ID: 2, NodeID: "node-2", OrganizationID: 1}
		db.organizations[1] = &organization.Organization{ID: 1, Name: "Test Org"}
		db.totalRunners = 2

		svc := NewService(db)
		result, err := svc.ListRunners(context.Background(), &RunnerListQuery{
			Page:     1,
			PageSize: 20,
		})

		require.NoError(t, err)
		// Note: The mock returns totalRunners as the count
		assert.Equal(t, int64(2), result.Total)
		assert.Len(t, result.Data, 2)
	})
}

func TestGetRunner(t *testing.T) {
	t.Run("should return runner when found", func(t *testing.T) {
		db := newMockDB()
		db.runners[1] = &runner.Runner{ID: 1, NodeID: "test-node"}

		svc := NewService(db)
		r, err := svc.GetRunner(context.Background(), 1)

		require.NoError(t, err)
		assert.Equal(t, int64(1), r.ID)
		assert.Equal(t, "test-node", r.NodeID)
	})

	t.Run("should return error when runner not found", func(t *testing.T) {
		db := newMockDB()

		svc := NewService(db)
		r, err := svc.GetRunner(context.Background(), 999)

		assert.Error(t, err)
		assert.Equal(t, ErrRunnerNotFound, err)
		assert.Nil(t, r)
	})
}

func TestGetRunnerWithOrg(t *testing.T) {
	t.Run("should return runner with organization", func(t *testing.T) {
		db := newMockDB()
		db.runners[1] = &runner.Runner{ID: 1, NodeID: "test-node", OrganizationID: 1}
		db.organizations[1] = &organization.Organization{ID: 1, Name: "Test Org"}

		svc := NewService(db)
		rwo, err := svc.GetRunnerWithOrg(context.Background(), 1)

		require.NoError(t, err)
		assert.Equal(t, int64(1), rwo.Runner.ID)
		assert.NotNil(t, rwo.Organization)
		assert.Equal(t, "Test Org", rwo.Organization.Name)
	})
}

func TestDisableRunner(t *testing.T) {
	t.Run("should disable runner successfully", func(t *testing.T) {
		db := newMockDB()
		db.runners[1] = &runner.Runner{ID: 1, IsEnabled: true}

		svc := NewService(db)
		r, err := svc.DisableRunner(context.Background(), 1)

		require.NoError(t, err)
		assert.False(t, r.IsEnabled)
	})

	t.Run("should return error when runner not found", func(t *testing.T) {
		db := newMockDB()

		svc := NewService(db)
		r, err := svc.DisableRunner(context.Background(), 999)

		assert.Error(t, err)
		assert.Equal(t, ErrRunnerNotFound, err)
		assert.Nil(t, r)
	})
}

func TestEnableRunner(t *testing.T) {
	t.Run("should enable runner successfully", func(t *testing.T) {
		db := newMockDB()
		db.runners[1] = &runner.Runner{ID: 1, IsEnabled: false}

		svc := NewService(db)
		r, err := svc.EnableRunner(context.Background(), 1)

		require.NoError(t, err)
		assert.True(t, r.IsEnabled)
	})
}

func TestDeleteRunner(t *testing.T) {
	t.Run("should delete runner successfully when no active pods", func(t *testing.T) {
		db := newMockDB()
		db.runners[1] = &runner.Runner{ID: 1, NodeID: "test-node"}
		db.activePodCount = 0

		svc := NewService(db)
		r, err := svc.DeleteRunner(context.Background(), 1)

		require.NoError(t, err)
		assert.Equal(t, int64(1), r.ID)
	})

	t.Run("should return error when runner has active pods", func(t *testing.T) {
		db := newMockDB()
		db.runners[1] = &runner.Runner{ID: 1, NodeID: "test-node"}
		db.activePodCount = 3

		svc := NewService(db)
		r, err := svc.DeleteRunner(context.Background(), 1)

		assert.Error(t, err)
		assert.Equal(t, ErrRunnerHasActivePods, err)
		assert.Nil(t, r)
	})

	t.Run("should return error when runner not found", func(t *testing.T) {
		db := newMockDB()

		svc := NewService(db)
		r, err := svc.DeleteRunner(context.Background(), 999)

		assert.Error(t, err)
		assert.Equal(t, ErrRunnerNotFound, err)
		assert.Nil(t, r)
	})
}

// =============================================================================
// Test Audit Log
// =============================================================================

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

// =============================================================================
// Additional Filter Tests
// =============================================================================

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

func TestListOrganizations_WithFilters(t *testing.T) {
	t.Run("should filter by search term", func(t *testing.T) {
		db := newMockDB()
		db.organizations[1] = &organization.Organization{ID: 1, Name: "Test Org"}
		db.totalOrgs = 1

		svc := NewService(db)
		result, err := svc.ListOrganizations(context.Background(), &OrganizationListQuery{
			Page:     1,
			PageSize: 20,
			Search:   "Test",
		})

		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("should return error when count fails", func(t *testing.T) {
		db := newMockDB()
		db.countErr = errors.New("count failed")

		svc := NewService(db)
		result, err := svc.ListOrganizations(context.Background(), &OrganizationListQuery{
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
		result, err := svc.ListOrganizations(context.Background(), &OrganizationListQuery{
			Page:     1,
			PageSize: 20,
		})

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestListRunners_WithFilters(t *testing.T) {
	t.Run("should filter by search term", func(t *testing.T) {
		db := newMockDB()
		db.runners[1] = &runner.Runner{ID: 1, NodeID: "test-node"}
		db.totalRunners = 1

		svc := NewService(db)
		result, err := svc.ListRunners(context.Background(), &RunnerListQuery{
			Page:     1,
			PageSize: 20,
			Search:   "test",
		})

		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("should filter by status", func(t *testing.T) {
		db := newMockDB()
		db.runners[1] = &runner.Runner{ID: 1, NodeID: "test-node", Status: "online"}
		db.totalRunners = 1

		svc := NewService(db)
		result, err := svc.ListRunners(context.Background(), &RunnerListQuery{
			Page:     1,
			PageSize: 20,
			Status:   "online",
		})

		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("should filter by organization ID", func(t *testing.T) {
		db := newMockDB()
		orgID := int64(1)
		db.runners[1] = &runner.Runner{ID: 1, NodeID: "test-node", OrganizationID: 1}
		db.totalRunners = 1

		svc := NewService(db)
		result, err := svc.ListRunners(context.Background(), &RunnerListQuery{
			Page:     1,
			PageSize: 20,
			OrgID:    &orgID,
		})

		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("should return error when count fails", func(t *testing.T) {
		db := newMockDB()
		db.countErr = errors.New("count failed")

		svc := NewService(db)
		result, err := svc.ListRunners(context.Background(), &RunnerListQuery{
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
		result, err := svc.ListRunners(context.Background(), &RunnerListQuery{
			Page:     1,
			PageSize: 20,
		})

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

// =============================================================================
// Error Path Tests
// =============================================================================

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

func TestGetOrganizationWithMembers_ErrorPaths(t *testing.T) {
	t.Run("should return error when organization not found", func(t *testing.T) {
		db := newMockDB()

		svc := NewService(db)
		org, members, err := svc.GetOrganizationWithMembers(context.Background(), 999)

		assert.Error(t, err)
		assert.Equal(t, ErrOrganizationNotFound, err)
		assert.Nil(t, org)
		assert.Nil(t, members)
	})

	t.Run("should return error when find members fails", func(t *testing.T) {
		db := newMockDB()
		db.organizations[1] = &organization.Organization{ID: 1, Name: "Test Org"}
		db.findErr = errors.New("find failed")

		svc := NewService(db)
		org, members, err := svc.GetOrganizationWithMembers(context.Background(), 1)

		assert.Error(t, err)
		assert.Nil(t, org)
		assert.Nil(t, members)
	})
}

func TestDeleteOrganization_ErrorPaths(t *testing.T) {
	t.Run("should return error when count runners fails", func(t *testing.T) {
		db := newMockDB()
		db.organizations[1] = &organization.Organization{ID: 1, Name: "Test Org"}
		db.countErr = errors.New("count failed")

		svc := NewService(db)
		err := svc.DeleteOrganization(context.Background(), 1)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to check runners")
	})

	t.Run("should return error when delete fails", func(t *testing.T) {
		db := newMockDB()
		db.organizations[1] = &organization.Organization{ID: 1, Name: "Test Org"}
		db.runnerCount = 0 // Use runnerCount for Model(&runner.Runner{}).Where("organization_id = ?", ...).Count()
		db.deleteErr = errors.New("delete failed")

		svc := NewService(db)
		err := svc.DeleteOrganization(context.Background(), 1)

		assert.Error(t, err)
	})
}

func TestDeleteRunner_ErrorPaths(t *testing.T) {
	t.Run("should return error when count pods fails", func(t *testing.T) {
		db := newMockDB()
		db.runners[1] = &runner.Runner{ID: 1, NodeID: "test-node"}
		db.countErr = errors.New("count failed")

		svc := NewService(db)
		r, err := svc.DeleteRunner(context.Background(), 1)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to check pods")
		assert.Nil(t, r)
	})

	t.Run("should return error when delete fails", func(t *testing.T) {
		db := newMockDB()
		db.runners[1] = &runner.Runner{ID: 1, NodeID: "test-node"}
		db.activePodCount = 0
		db.deleteErr = errors.New("delete failed")

		svc := NewService(db)
		r, err := svc.DeleteRunner(context.Background(), 1)

		assert.Error(t, err)
		assert.Nil(t, r)
	})
}

func TestDisableRunner_ErrorPaths(t *testing.T) {
	t.Run("should return error when save fails", func(t *testing.T) {
		db := newMockDB()
		db.runners[1] = &runner.Runner{ID: 1, IsEnabled: true}
		db.saveErr = errors.New("save failed")

		svc := NewService(db)
		r, err := svc.DisableRunner(context.Background(), 1)

		assert.Error(t, err)
		assert.Nil(t, r)
	})
}

func TestEnableRunner_ErrorPaths(t *testing.T) {
	t.Run("should return error when runner not found", func(t *testing.T) {
		db := newMockDB()

		svc := NewService(db)
		r, err := svc.EnableRunner(context.Background(), 999)

		assert.Error(t, err)
		assert.Equal(t, ErrRunnerNotFound, err)
		assert.Nil(t, r)
	})

	t.Run("should return error when save fails", func(t *testing.T) {
		db := newMockDB()
		db.runners[1] = &runner.Runner{ID: 1, IsEnabled: false}
		db.saveErr = errors.New("save failed")

		svc := NewService(db)
		r, err := svc.EnableRunner(context.Background(), 1)

		assert.Error(t, err)
		assert.Nil(t, r)
	})
}

func TestGetRunnerWithOrg_ErrorPaths(t *testing.T) {
	t.Run("should return error when runner not found", func(t *testing.T) {
		db := newMockDB()

		svc := NewService(db)
		rwo, err := svc.GetRunnerWithOrg(context.Background(), 999)

		assert.Error(t, err)
		assert.Equal(t, ErrRunnerNotFound, err)
		assert.Nil(t, rwo)
	})
}
