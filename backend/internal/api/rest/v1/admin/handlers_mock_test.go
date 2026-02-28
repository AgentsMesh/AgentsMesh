package admin

import (
	"context"
	"net/http/httptest"

	"github.com/anthropics/agentsmesh/backend/internal/domain/admin"
	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/domain/organization"
	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
	"github.com/anthropics/agentsmesh/backend/internal/infra/database"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
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
	// Return a real SQLite DB with loops/loop_runs tables
	// so that DeleteOrganization's cleanup Exec calls don't panic.
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	db.Exec(`CREATE TABLE IF NOT EXISTS loops (id INTEGER PRIMARY KEY, organization_id INTEGER, runner_id INTEGER)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS loop_runs (id INTEGER PRIMARY KEY, organization_id INTEGER)`)
	return db
}

var _ database.DB = (*mockHandlerDB)(nil)

// Helper function to create test context with admin user
func createAdminContext(w *httptest.ResponseRecorder) *gin.Context {
	c, _ := gin.CreateTestContext(w)
	c.Set("admin_user_id", int64(1))
	c.Set("admin_user", &user.User{ID: 1, Email: "admin@example.com", IsSystemAdmin: true})
	return c
}
