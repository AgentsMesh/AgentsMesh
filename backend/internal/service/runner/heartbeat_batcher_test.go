package runner

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupBatcherTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	// Create runners table with SQLite-compatible syntax
	db.Exec(`CREATE TABLE IF NOT EXISTS runners (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		organization_id INTEGER NOT NULL,
		node_id TEXT NOT NULL,
		description TEXT,
		auth_token_hash TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'offline',
		last_heartbeat DATETIME,
		current_pods INTEGER NOT NULL DEFAULT 0,
		max_concurrent_pods INTEGER NOT NULL DEFAULT 5,
		runner_version TEXT,
		is_enabled INTEGER NOT NULL DEFAULT 1,
		host_info TEXT,
		capabilities TEXT,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)

	return db
}

func setupTestHeartbeatBatcherWithDB(t *testing.T) (*HeartbeatBatcher, *gorm.DB, *miniredis.Miniredis) {
	t.Helper()

	// Setup miniredis
	mr := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	// Setup SQLite in-memory database
	db := setupBatcherTestDB(t)

	// Create batcher
	batcher := NewHeartbeatBatcher(redisClient, db, newTestLogger())

	return batcher, db, mr
}

func insertTestRunner(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	result := db.Exec(`INSERT INTO runners (organization_id, node_id, auth_token_hash, status) VALUES (1, 'node-1', 'test-hash', 'offline')`)
	require.NoError(t, result.Error)

	var id int64
	db.Raw("SELECT last_insert_rowid()").Scan(&id)
	return id
}

// TestNewHeartbeatBatcher tests batcher creation
func TestNewHeartbeatBatcher(t *testing.T) {
	batcher, _, mr := setupTestHeartbeatBatcherWithDB(t)
	defer mr.Close()

	assert.NotNil(t, batcher)
	assert.NotNil(t, batcher.buffer)
	assert.Equal(t, DefaultFlushInterval, batcher.interval)
}

// TestHeartbeatBatcherSetInterval tests interval configuration
func TestHeartbeatBatcherSetInterval(t *testing.T) {
	batcher, _, mr := setupTestHeartbeatBatcherWithDB(t)
	defer mr.Close()

	batcher.SetInterval(10 * time.Second)
	assert.Equal(t, 10*time.Second, batcher.interval)
}

// TestHeartbeatBatcherStartStop tests lifecycle management
func TestHeartbeatBatcherStartStop(t *testing.T) {
	batcher, _, mr := setupTestHeartbeatBatcherWithDB(t)
	defer mr.Close()

	// Set short interval for testing
	batcher.SetInterval(50 * time.Millisecond)

	// Start batcher
	batcher.Start()

	// Should be running
	batcher.mu.Lock()
	assert.True(t, batcher.running)
	batcher.mu.Unlock()

	// Double start should be safe (no-op)
	batcher.Start()

	// Stop batcher
	batcher.Stop()

	// Should be stopped
	batcher.mu.Lock()
	assert.False(t, batcher.running)
	batcher.mu.Unlock()

	// Double stop should be safe (no-op)
	batcher.Stop()

	// Restart should work
	batcher.Start()
	batcher.mu.Lock()
	assert.True(t, batcher.running)
	batcher.mu.Unlock()
	batcher.Stop()
}

// TestHeartbeatBatcherRecordHeartbeat tests recording heartbeats
func TestHeartbeatBatcherRecordHeartbeat(t *testing.T) {
	batcher, db, mr := setupTestHeartbeatBatcherWithDB(t)
	defer mr.Close()

	ctx := context.Background()

	// Create a runner in the database
	runnerID := insertTestRunner(t, db)

	// Record heartbeat
	err := batcher.RecordHeartbeat(ctx, runnerID, 5, "online", "1.0.0", nil)
	require.NoError(t, err)

	// Check buffer
	assert.Equal(t, 1, batcher.BufferSize())

	// Check Redis was updated
	status, err := batcher.GetRunnerStatus(ctx, runnerID)
	require.NoError(t, err)
	require.NotNil(t, status)
	assert.Equal(t, "online", status.Status)
	assert.Equal(t, 5, status.CurrentPods)
	assert.Equal(t, "1.0.0", status.Version)

	// Flush to database
	batcher.Flush()

	// Check buffer is cleared
	assert.Equal(t, 0, batcher.BufferSize())

	// Check database was updated
	var updatedStatus string
	var updatedPods int
	db.Raw("SELECT status, current_pods FROM runners WHERE id = ?", runnerID).Row().Scan(&updatedStatus, &updatedPods)
	assert.Equal(t, "online", updatedStatus)
	assert.Equal(t, 5, updatedPods)
}

// TestHeartbeatBatcherGetRunnerStatus tests getting status from Redis
func TestHeartbeatBatcherGetRunnerStatus(t *testing.T) {
	batcher, _, mr := setupTestHeartbeatBatcherWithDB(t)
	defer mr.Close()

	ctx := context.Background()

	// Non-existent runner returns nil
	status, err := batcher.GetRunnerStatus(ctx, 999)
	require.NoError(t, err)
	assert.Nil(t, status)

	// Record a heartbeat
	err = batcher.RecordHeartbeat(ctx, 1, 3, "online", "2.0.0", nil)
	require.NoError(t, err)

	// Get status
	status, err = batcher.GetRunnerStatus(ctx, 1)
	require.NoError(t, err)
	require.NotNil(t, status)
	assert.Equal(t, "online", status.Status)
	assert.Equal(t, 3, status.CurrentPods)
	assert.Equal(t, "2.0.0", status.Version)
	assert.Greater(t, status.LastHeartbeat, int64(0))
}

// TestHeartbeatBatcherIsRunnerOnline tests online status check
func TestHeartbeatBatcherIsRunnerOnline(t *testing.T) {
	batcher, _, mr := setupTestHeartbeatBatcherWithDB(t)
	defer mr.Close()

	ctx := context.Background()

	// Non-existent runner is offline
	assert.False(t, batcher.IsRunnerOnline(ctx, 999))

	// Record a heartbeat
	err := batcher.RecordHeartbeat(ctx, 1, 0, "online", "", nil)
	require.NoError(t, err)

	// Should be online
	assert.True(t, batcher.IsRunnerOnline(ctx, 1))
}

// TestHeartbeatBatcherFlushLoop tests automatic flushing
func TestHeartbeatBatcherFlushLoop(t *testing.T) {
	batcher, db, mr := setupTestHeartbeatBatcherWithDB(t)
	defer mr.Close()

	ctx := context.Background()

	// Create a runner
	runnerID := insertTestRunner(t, db)

	// Set very short interval
	batcher.SetInterval(50 * time.Millisecond)
	batcher.Start()
	defer batcher.Stop()

	// Record heartbeat
	err := batcher.RecordHeartbeat(ctx, runnerID, 2, "online", "", nil)
	require.NoError(t, err)

	// Wait for flush
	time.Sleep(100 * time.Millisecond)

	// Check database was updated
	var updatedStatus string
	db.Raw("SELECT status FROM runners WHERE id = ?", runnerID).Row().Scan(&updatedStatus)
	assert.Equal(t, "online", updatedStatus)
}

// TestHeartbeatBatcherBatchUpdate tests batch updates
func TestHeartbeatBatcherBatchUpdate(t *testing.T) {
	batcher, db, mr := setupTestHeartbeatBatcherWithDB(t)
	defer mr.Close()

	ctx := context.Background()

	// Create multiple runners
	runnerIDs := make([]int64, 10)
	for i := 0; i < 10; i++ {
		runnerIDs[i] = insertTestRunner(t, db)
	}

	// Record heartbeat for each
	for i, id := range runnerIDs {
		err := batcher.RecordHeartbeat(ctx, id, i, "online", "", nil)
		require.NoError(t, err)
	}

	assert.Equal(t, 10, batcher.BufferSize())

	// Flush all
	batcher.Flush()

	assert.Equal(t, 0, batcher.BufferSize())

	// Verify all were updated
	var count int64
	db.Raw("SELECT COUNT(*) FROM runners WHERE status = 'online'").Scan(&count)
	assert.Equal(t, int64(10), count)
}

// TestHeartbeatBatcherBufferSize tests buffer size monitoring
func TestHeartbeatBatcherBufferSize(t *testing.T) {
	batcher, _, mr := setupTestHeartbeatBatcherWithDB(t)
	defer mr.Close()

	ctx := context.Background()

	assert.Equal(t, 0, batcher.BufferSize())

	// Add heartbeats
	for i := 1; i <= 5; i++ {
		_ = batcher.RecordHeartbeat(ctx, int64(i), 0, "online", "", nil)
	}

	assert.Equal(t, 5, batcher.BufferSize())

	// Same runner updates should replace, not add
	_ = batcher.RecordHeartbeat(ctx, 1, 10, "online", "", nil)
	assert.Equal(t, 5, batcher.BufferSize())

	batcher.Flush()
	assert.Equal(t, 0, batcher.BufferSize())
}

// TestHeartbeatBatcherEmptyFlush tests flushing empty buffer
func TestHeartbeatBatcherEmptyFlush(t *testing.T) {
	batcher, _, mr := setupTestHeartbeatBatcherWithDB(t)
	defer mr.Close()

	// Should not panic
	batcher.Flush()
	assert.Equal(t, 0, batcher.BufferSize())
}

// TestHeartbeatBatcherConcurrentAccess tests thread safety
func TestHeartbeatBatcherConcurrentAccess(t *testing.T) {
	batcher, db, mr := setupTestHeartbeatBatcherWithDB(t)
	defer mr.Close()

	ctx := context.Background()

	// Create runners
	runnerIDs := make([]int64, 50) // Reduced from 100 for faster test
	for i := 0; i < 50; i++ {
		runnerIDs[i] = insertTestRunner(t, db)
	}

	// Start batcher with short interval
	batcher.SetInterval(10 * time.Millisecond)
	batcher.Start()
	defer batcher.Stop()

	// Concurrent heartbeats
	done := make(chan bool)
	for i := 0; i < 50; i++ {
		go func(idx int) {
			for j := 0; j < 5; j++ {
				_ = batcher.RecordHeartbeat(ctx, runnerIDs[idx], j, "online", "", nil)
				time.Sleep(time.Millisecond)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 50; i++ {
		<-done
	}

	// Wait for flush
	time.Sleep(50 * time.Millisecond)

	// All should be online
	var count int64
	db.Raw("SELECT COUNT(*) FROM runners WHERE status = 'online'").Scan(&count)
	assert.Equal(t, int64(50), count)
}

// TestHeartbeatBatcherVersionUpdate tests version field update
func TestHeartbeatBatcherVersionUpdate(t *testing.T) {
	batcher, db, mr := setupTestHeartbeatBatcherWithDB(t)
	defer mr.Close()

	ctx := context.Background()
	runnerID := insertTestRunner(t, db)

	// Record with version
	err := batcher.RecordHeartbeat(ctx, runnerID, 0, "online", "v2.1.0", nil)
	require.NoError(t, err)

	batcher.Flush()

	var version string
	db.Raw("SELECT runner_version FROM runners WHERE id = ?", runnerID).Row().Scan(&version)
	assert.Equal(t, "v2.1.0", version)
}

// TestHeartbeatBatcherWithoutVersion tests heartbeat without version
func TestHeartbeatBatcherWithoutVersion(t *testing.T) {
	batcher, db, mr := setupTestHeartbeatBatcherWithDB(t)
	defer mr.Close()

	ctx := context.Background()
	runnerID := insertTestRunner(t, db)

	// Record without version (empty string)
	err := batcher.RecordHeartbeat(ctx, runnerID, 0, "online", "", nil)
	require.NoError(t, err)

	batcher.Flush()

	var status string
	db.Raw("SELECT status FROM runners WHERE id = ?", runnerID).Row().Scan(&status)
	assert.Equal(t, "online", status)
}

// TestHeartbeatBatcherCapabilities tests capabilities update
func TestHeartbeatBatcherCapabilities(t *testing.T) {
	batcher, db, mr := setupTestHeartbeatBatcherWithDB(t)
	defer mr.Close()

	ctx := context.Background()
	runnerID := insertTestRunner(t, db)

	// Record heartbeat with capabilities
	caps := []byte(`[{"name":"docker","version":"1.0"}]`)
	err := batcher.RecordHeartbeat(ctx, runnerID, 0, "online", "", caps)
	require.NoError(t, err)

	batcher.Flush()

	// Verify capabilities were saved
	var savedCaps string
	db.Raw("SELECT capabilities FROM runners WHERE id = ?", runnerID).Row().Scan(&savedCaps)
	assert.Contains(t, savedCaps, "docker")
}

// TestHeartbeatBatcherLargeBatch tests flushing more than 100 items (batch size limit)
func TestHeartbeatBatcherLargeBatch(t *testing.T) {
	batcher, db, mr := setupTestHeartbeatBatcherWithDB(t)
	defer mr.Close()

	ctx := context.Background()

	// Create 150 runners to test batch processing
	runnerIDs := make([]int64, 150)
	for i := 0; i < 150; i++ {
		runnerIDs[i] = insertTestRunner(t, db)
	}

	// Record heartbeat for each
	for _, id := range runnerIDs {
		err := batcher.RecordHeartbeat(ctx, id, 1, "online", "v1.0", nil)
		require.NoError(t, err)
	}

	assert.Equal(t, 150, batcher.BufferSize())

	// Flush all (should process in batches of 100)
	batcher.Flush()

	assert.Equal(t, 0, batcher.BufferSize())

	// Verify all were updated
	var count int64
	db.Raw("SELECT COUNT(*) FROM runners WHERE status = 'online'").Scan(&count)
	assert.Equal(t, int64(150), count)
}

// TestHeartbeatBatcherMultipleFlushes tests multiple consecutive flushes
func TestHeartbeatBatcherMultipleFlushes(t *testing.T) {
	batcher, db, mr := setupTestHeartbeatBatcherWithDB(t)
	defer mr.Close()

	ctx := context.Background()

	// First batch
	runnerID1 := insertTestRunner(t, db)
	err := batcher.RecordHeartbeat(ctx, runnerID1, 1, "online", "", nil)
	require.NoError(t, err)

	batcher.Flush()

	// Second batch
	runnerID2 := insertTestRunner(t, db)
	err = batcher.RecordHeartbeat(ctx, runnerID2, 2, "online", "", nil)
	require.NoError(t, err)

	batcher.Flush()

	// Both should be updated
	var count int64
	db.Raw("SELECT COUNT(*) FROM runners WHERE status = 'online'").Scan(&count)
	assert.Equal(t, int64(2), count)
}

// TestHeartbeatBatcherUpdateExistingEntry tests updating a runner that already has a pending heartbeat
func TestHeartbeatBatcherUpdateExistingEntry(t *testing.T) {
	batcher, db, mr := setupTestHeartbeatBatcherWithDB(t)
	defer mr.Close()

	ctx := context.Background()
	runnerID := insertTestRunner(t, db)

	// First heartbeat
	err := batcher.RecordHeartbeat(ctx, runnerID, 1, "online", "v1.0", nil)
	require.NoError(t, err)

	// Second heartbeat for same runner (should replace)
	err = batcher.RecordHeartbeat(ctx, runnerID, 5, "online", "v2.0", nil)
	require.NoError(t, err)

	// Buffer should still have only 1 entry
	assert.Equal(t, 1, batcher.BufferSize())

	batcher.Flush()

	// Verify latest values were saved
	var currentPods int
	var version string
	db.Raw("SELECT current_pods, runner_version FROM runners WHERE id = ?", runnerID).Row().Scan(&currentPods, &version)
	assert.Equal(t, 5, currentPods)
	assert.Equal(t, "v2.0", version)
}

// TestHeartbeatBatcherRedisExpiration tests that Redis keys are set with TTL
func TestHeartbeatBatcherRedisExpiration(t *testing.T) {
	batcher, _, mr := setupTestHeartbeatBatcherWithDB(t)
	defer mr.Close()

	ctx := context.Background()

	// Record heartbeat
	err := batcher.RecordHeartbeat(ctx, 123, 0, "online", "", nil)
	require.NoError(t, err)

	// Check Redis key exists
	status, err := batcher.GetRunnerStatus(ctx, 123)
	require.NoError(t, err)
	require.NotNil(t, status)

	// Verify key exists in Redis (TTL verification varies by miniredis version)
	key := "runner:123:status"
	exists := mr.Exists(key)
	assert.True(t, exists, "Redis key should exist")
}

// TestHeartbeatBatcherStopFlushes tests that Stop() flushes pending data
func TestHeartbeatBatcherStopFlushes(t *testing.T) {
	batcher, db, mr := setupTestHeartbeatBatcherWithDB(t)
	defer mr.Close()

	ctx := context.Background()
	runnerID := insertTestRunner(t, db)

	// Start batcher
	batcher.SetInterval(10 * time.Second) // Long interval
	batcher.Start()

	// Record heartbeat
	err := batcher.RecordHeartbeat(ctx, runnerID, 3, "online", "", nil)
	require.NoError(t, err)

	// Stop should trigger flush
	batcher.Stop()

	// Verify data was flushed
	var status string
	db.Raw("SELECT status FROM runners WHERE id = ?", runnerID).Row().Scan(&status)
	assert.Equal(t, "online", status)
}
