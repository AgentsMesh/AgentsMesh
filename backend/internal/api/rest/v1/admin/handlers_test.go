package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/admin"
	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/domain/organization"
	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
	"github.com/anthropics/agentsmesh/backend/internal/infra/database"
	adminservice "github.com/anthropics/agentsmesh/backend/internal/service/admin"
	"github.com/anthropics/agentsmesh/backend/internal/service/auth"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// mockHandlerDB implements database.DB interface for handler testing
type mockHandlerDB struct {
	users         map[int64]*user.User
	organizations map[int64]*organization.Organization
	runners       map[int64]*runner.Runner
	members       []organization.Member
	auditLogs     []admin.AuditLog

	// For count queries
	totalCount     int64
	runnerCount    int64
	activePodCount int64

	// Control behavior
	createErr  error
	firstErr   error
	findErr    error
	saveErr    error
	deleteErr  error
	updatesErr error
	countErr   error

	// Track calls
	lastTable string
	lastModel interface{}
	lastWhere interface{}
}

func newMockHandlerDB() *mockHandlerDB {
	return &mockHandlerDB{
		users:         make(map[int64]*user.User),
		organizations: make(map[int64]*organization.Organization),
		runners:       make(map[int64]*runner.Runner),
	}
}

func (m *mockHandlerDB) Transaction(fc func(tx database.DB) error) error {
	return fc(m)
}

func (m *mockHandlerDB) WithContext(ctx context.Context) database.DB {
	return m
}

func (m *mockHandlerDB) Create(value interface{}) error {
	if m.createErr != nil {
		return m.createErr
	}
	if log, ok := value.(*admin.AuditLog); ok {
		m.auditLogs = append(m.auditLogs, *log)
	}
	return nil
}

