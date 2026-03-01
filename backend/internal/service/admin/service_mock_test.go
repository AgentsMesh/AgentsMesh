package admin

import (
	"context"
	"errors"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/admin"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/agentpod"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/organization"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/runner"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/user"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/infra/database"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
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
	// Return a real SQLite DB with loops/loop_runs tables
	// so that DeleteOrganization's cleanup Exec calls don't panic.
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	db.Exec(`CREATE TABLE IF NOT EXISTS loops (id INTEGER PRIMARY KEY, organization_id INTEGER, runner_id INTEGER)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS loop_runs (id INTEGER PRIMARY KEY, organization_id INTEGER)`)
	return db
}

// Ensure mockDB implements database.DB
var _ database.DB = (*mockDB)(nil)
