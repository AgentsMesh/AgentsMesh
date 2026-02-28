package instance

import (
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// mockRunnerConnector is a test double for ConnectedRunnerIDsProvider
type mockRunnerConnector struct {
	mu        sync.Mutex
	runnerIDs []int64
}

func (m *mockRunnerConnector) GetConnectedRunnerIDs() []int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.runnerIDs == nil {
		return nil
	}
	result := make([]int64, len(m.runnerIDs))
	copy(result, m.runnerIDs)
	return result
}

func (m *mockRunnerConnector) setRunnerIDs(ids []int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.runnerIDs = ids
}

// testLogger returns a silent slog.Logger for tests
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)

	// Create minimal runners table
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS runners (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL,
			node_id TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'online'
		)
	`).Error
	require.NoError(t, err)

	return db
}

type runnerSeed struct {
	ID    int64
	OrgID int64
}

func seedRunners(t *testing.T, db *gorm.DB, runners []runnerSeed) {
	for _, r := range runners {
		err := db.Exec("INSERT INTO runners (id, organization_id) VALUES (?, ?)", r.ID, r.OrgID).Error
		require.NoError(t, err)
	}
}

func TestOrgAwarenessService_GetLocalOrgIDs_NoRunners(t *testing.T) {
	db := setupTestDB(t)
	connector := &mockRunnerConnector{}
	svc := NewOrgAwarenessService(db, connector, nil, "test:8080", testLogger())

	svc.Refresh()

	result := svc.GetLocalOrgIDs()
	assert.Nil(t, result, "should return nil when no runners are connected")
}

func TestOrgAwarenessService_GetLocalOrgIDs_WithRunners(t *testing.T) {
	db := setupTestDB(t)
	seedRunners(t, db, []runnerSeed{
		{1, 100},
		{2, 100},
		{3, 200},
		{4, 300},
	})

	connector := &mockRunnerConnector{runnerIDs: []int64{1, 3}} // org 100 and 200
	svc := NewOrgAwarenessService(db, connector, nil, "test:8080", testLogger())

	svc.Refresh()

	result := svc.GetLocalOrgIDs()
	assert.NotNil(t, result)
	assert.ElementsMatch(t, []int64{100, 200}, result)
}

func TestOrgAwarenessService_GetLocalOrgIDs_ReturnsDefensiveCopy(t *testing.T) {
	db := setupTestDB(t)
	seedRunners(t, db, []runnerSeed{{1, 100}})

	connector := &mockRunnerConnector{runnerIDs: []int64{1}}
	svc := NewOrgAwarenessService(db, connector, nil, "test:8080", testLogger())
	svc.Refresh()

	// Mutate the returned slice
	result1 := svc.GetLocalOrgIDs()
	result1[0] = 999

	// Internal state should be unaffected
	result2 := svc.GetLocalOrgIDs()
	assert.Equal(t, int64(100), result2[0], "internal state should not be affected by caller mutation")
}

func TestOrgAwarenessService_Refresh_UpdatesOnRunnerChange(t *testing.T) {
	db := setupTestDB(t)
	seedRunners(t, db, []runnerSeed{
		{1, 100},
		{2, 200},
		{3, 300},
	})

	connector := &mockRunnerConnector{runnerIDs: []int64{1}} // only org 100
	svc := NewOrgAwarenessService(db, connector, nil, "test:8080", testLogger())
	svc.Refresh()

	assert.ElementsMatch(t, []int64{100}, svc.GetLocalOrgIDs())

	// Simulate additional runners connecting
	connector.setRunnerIDs([]int64{1, 2, 3})
	svc.Refresh()

	assert.ElementsMatch(t, []int64{100, 200, 300}, svc.GetLocalOrgIDs())

	// Simulate all runners disconnecting
	connector.setRunnerIDs(nil)
	svc.Refresh()

	assert.Nil(t, svc.GetLocalOrgIDs(), "should return nil when all runners disconnect")
}

func TestOrgAwarenessService_Refresh_DeduplicatesOrgs(t *testing.T) {
	db := setupTestDB(t)
	seedRunners(t, db, []runnerSeed{
		{1, 100},
		{2, 100}, // same org
		{3, 100}, // same org
	})

	connector := &mockRunnerConnector{runnerIDs: []int64{1, 2, 3}}
	svc := NewOrgAwarenessService(db, connector, nil, "test:8080", testLogger())
	svc.Refresh()

	result := svc.GetLocalOrgIDs()
	assert.Len(t, result, 1, "should deduplicate org IDs")
	assert.Equal(t, int64(100), result[0])
}

func TestOrgAwarenessService_Refresh_IgnoresUnknownRunners(t *testing.T) {
	db := setupTestDB(t)
	seedRunners(t, db, []runnerSeed{{1, 100}})

	// Runner ID 999 doesn't exist in DB
	connector := &mockRunnerConnector{runnerIDs: []int64{1, 999}}
	svc := NewOrgAwarenessService(db, connector, nil, "test:8080", testLogger())
	svc.Refresh()

	result := svc.GetLocalOrgIDs()
	assert.ElementsMatch(t, []int64{100}, result, "should only return orgs for known runners")
}

func TestOrgAwarenessService_StartStop(t *testing.T) {
	db := setupTestDB(t)
	seedRunners(t, db, []runnerSeed{{1, 100}})

	connector := &mockRunnerConnector{runnerIDs: []int64{1}}
	svc := NewOrgAwarenessService(db, connector, nil, "test:8080", testLogger())

	svc.Start()

	// Should have initial data from Start's Refresh call
	result := svc.GetLocalOrgIDs()
	assert.ElementsMatch(t, []int64{100}, result)

	svc.Stop()

	// Allow goroutine to exit
	time.Sleep(10 * time.Millisecond)
}

func TestOrgAwarenessService_ConcurrentAccess(t *testing.T) {
	db := setupTestDB(t)
	seedRunners(t, db, []runnerSeed{
		{1, 100},
		{2, 200},
	})

	connector := &mockRunnerConnector{runnerIDs: []int64{1, 2}}
	svc := NewOrgAwarenessService(db, connector, nil, "test:8080", testLogger())
	svc.Refresh()

	// Concurrent reads and writes should not panic or race
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			_ = svc.GetLocalOrgIDs()
		}()
		go func() {
			defer wg.Done()
			svc.Refresh()
		}()
	}
	wg.Wait()
}