func (m *mockHandlerDB) First(dest interface{}, conds ...interface{}) error {
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

func (m *mockHandlerDB) Find(dest interface{}, conds ...interface{}) error {
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

func (m *mockHandlerDB) Save(value interface{}) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	if r, ok := value.(*runner.Runner); ok {
		m.runners[r.ID] = r
	}
	return nil
}

func (m *mockHandlerDB) Delete(value interface{}, conds ...interface{}) error {
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

func (m *mockHandlerDB) Updates(model interface{}, values interface{}) error {
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

func (m *mockHandlerDB) Model(value interface{}) database.DB {
	m.lastModel = value
	// Set lastTable based on model type for proper Count behavior
	switch value.(type) {
	case *agentpod.Pod:
		m.lastTable = "agent_pods"
	case *runner.Runner:
		m.lastTable = "runners"
	default:
		m.lastTable = ""
	}
	return m
}

func (m *mockHandlerDB) Table(name string) database.DB {
	m.lastTable = name
	return m
}

func (m *mockHandlerDB) Where(query interface{}, args ...interface{}) database.DB {
	m.lastWhere = query
	return m
}

func (m *mockHandlerDB) Select(query interface{}, args ...interface{}) database.DB {
	return m
}

func (m *mockHandlerDB) Joins(query string, args ...interface{}) database.DB {
	return m
}

func (m *mockHandlerDB) Preload(query string, args ...interface{}) database.DB {
	return m
}

func (m *mockHandlerDB) Order(value interface{}) database.DB {
	return m
}

func (m *mockHandlerDB) Limit(limit int) database.DB {
	return m
}

func (m *mockHandlerDB) Offset(offset int) database.DB {
	return m
}

func (m *mockHandlerDB) Group(name string) database.DB {
	return m
}

func (m *mockHandlerDB) Count(count *int64) error {
	if m.countErr != nil {
		return m.countErr
	}

	// Check model type first (for Model().Where().Count() pattern)
	switch m.lastModel.(type) {
	case *runner.Runner:
		*count = m.runnerCount
		return nil
	}

	// Fallback to table name (for Table().Where().Count() pattern)
	switch m.lastTable {
	case "runners":
		*count = m.runnerCount
	case "agent_pods":
		*count = m.activePodCount
	default:
		*count = m.totalCount
	}
	return nil
}

func (m *mockHandlerDB) Scan(dest interface{}) error {
	return nil
}

func (m *mockHandlerDB) GormDB() *gorm.DB {
	return nil
}

var _ database.DB = (*mockHandlerDB)(nil)

// Helper function to create test context with admin user
func createAdminContext(w *httptest.ResponseRecorder) *gin.Context {
	c, _ := gin.CreateTestContext(w)
	c.Set("admin_user_id", int64(1))
	c.Set("admin_user", &user.User{ID: 1, Email: "admin@example.com", IsSystemAdmin: true})
	return c
}

// =============================================================================
// User Handler Tests
// =============================================================================

func TestUserHandler_ListUsers(t *testing.T) {
	t.Run("should list users successfully", func(t *testing.T) {
		db := newMockHandlerDB()
		db.users[1] = &user.User{ID: 1, Email: "user1@example.com", IsActive: true}
		db.users[2] = &user.User{ID: 2, Email: "user2@example.com", IsActive: true}
		db.totalCount = 2

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/users?page=1&page_size=20", nil)

		handler.ListUsers(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, float64(2), response["total"])
	})
}

func TestUserHandler_GetUser(t *testing.T) {
	t.Run("should get user successfully", func(t *testing.T) {
		db := newMockHandlerDB()
		testName := "Test User"
		db.users[1] = &user.User{ID: 1, Email: "test@example.com", Name: &testName}

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/users/1", nil)
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.GetUser(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "test@example.com", response["email"])
	})

	t.Run("should return 404 when user not found", func(t *testing.T) {
		db := newMockHandlerDB()

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/users/999", nil)
		c.Params = gin.Params{{Key: "id", Value: "999"}}

		handler.GetUser(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("should return 400 for invalid user ID", func(t *testing.T) {
		db := newMockHandlerDB()

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/users/invalid", nil)
		c.Params = gin.Params{{Key: "id", Value: "invalid"}}

		handler.GetUser(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestUserHandler_DisableUser(t *testing.T) {
	t.Run("should disable user successfully", func(t *testing.T) {
		db := newMockHandlerDB()
		db.users[2] = &user.User{ID: 2, Email: "user@example.com", IsActive: true}

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("POST", "/users/2/disable", nil)
		c.Params = gin.Params{{Key: "id", Value: "2"}}

		handler.DisableUser(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("should prevent disabling self", func(t *testing.T) {
		db := newMockHandlerDB()
		db.users[1] = &user.User{ID: 1, Email: "admin@example.com", IsActive: true}

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w) // admin_user_id is 1
		c.Request = httptest.NewRequest("POST", "/users/1/disable", nil)
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.DisableUser(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestUserHandler_RevokeAdmin(t *testing.T) {
	t.Run("should revoke admin successfully", func(t *testing.T) {
		db := newMockHandlerDB()
		db.users[2] = &user.User{ID: 2, Email: "other@example.com", IsSystemAdmin: true}

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("POST", "/users/2/revoke-admin", nil)
		c.Params = gin.Params{{Key: "id", Value: "2"}}

		handler.RevokeAdmin(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("should prevent revoking own admin", func(t *testing.T) {
		db := newMockHandlerDB()
		db.users[1] = &user.User{ID: 1, Email: "admin@example.com", IsSystemAdmin: true}

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("POST", "/users/1/revoke-admin", nil)
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.RevokeAdmin(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestUserHandler_UpdateUser(t *testing.T) {
	t.Run("should update user successfully", func(t *testing.T) {
		db := newMockHandlerDB()
		oldName := "Old Name"
		db.users[1] = &user.User{ID: 1, Email: "old@example.com", Name: &oldName}

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		body := bytes.NewBufferString(`{"name": "New Name"}`)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("PUT", "/users/1", body)
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.UpdateUser(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("should return 400 for empty updates", func(t *testing.T) {
		db := newMockHandlerDB()
		db.users[1] = &user.User{ID: 1, Email: "test@example.com"}

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		body := bytes.NewBufferString(`{}`)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("PUT", "/users/1", body)
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.UpdateUser(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// =============================================================================
// Organization Handler Tests
// =============================================================================

func TestOrganizationHandler_ListOrganizations(t *testing.T) {
	t.Run("should list organizations successfully", func(t *testing.T) {
		db := newMockHandlerDB()
		db.organizations[1] = &organization.Organization{ID: 1, Name: "Org 1", Slug: "org-1"}
		db.totalCount = 1

		svc := adminservice.NewService(db)
		handler := NewOrganizationHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/organizations", nil)

		handler.ListOrganizations(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestOrganizationHandler_DeleteOrganization(t *testing.T) {
	t.Run("should delete organization successfully", func(t *testing.T) {
		db := newMockHandlerDB()
		db.organizations[1] = &organization.Organization{ID: 1, Name: "Test Org"}
		db.runnerCount = 0

		svc := adminservice.NewService(db)
		handler := NewOrganizationHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("DELETE", "/organizations/1", nil)
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.DeleteOrganization(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("should return 409 when organization has runners", func(t *testing.T) {
		db := newMockHandlerDB()
		db.organizations[1] = &organization.Organization{ID: 1, Name: "Test Org"}
		db.runnerCount = 5

		svc := adminservice.NewService(db)
		handler := NewOrganizationHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("DELETE", "/organizations/1", nil)
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.DeleteOrganization(c)

		assert.Equal(t, http.StatusConflict, w.Code)
	})
}

// =============================================================================
// Runner Handler Tests
// =============================================================================

func TestRunnerHandler_ListRunners(t *testing.T) {
	t.Run("should list runners successfully", func(t *testing.T) {
		db := newMockHandlerDB()
		db.runners[1] = &runner.Runner{ID: 1, NodeID: "node-1", OrganizationID: 1}
		db.organizations[1] = &organization.Organization{ID: 1, Name: "Org 1"}
		db.totalCount = 1

		svc := adminservice.NewService(db)
		handler := NewRunnerHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/runners", nil)

		handler.ListRunners(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestRunnerHandler_DisableRunner(t *testing.T) {
	t.Run("should disable runner successfully", func(t *testing.T) {
		db := newMockHandlerDB()
		db.runners[1] = &runner.Runner{ID: 1, NodeID: "test-node", IsEnabled: true}

		svc := adminservice.NewService(db)
		handler := NewRunnerHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("POST", "/runners/1/disable", nil)
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.DisableRunner(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("should return 404 when runner not found", func(t *testing.T) {
		db := newMockHandlerDB()

		svc := adminservice.NewService(db)
		handler := NewRunnerHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("POST", "/runners/999/disable", nil)
		c.Params = gin.Params{{Key: "id", Value: "999"}}

		handler.DisableRunner(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestRunnerHandler_DeleteRunner(t *testing.T) {
	t.Run("should delete runner successfully", func(t *testing.T) {
		db := newMockHandlerDB()
		db.runners[1] = &runner.Runner{ID: 1, NodeID: "test-node"}
		db.activePodCount = 0

		svc := adminservice.NewService(db)
		handler := NewRunnerHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("DELETE", "/runners/1", nil)
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.DeleteRunner(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("should return 409 when runner has active pods", func(t *testing.T) {
		db := newMockHandlerDB()
		db.runners[1] = &runner.Runner{ID: 1, NodeID: "test-node"}
		db.activePodCount = 3

		svc := adminservice.NewService(db)
		handler := NewRunnerHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("DELETE", "/runners/1", nil)
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.DeleteRunner(c)

		assert.Equal(t, http.StatusConflict, w.Code)
	})
}

// =============================================================================
// Additional User Handler Tests
// =============================================================================

func TestUserHandler_EnableUser(t *testing.T) {
	t.Run("should enable user successfully", func(t *testing.T) {
		db := newMockHandlerDB()
		db.users[2] = &user.User{ID: 2, Email: "user@example.com", IsActive: false}

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("POST", "/users/2/enable", nil)
		c.Params = gin.Params{{Key: "id", Value: "2"}}

		handler.EnableUser(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestUserHandler_GrantAdmin(t *testing.T) {
	t.Run("should grant admin successfully", func(t *testing.T) {
		db := newMockHandlerDB()
		db.users[2] = &user.User{ID: 2, Email: "user@example.com", IsSystemAdmin: false}

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("POST", "/users/2/grant-admin", nil)
		c.Params = gin.Params{{Key: "id", Value: "2"}}

		handler.GrantAdmin(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// =============================================================================
// Additional Organization Handler Tests
// =============================================================================

func TestOrganizationHandler_GetOrganization(t *testing.T) {
	t.Run("should get organization successfully", func(t *testing.T) {
		db := newMockHandlerDB()
		db.organizations[1] = &organization.Organization{ID: 1, Name: "Test Org", Slug: "test-org"}

		svc := adminservice.NewService(db)
		handler := NewOrganizationHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/organizations/1", nil)
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.GetOrganization(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("should return 404 when organization not found", func(t *testing.T) {
		db := newMockHandlerDB()

		svc := adminservice.NewService(db)
		handler := NewOrganizationHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/organizations/999", nil)
		c.Params = gin.Params{{Key: "id", Value: "999"}}

		handler.GetOrganization(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("should return 400 for invalid ID", func(t *testing.T) {
		db := newMockHandlerDB()

		svc := adminservice.NewService(db)
		handler := NewOrganizationHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/organizations/invalid", nil)
		c.Params = gin.Params{{Key: "id", Value: "invalid"}}

		handler.GetOrganization(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestOrganizationHandler_GetOrganizationMembers(t *testing.T) {
	t.Run("should get organization members successfully", func(t *testing.T) {
		db := newMockHandlerDB()
		db.organizations[1] = &organization.Organization{ID: 1, Name: "Test Org", Slug: "test-org"}
		db.members = []organization.Member{
			{ID: 1, UserID: 1, OrganizationID: 1, Role: "owner"},
		}

		svc := adminservice.NewService(db)
		handler := NewOrganizationHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/organizations/1/members", nil)
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.GetOrganizationMembers(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// =============================================================================
// Additional Runner Handler Tests
// =============================================================================

func TestRunnerHandler_GetRunner(t *testing.T) {
	t.Run("should get runner successfully", func(t *testing.T) {
		db := newMockHandlerDB()
		db.runners[1] = &runner.Runner{ID: 1, NodeID: "test-node", OrganizationID: 1}
		db.organizations[1] = &organization.Organization{ID: 1, Name: "Test Org"}

		svc := adminservice.NewService(db)
		handler := NewRunnerHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/runners/1", nil)
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.GetRunner(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("should return 404 when runner not found", func(t *testing.T) {
		db := newMockHandlerDB()

		svc := adminservice.NewService(db)
		handler := NewRunnerHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/runners/999", nil)
		c.Params = gin.Params{{Key: "id", Value: "999"}}

		handler.GetRunner(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestRunnerHandler_EnableRunner(t *testing.T) {
	t.Run("should enable runner successfully", func(t *testing.T) {
		db := newMockHandlerDB()
		db.runners[1] = &runner.Runner{ID: 1, NodeID: "test-node", IsEnabled: false}

		svc := adminservice.NewService(db)
		handler := NewRunnerHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("POST", "/runners/1/enable", nil)
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.EnableRunner(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// =============================================================================
// Dashboard Handler Tests
// =============================================================================

func TestDashboardHandler_GetStats(t *testing.T) {
	t.Run("should get stats successfully", func(t *testing.T) {
		db := newMockHandlerDB()
		db.totalCount = 10

		svc := adminservice.NewService(db)
		handler := NewDashboardHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/dashboard/stats", nil)

		handler.GetStats(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// =============================================================================
// Audit Log Handler Tests
// =============================================================================

func TestAuditLogHandler_ListAuditLogs(t *testing.T) {
	t.Run("should list audit logs successfully", func(t *testing.T) {
		db := newMockHandlerDB()
		db.auditLogs = []admin.AuditLog{
			{ID: 1, AdminUserID: 1, Action: admin.AuditActionUserView},
		}
		db.totalCount = 1

		svc := adminservice.NewService(db)
		handler := NewAuditLogHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/audit-logs", nil)

		handler.ListAuditLogs(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("should filter by admin_user_id", func(t *testing.T) {
		db := newMockHandlerDB()
		db.totalCount = 0

		svc := adminservice.NewService(db)
		handler := NewAuditLogHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/audit-logs?admin_user_id=1", nil)

		handler.ListAuditLogs(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("should filter by action", func(t *testing.T) {
		db := newMockHandlerDB()
		db.totalCount = 0

		svc := adminservice.NewService(db)
		handler := NewAuditLogHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/audit-logs?action=user.view", nil)

		handler.ListAuditLogs(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("should filter by target_type and target_id", func(t *testing.T) {
		db := newMockHandlerDB()
		db.totalCount = 0

		svc := adminservice.NewService(db)
		handler := NewAuditLogHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/audit-logs?target_type=user&target_id=1", nil)

		handler.ListAuditLogs(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("should filter by date range", func(t *testing.T) {
		db := newMockHandlerDB()
		db.totalCount = 0

		svc := adminservice.NewService(db)
		handler := NewAuditLogHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/audit-logs?start_time=2024-01-01T00:00:00Z&end_time=2024-12-31T23:59:59Z", nil)

		handler.ListAuditLogs(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("should return 500 when service fails", func(t *testing.T) {
		db := newMockHandlerDB()
		db.countErr = gorm.ErrInvalidDB

		svc := adminservice.NewService(db)
		handler := NewAuditLogHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/audit-logs", nil)

		handler.ListAuditLogs(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("should include audit log with old_data and new_data", func(t *testing.T) {
		db := newMockHandlerDB()
		oldData := `{"is_active": true}`
		newData := `{"is_active": false}`
		testUser := &user.User{ID: 1, Email: "admin@example.com"}
		db.auditLogs = []admin.AuditLog{
			{
				ID:          1,
				AdminUserID: 1,
				Action:      admin.AuditActionUserDisable,
				OldData:     &oldData,
				NewData:     &newData,
				AdminUser:   testUser,
			},
		}
		db.totalCount = 1

		svc := adminservice.NewService(db)
		handler := NewAuditLogHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/audit-logs", nil)

		handler.ListAuditLogs(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// =============================================================================
// Audit Helper Tests
// =============================================================================

func TestLogAdminAction(t *testing.T) {
	t.Run("should log action successfully", func(t *testing.T) {
		db := newMockHandlerDB()
		svc := adminservice.NewService(db)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)
		c.Request.Header.Set("User-Agent", "Test Agent")

		LogAdminAction(c, svc, admin.AuditActionUserView, admin.TargetTypeUser, 1, nil, nil)

		assert.Len(t, db.auditLogs, 1)
		assert.Equal(t, admin.AuditActionUserView, db.auditLogs[0].Action)
	})

	t.Run("should handle missing admin user ID", func(t *testing.T) {
		db := newMockHandlerDB()
		svc := adminservice.NewService(db)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)
		// No admin_user_id set

		// Should not panic
		LogAdminAction(c, svc, admin.AuditActionUserView, admin.TargetTypeUser, 1, nil, nil)

		// No log should be created
		assert.Len(t, db.auditLogs, 0)
	})
}

// =============================================================================
// Error Path Tests - Users
// =============================================================================

func TestUserHandler_ListUsers_WithFilters(t *testing.T) {
	t.Run("should filter by is_active", func(t *testing.T) {
		db := newMockHandlerDB()
		db.users[1] = &user.User{ID: 1, Email: "active@example.com", IsActive: true}
		db.totalCount = 1

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/users?is_active=true", nil)

		handler.ListUsers(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("should filter by is_admin", func(t *testing.T) {
		db := newMockHandlerDB()
		db.users[1] = &user.User{ID: 1, Email: "admin@example.com", IsSystemAdmin: true}
		db.totalCount = 1

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/users?is_admin=true", nil)

		handler.ListUsers(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestUserHandler_DisableUser_NotFound(t *testing.T) {
	t.Run("should return 404 when user not found", func(t *testing.T) {
		db := newMockHandlerDB()

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("POST", "/users/999/disable", nil)
		c.Params = gin.Params{{Key: "id", Value: "999"}}

		handler.DisableUser(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestUserHandler_EnableUser_NotFound(t *testing.T) {
	t.Run("should return 404 when user not found", func(t *testing.T) {
		db := newMockHandlerDB()

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("POST", "/users/999/enable", nil)
		c.Params = gin.Params{{Key: "id", Value: "999"}}

		handler.EnableUser(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestUserHandler_GrantAdmin_NotFound(t *testing.T) {
	t.Run("should return 404 when user not found", func(t *testing.T) {
		db := newMockHandlerDB()

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("POST", "/users/999/grant-admin", nil)
		c.Params = gin.Params{{Key: "id", Value: "999"}}

		handler.GrantAdmin(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestUserHandler_RevokeAdmin_NotFound(t *testing.T) {
	t.Run("should return 404 when user not found", func(t *testing.T) {
		db := newMockHandlerDB()

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("POST", "/users/999/revoke-admin", nil)
		c.Params = gin.Params{{Key: "id", Value: "999"}}

		handler.RevokeAdmin(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestUserHandler_UpdateUser_NotFound(t *testing.T) {
	t.Run("should return 404 when user not found", func(t *testing.T) {
		db := newMockHandlerDB()

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		body := bytes.NewBufferString(`{"name": "New Name"}`)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("PUT", "/users/999", body)
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "id", Value: "999"}}

		handler.UpdateUser(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

// =============================================================================
// Error Path Tests - Organizations
// =============================================================================

func TestOrganizationHandler_GetOrganizationMembers_NotFound(t *testing.T) {
	t.Run("should return 404 when organization not found", func(t *testing.T) {
		db := newMockHandlerDB()

		svc := adminservice.NewService(db)
		handler := NewOrganizationHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/organizations/999/members", nil)
		c.Params = gin.Params{{Key: "id", Value: "999"}}

		handler.GetOrganizationMembers(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("should return 400 for invalid ID", func(t *testing.T) {
		db := newMockHandlerDB()

		svc := adminservice.NewService(db)
		handler := NewOrganizationHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/organizations/invalid/members", nil)
		c.Params = gin.Params{{Key: "id", Value: "invalid"}}

		handler.GetOrganizationMembers(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestOrganizationHandler_DeleteOrganization_NotFound(t *testing.T) {
	t.Run("should return 404 when organization not found", func(t *testing.T) {
		db := newMockHandlerDB()

		svc := adminservice.NewService(db)
		handler := NewOrganizationHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("DELETE", "/organizations/999", nil)
		c.Params = gin.Params{{Key: "id", Value: "999"}}

		handler.DeleteOrganization(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("should return 400 for invalid ID", func(t *testing.T) {
		db := newMockHandlerDB()

		svc := adminservice.NewService(db)
		handler := NewOrganizationHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("DELETE", "/organizations/invalid", nil)
		c.Params = gin.Params{{Key: "id", Value: "invalid"}}

		handler.DeleteOrganization(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// =============================================================================
// Error Path Tests - Runners
// =============================================================================

func TestRunnerHandler_DisableRunner_InvalidID(t *testing.T) {
	t.Run("should return 400 for invalid ID", func(t *testing.T) {
		db := newMockHandlerDB()

		svc := adminservice.NewService(db)
		handler := NewRunnerHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("POST", "/runners/invalid/disable", nil)
		c.Params = gin.Params{{Key: "id", Value: "invalid"}}

		handler.DisableRunner(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestRunnerHandler_EnableRunner_NotFound(t *testing.T) {
	t.Run("should return 404 when runner not found", func(t *testing.T) {
		db := newMockHandlerDB()

		svc := adminservice.NewService(db)
		handler := NewRunnerHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("POST", "/runners/999/enable", nil)
		c.Params = gin.Params{{Key: "id", Value: "999"}}

		handler.EnableRunner(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("should return 400 for invalid ID", func(t *testing.T) {
		db := newMockHandlerDB()

		svc := adminservice.NewService(db)
		handler := NewRunnerHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("POST", "/runners/invalid/enable", nil)
		c.Params = gin.Params{{Key: "id", Value: "invalid"}}

		handler.EnableRunner(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestRunnerHandler_DeleteRunner_NotFound(t *testing.T) {
	t.Run("should return 404 when runner not found", func(t *testing.T) {
		db := newMockHandlerDB()

		svc := adminservice.NewService(db)
		handler := NewRunnerHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("DELETE", "/runners/999", nil)
		c.Params = gin.Params{{Key: "id", Value: "999"}}

		handler.DeleteRunner(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("should return 400 for invalid ID", func(t *testing.T) {
		db := newMockHandlerDB()

		svc := adminservice.NewService(db)
		handler := NewRunnerHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("DELETE", "/runners/invalid", nil)
		c.Params = gin.Params{{Key: "id", Value: "invalid"}}

		handler.DeleteRunner(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestRunnerHandler_GetRunner_InvalidID(t *testing.T) {
	t.Run("should return 400 for invalid ID", func(t *testing.T) {
		db := newMockHandlerDB()

		svc := adminservice.NewService(db)
		handler := NewRunnerHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/runners/invalid", nil)
		c.Params = gin.Params{{Key: "id", Value: "invalid"}}

		handler.GetRunner(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// =============================================================================
// Auth Handler Tests
// =============================================================================

func TestAuthHandler_GetMe(t *testing.T) {
	t.Run("should return admin user info", func(t *testing.T) {
		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/me", nil)

		handler := NewAuthHandler(nil, nil)
		handler.GetMe(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("should return 401 when admin user not in context", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/me", nil)
		// No admin_user set

		handler := NewAuthHandler(nil, nil)
		handler.GetMe(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// =============================================================================
// Internal Error Tests - Handler Layer
// =============================================================================

func TestDashboardHandler_GetStats_Error(t *testing.T) {
	t.Run("should return 500 when service fails", func(t *testing.T) {
		db := newMockHandlerDB()
		db.countErr = gorm.ErrInvalidDB

		svc := adminservice.NewService(db)
		handler := NewDashboardHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/dashboard/stats", nil)

		handler.GetStats(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestUserHandler_ListUsers_Error(t *testing.T) {
	t.Run("should return 500 when service fails", func(t *testing.T) {
		db := newMockHandlerDB()
		db.countErr = gorm.ErrInvalidDB

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/users", nil)

		handler.ListUsers(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestUserHandler_GetUser_InternalError(t *testing.T) {
	t.Run("should return 500 when service fails with internal error", func(t *testing.T) {
		db := newMockHandlerDB()
		db.firstErr = gorm.ErrInvalidDB

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/users/1", nil)
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.GetUser(c)

		// When firstErr is set, it returns ErrUserNotFound which maps to 404
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestUserHandler_UpdateUser_InvalidID(t *testing.T) {
	t.Run("should return 400 for invalid user ID", func(t *testing.T) {
		db := newMockHandlerDB()

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		body := bytes.NewBufferString(`{"name": "New Name"}`)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("PUT", "/users/invalid", body)
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "id", Value: "invalid"}}

		handler.UpdateUser(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("should return 400 for invalid JSON body", func(t *testing.T) {
		db := newMockHandlerDB()
		db.users[1] = &user.User{ID: 1, Email: "test@example.com"}

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		body := bytes.NewBufferString(`{invalid json}`)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("PUT", "/users/1", body)
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.UpdateUser(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestUserHandler_DisableUser_InvalidID(t *testing.T) {
	t.Run("should return 400 for invalid user ID", func(t *testing.T) {
		db := newMockHandlerDB()

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("POST", "/users/invalid/disable", nil)
		c.Params = gin.Params{{Key: "id", Value: "invalid"}}

		handler.DisableUser(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestUserHandler_EnableUser_InvalidID(t *testing.T) {
	t.Run("should return 400 for invalid user ID", func(t *testing.T) {
		db := newMockHandlerDB()

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("POST", "/users/invalid/enable", nil)
		c.Params = gin.Params{{Key: "id", Value: "invalid"}}

		handler.EnableUser(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestUserHandler_GrantAdmin_InvalidID(t *testing.T) {
	t.Run("should return 400 for invalid user ID", func(t *testing.T) {
		db := newMockHandlerDB()

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("POST", "/users/invalid/grant-admin", nil)
		c.Params = gin.Params{{Key: "id", Value: "invalid"}}

		handler.GrantAdmin(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestUserHandler_RevokeAdmin_InvalidID(t *testing.T) {
	t.Run("should return 400 for invalid user ID", func(t *testing.T) {
		db := newMockHandlerDB()

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("POST", "/users/invalid/revoke-admin", nil)
		c.Params = gin.Params{{Key: "id", Value: "invalid"}}

		handler.RevokeAdmin(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestOrganizationHandler_ListOrganizations_Error(t *testing.T) {
	t.Run("should return 500 when service fails", func(t *testing.T) {
		db := newMockHandlerDB()
		db.countErr = gorm.ErrInvalidDB

		svc := adminservice.NewService(db)
		handler := NewOrganizationHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/organizations", nil)

		handler.ListOrganizations(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestRunnerHandler_ListRunners_Error(t *testing.T) {
	t.Run("should return 500 when service fails", func(t *testing.T) {
		db := newMockHandlerDB()
		db.countErr = gorm.ErrInvalidDB

		svc := adminservice.NewService(db)
		handler := NewRunnerHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/runners", nil)

		handler.ListRunners(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestOrganizationHandler_GetOrganization_InternalError(t *testing.T) {
	t.Run("should handle internal error scenario", func(t *testing.T) {
		db := newMockHandlerDB()
		// Setting firstErr causes GetOrganization to return ErrOrganizationNotFound which maps to 404
		db.firstErr = gorm.ErrInvalidDB

		svc := adminservice.NewService(db)
		handler := NewOrganizationHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/organizations/1", nil)
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.GetOrganization(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestOrganizationHandler_ListOrganizations_WithFilters(t *testing.T) {
	t.Run("should filter by search term", func(t *testing.T) {
		db := newMockHandlerDB()
		db.organizations[1] = &organization.Organization{ID: 1, Name: "Test Org"}
		db.totalCount = 1

		svc := adminservice.NewService(db)
		handler := NewOrganizationHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/organizations?search=test&page=1&page_size=10", nil)

		handler.ListOrganizations(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestRunnerHandler_ListRunners_WithFilters(t *testing.T) {
	t.Run("should filter by search and status", func(t *testing.T) {
		db := newMockHandlerDB()
		db.runners[1] = &runner.Runner{ID: 1, NodeID: "test-node", Status: "online"}
		db.totalCount = 1

		svc := adminservice.NewService(db)
		handler := NewRunnerHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/runners?search=test&status=online&org_id=1&page=1&page_size=10", nil)

		handler.ListRunners(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// Test member response with User field populated
func TestOrganizationHandler_GetOrganizationMembers_WithUser(t *testing.T) {
	t.Run("should return members with user info", func(t *testing.T) {
		db := newMockHandlerDB()
		db.organizations[1] = &organization.Organization{ID: 1, Name: "Test Org"}
		testName := "Test User"
		db.members = []organization.Member{
			{
				ID:             1,
				UserID:         1,
				OrganizationID: 1,
				Role:           "owner",
				User:           &user.User{ID: 1, Email: "user@example.com", Name: &testName},
			},
		}

		svc := adminservice.NewService(db)
		handler := NewOrganizationHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/organizations/1/members", nil)
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.GetOrganizationMembers(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// =============================================================================
// Additional Internal Error Tests - Handler Layer
// =============================================================================

func TestUserHandler_UpdateUser_InternalError(t *testing.T) {
	t.Run("should return 500 when update fails", func(t *testing.T) {
		db := newMockHandlerDB()
		db.users[1] = &user.User{ID: 1, Email: "test@example.com"}
		db.updatesErr = gorm.ErrInvalidDB

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		body := bytes.NewBufferString(`{"name": "New Name"}`)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("PUT", "/users/1", body)
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.UpdateUser(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestUserHandler_DisableUser_InternalError(t *testing.T) {
	t.Run("should return 500 when disable fails", func(t *testing.T) {
		db := newMockHandlerDB()
		db.users[2] = &user.User{ID: 2, Email: "user@example.com", IsActive: true}
		db.updatesErr = gorm.ErrInvalidDB

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("POST", "/users/2/disable", nil)
		c.Params = gin.Params{{Key: "id", Value: "2"}}

		handler.DisableUser(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestUserHandler_EnableUser_InternalError(t *testing.T) {
	t.Run("should return 500 when enable fails", func(t *testing.T) {
		db := newMockHandlerDB()
		db.users[2] = &user.User{ID: 2, Email: "user@example.com", IsActive: false}
		db.updatesErr = gorm.ErrInvalidDB

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("POST", "/users/2/enable", nil)
		c.Params = gin.Params{{Key: "id", Value: "2"}}

		handler.EnableUser(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestUserHandler_GrantAdmin_InternalError(t *testing.T) {
	t.Run("should return 500 when grant admin fails", func(t *testing.T) {
		db := newMockHandlerDB()
		db.users[2] = &user.User{ID: 2, Email: "user@example.com", IsSystemAdmin: false}
		db.updatesErr = gorm.ErrInvalidDB

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("POST", "/users/2/grant-admin", nil)
		c.Params = gin.Params{{Key: "id", Value: "2"}}

		handler.GrantAdmin(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestUserHandler_RevokeAdmin_InternalError(t *testing.T) {
	t.Run("should return 500 when revoke admin fails", func(t *testing.T) {
		db := newMockHandlerDB()
		db.users[2] = &user.User{ID: 2, Email: "other@example.com", IsSystemAdmin: true}
		db.updatesErr = gorm.ErrInvalidDB

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("POST", "/users/2/revoke-admin", nil)
		c.Params = gin.Params{{Key: "id", Value: "2"}}

		handler.RevokeAdmin(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestOrganizationHandler_DeleteOrganization_InternalError(t *testing.T) {
	t.Run("should return 500 when delete fails", func(t *testing.T) {
		db := newMockHandlerDB()
		db.organizations[1] = &organization.Organization{ID: 1, Name: "Test Org"}
		db.runnerCount = 0
		db.deleteErr = gorm.ErrInvalidDB

		svc := adminservice.NewService(db)
		handler := NewOrganizationHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("DELETE", "/organizations/1", nil)
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.DeleteOrganization(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestOrganizationHandler_GetOrganizationMembers_InternalError(t *testing.T) {
	t.Run("should return 500 when get members fails", func(t *testing.T) {
		db := newMockHandlerDB()
		db.organizations[1] = &organization.Organization{ID: 1, Name: "Test Org"}
		db.findErr = gorm.ErrInvalidDB

		svc := adminservice.NewService(db)
		handler := NewOrganizationHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/organizations/1/members", nil)
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.GetOrganizationMembers(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestRunnerHandler_DisableRunner_InternalError(t *testing.T) {
	t.Run("should return 500 when disable fails", func(t *testing.T) {
		db := newMockHandlerDB()
		db.runners[1] = &runner.Runner{ID: 1, NodeID: "test-node", IsEnabled: true}
		db.saveErr = gorm.ErrInvalidDB

		svc := adminservice.NewService(db)
		handler := NewRunnerHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("POST", "/runners/1/disable", nil)
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.DisableRunner(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestRunnerHandler_EnableRunner_InternalError(t *testing.T) {
	t.Run("should return 500 when enable fails", func(t *testing.T) {
		db := newMockHandlerDB()
		db.runners[1] = &runner.Runner{ID: 1, NodeID: "test-node", IsEnabled: false}
		db.saveErr = gorm.ErrInvalidDB

		svc := adminservice.NewService(db)
		handler := NewRunnerHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("POST", "/runners/1/enable", nil)
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.EnableRunner(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestRunnerHandler_DeleteRunner_InternalError(t *testing.T) {
	t.Run("should return 500 when delete fails", func(t *testing.T) {
		db := newMockHandlerDB()
		db.runners[1] = &runner.Runner{ID: 1, NodeID: "test-node"}
		db.activePodCount = 0
		db.deleteErr = gorm.ErrInvalidDB

		svc := adminservice.NewService(db)
		handler := NewRunnerHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("DELETE", "/runners/1", nil)
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.DeleteRunner(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestRunnerHandler_GetRunner_InternalError(t *testing.T) {
	t.Run("should return 500 when org lookup fails", func(t *testing.T) {
		db := newMockHandlerDB()
		db.runners[1] = &runner.Runner{ID: 1, NodeID: "test-node", OrganizationID: 999}
		// Organization not found, but runner exists - should still return runner info

		svc := adminservice.NewService(db)
		handler := NewRunnerHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/runners/1", nil)
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.GetRunner(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// =============================================================================
// Auth Handler Login Tests
// =============================================================================

// mockAuthService implements authServiceInterface for testing
type mockAuthService struct {
	loginResult *auth.LoginResult
	loginErr    error
}

func (m *mockAuthService) Login(ctx context.Context, email, password string) (*auth.LoginResult, error) {
	if m.loginErr != nil {
		return nil, m.loginErr
	}
	return m.loginResult, nil
}

func TestAuthHandler_Login(t *testing.T) {
	t.Run("should return 400 for invalid request body", func(t *testing.T) {
		handler := NewAuthHandler(nil, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/auth/login", bytes.NewBufferString(`{}`))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.Login(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Contains(t, response["error"], "Invalid request")
	})

	t.Run("should return 400 for missing email", func(t *testing.T) {
		handler := NewAuthHandler(nil, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/auth/login", bytes.NewBufferString(`{"password":"test123"}`))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.Login(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("should return 400 for invalid email format", func(t *testing.T) {
		handler := NewAuthHandler(nil, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/auth/login", bytes.NewBufferString(`{"email":"invalid-email","password":"test123"}`))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.Login(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("should return 400 for missing password", func(t *testing.T) {
		handler := NewAuthHandler(nil, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/auth/login", bytes.NewBufferString(`{"email":"test@example.com"}`))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.Login(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("should return 400 for malformed JSON", func(t *testing.T) {
		handler := NewAuthHandler(nil, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/auth/login", bytes.NewBufferString(`{invalid json}`))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.Login(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("should return 401 when auth service returns error", func(t *testing.T) {
		mockSvc := &mockAuthService{
			loginErr: errors.New("invalid credentials"),
		}
		handler := NewAuthHandlerWithInterface(mockSvc, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/auth/login", bytes.NewBufferString(`{"email":"test@example.com","password":"wrongpass"}`))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.Login(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Contains(t, response["error"], "Invalid email or password")
	})

	t.Run("should return 403 when user is not system admin", func(t *testing.T) {
		mockSvc := &mockAuthService{
			loginResult: &auth.LoginResult{
				User: &user.User{
					ID:            1,
					Email:         "user@example.com",
					IsSystemAdmin: false,
					IsActive:      true,
				},
				Token:        "test-token",
				RefreshToken: "test-refresh",
			},
		}
		handler := NewAuthHandlerWithInterface(mockSvc, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/auth/login", bytes.NewBufferString(`{"email":"user@example.com","password":"password123"}`))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.Login(c)

		assert.Equal(t, http.StatusForbidden, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Contains(t, response["error"], "system administrator privileges")
	})

	t.Run("should return 403 when admin user is disabled", func(t *testing.T) {
		mockSvc := &mockAuthService{
			loginResult: &auth.LoginResult{
				User: &user.User{
					ID:            1,
					Email:         "admin@example.com",
					IsSystemAdmin: true,
					IsActive:      false, // Disabled user
				},
				Token:        "test-token",
				RefreshToken: "test-refresh",
			},
		}
		handler := NewAuthHandlerWithInterface(mockSvc, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/auth/login", bytes.NewBufferString(`{"email":"admin@example.com","password":"password123"}`))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.Login(c)

		assert.Equal(t, http.StatusForbidden, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Contains(t, response["error"], "disabled")
	})

	t.Run("should return 200 and tokens for successful admin login", func(t *testing.T) {
		mockSvc := &mockAuthService{
			loginResult: &auth.LoginResult{
				User: &user.User{
					ID:            1,
					Email:         "admin@example.com",
					Username:      "admin",
					IsSystemAdmin: true,
					IsActive:      true,
				},
				Token:        "test-access-token",
				RefreshToken: "test-refresh-token",
				ExpiresIn:    3600,
			},
		}
		handler := NewAuthHandlerWithInterface(mockSvc, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/auth/login", bytes.NewBufferString(`{"email":"admin@example.com","password":"password123"}`))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.Login(c)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "test-access-token", response["token"])
		assert.Equal(t, "test-refresh-token", response["refresh_token"])
		assert.NotNil(t, response["user"])

		userResp := response["user"].(map[string]interface{})
		assert.Equal(t, "admin@example.com", userResp["email"])
		assert.Equal(t, true, userResp["is_system_admin"])
	})
}

// =============================================================================
// AuditLog Handler Internal Error Tests
// =============================================================================

func TestAuditLogHandler_ListAuditLogs_InternalError(t *testing.T) {
	t.Run("should return 500 when service fails", func(t *testing.T) {
		db := newMockHandlerDB()
		db.findErr = gorm.ErrInvalidDB

		svc := adminservice.NewService(db)
		handler := NewAuditLogHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/audit-logs", nil)

		handler.ListAuditLogs(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
