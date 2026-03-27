package loop

import (
	"context"
	"testing"

	loopDomain "github.com/anthropics/agentsmesh/backend/internal/domain/loop"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func setupLoopServiceTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		Logger:                                   gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	db.Exec(`
		CREATE TABLE IF NOT EXISTS loops (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			slug TEXT NOT NULL,
			description TEXT,
			agent_slug INTEGER,
			custom_agent_slug INTEGER,
			permission_mode TEXT NOT NULL DEFAULT 'bypassPermissions',
			prompt_template TEXT NOT NULL DEFAULT '',
			repository_id INTEGER,
			runner_id INTEGER,
			branch_name TEXT,
			ticket_id INTEGER,
			credential_profile_id INTEGER,
			config_overrides BLOB DEFAULT NULL,
			prompt_variables BLOB DEFAULT NULL,
			execution_mode TEXT NOT NULL DEFAULT 'autopilot',
			cron_expression TEXT,
			autopilot_config BLOB DEFAULT NULL,
			callback_url TEXT,
			status TEXT NOT NULL DEFAULT 'enabled',
			sandbox_strategy TEXT NOT NULL DEFAULT 'persistent',
			session_persistence INTEGER NOT NULL DEFAULT 1,
			concurrency_policy TEXT NOT NULL DEFAULT 'skip',
			max_concurrent_runs INTEGER NOT NULL DEFAULT 1,
			max_retained_runs INTEGER NOT NULL DEFAULT 0,
			timeout_minutes INTEGER NOT NULL DEFAULT 60,
			idle_timeout_sec INTEGER NOT NULL DEFAULT 30,
			sandbox_path TEXT,
			last_pod_key TEXT,
			created_by_id INTEGER NOT NULL DEFAULT 0,
			total_runs INTEGER NOT NULL DEFAULT 0,
			successful_runs INTEGER NOT NULL DEFAULT 0,
			failed_runs INTEGER NOT NULL DEFAULT 0,
			last_run_at DATETIME,
			next_run_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(organization_id, slug)
		)
	`)

	// loop_runs table is needed for Delete (active run check)
	db.Exec(`
		CREATE TABLE IF NOT EXISTS loop_runs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL,
			loop_id INTEGER NOT NULL,
			run_number INTEGER NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			pod_key TEXT,
			autopilot_controller_key TEXT,
			trigger_type TEXT NOT NULL DEFAULT 'manual',
			trigger_source TEXT,
			trigger_params BLOB DEFAULT NULL,
			resolved_prompt TEXT,
			started_at DATETIME,
			finished_at DATETIME,
			duration_sec INTEGER,
			exit_summary TEXT,
			error_message TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)

	// Minimal pods table for SSOT queries
	db.Exec(`
		CREATE TABLE IF NOT EXISTS pods (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			pod_key TEXT NOT NULL UNIQUE,
			status TEXT NOT NULL DEFAULT 'initializing',
			agent_status TEXT NOT NULL DEFAULT 'idle',
			finished_at DATETIME,
			last_activity DATETIME,
			agent_waiting_since DATETIME,
			alias TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	return db
}

func newTestLoopService(t *testing.T) (*LoopService, *gorm.DB) {
	db := setupLoopServiceTestDB(t)
	repo := infra.NewLoopRepository(db)
	svc := NewLoopService(repo)
	return svc, db
}

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

func TestLoopService_Create(t *testing.T) {
	svc, _ := newTestLoopService(t)
	ctx := context.Background()

	t.Run("should create loop with defaults", func(t *testing.T) {
		loop, err := svc.Create(ctx, &CreateLoopRequest{
			OrganizationID: 1,
			CreatedByID:    1,
			Name:           "Daily Review",
			Slug:           "daily-review",
			PromptTemplate: "Review code",
		})
		require.NoError(t, err)
		assert.NotZero(t, loop.ID)
		assert.Equal(t, "daily-review", loop.Slug)
		assert.Equal(t, loopDomain.StatusEnabled, loop.Status)
		assert.Equal(t, loopDomain.ExecutionModeAutopilot, loop.ExecutionMode)
		assert.Equal(t, loopDomain.SandboxStrategyPersistent, loop.SandboxStrategy)
		assert.Equal(t, loopDomain.ConcurrencyPolicySkip, loop.ConcurrencyPolicy)
		assert.Equal(t, 1, loop.MaxConcurrentRuns)
		assert.Equal(t, 60, loop.TimeoutMinutes)
	})

	t.Run("should auto-generate slug from name", func(t *testing.T) {
		loop, err := svc.Create(ctx, &CreateLoopRequest{
			OrganizationID: 1,
			CreatedByID:    1,
			Name:           "My Cool Loop",
			PromptTemplate: "Do something",
		})
		require.NoError(t, err)
		assert.Equal(t, "my-cool-loop", loop.Slug)
	})

	t.Run("should reject invalid slug", func(t *testing.T) {
		_, err := svc.Create(ctx, &CreateLoopRequest{
			OrganizationID: 1,
			CreatedByID:    1,
			Name:           "Test",
			Slug:           "AB", // too short
			PromptTemplate: "prompt",
		})
		assert.ErrorIs(t, err, ErrInvalidSlug)
	})
}

func TestLoopService_GetBySlug(t *testing.T) {
	svc, _ := newTestLoopService(t)
	ctx := context.Background()

	_, err := svc.Create(ctx, &CreateLoopRequest{
		OrganizationID: 1, CreatedByID: 1,
		Name: "Test", Slug: "test-get", PromptTemplate: "p",
	})
	require.NoError(t, err)

	t.Run("should return existing loop", func(t *testing.T) {
		loop, err := svc.GetBySlug(ctx, 1, "test-get")
		require.NoError(t, err)
		assert.Equal(t, "Test", loop.Name)
	})

	t.Run("should return ErrLoopNotFound for non-existent", func(t *testing.T) {
		_, err := svc.GetBySlug(ctx, 1, "no-such")
		assert.ErrorIs(t, err, ErrLoopNotFound)
	})

	t.Run("should return ErrLoopNotFound for wrong org", func(t *testing.T) {
		_, err := svc.GetBySlug(ctx, 999, "test-get")
		assert.ErrorIs(t, err, ErrLoopNotFound)
	})
}

func TestLoopService_Update(t *testing.T) {
	svc, _ := newTestLoopService(t)
	ctx := context.Background()

	_, err := svc.Create(ctx, &CreateLoopRequest{
		OrganizationID: 1, CreatedByID: 1,
		Name: "Before", Slug: "update-test", PromptTemplate: "original",
	})
	require.NoError(t, err)

	t.Run("should update specified fields", func(t *testing.T) {
		updated, err := svc.Update(ctx, 1, "update-test", &UpdateLoopRequest{
			Name:           strPtr("After"),
			PromptTemplate: strPtr("updated prompt"),
			TimeoutMinutes: intPtr(120),
		})
		require.NoError(t, err)
		assert.Equal(t, "After", updated.Name)
		assert.Equal(t, "updated prompt", updated.PromptTemplate)
		assert.Equal(t, 120, updated.TimeoutMinutes)
	})

	t.Run("should return error for non-existent loop", func(t *testing.T) {
		_, err := svc.Update(ctx, 1, "no-such", &UpdateLoopRequest{
			Name: strPtr("X"),
		})
		assert.ErrorIs(t, err, ErrLoopNotFound)
	})
}

func TestLoopService_Delete(t *testing.T) {
	svc, _ := newTestLoopService(t)
	ctx := context.Background()

	_, err := svc.Create(ctx, &CreateLoopRequest{
		OrganizationID: 1, CreatedByID: 1,
		Name: "Deletable", Slug: "deletable", PromptTemplate: "p",
	})
	require.NoError(t, err)

	t.Run("should delete existing loop", func(t *testing.T) {
		err := svc.Delete(ctx, 1, "deletable")
		require.NoError(t, err)

		_, err = svc.GetBySlug(ctx, 1, "deletable")
		assert.ErrorIs(t, err, ErrLoopNotFound)
	})

	t.Run("should return error for non-existent", func(t *testing.T) {
		err := svc.Delete(ctx, 1, "no-such")
		assert.ErrorIs(t, err, ErrLoopNotFound)
	})
}

func TestLoopService_SetStatus(t *testing.T) {
	svc, _ := newTestLoopService(t)
	ctx := context.Background()

	_, err := svc.Create(ctx, &CreateLoopRequest{
		OrganizationID: 1, CreatedByID: 1,
		Name: "Status Test", Slug: "status-test", PromptTemplate: "p",
	})
	require.NoError(t, err)

	t.Run("should change status to disabled", func(t *testing.T) {
		loop, err := svc.SetStatus(ctx, 1, "status-test", loopDomain.StatusDisabled)
		require.NoError(t, err)
		assert.Equal(t, loopDomain.StatusDisabled, loop.Status)
	})

	t.Run("should change status back to enabled", func(t *testing.T) {
		loop, err := svc.SetStatus(ctx, 1, "status-test", loopDomain.StatusEnabled)
		require.NoError(t, err)
		assert.Equal(t, loopDomain.StatusEnabled, loop.Status)
	})
}

func TestLoopService_UpdateStats(t *testing.T) {
	svc, _ := newTestLoopService(t)
	ctx := context.Background()

	created, err := svc.Create(ctx, &CreateLoopRequest{
		OrganizationID: 1, CreatedByID: 1,
		Name: "Stats", Slug: "stats-test", PromptTemplate: "p",
	})
	require.NoError(t, err)

	err = svc.UpdateStats(ctx, created.ID, 10, 8, 2)
	require.NoError(t, err)

	got, err := svc.GetBySlug(ctx, 1, "stats-test")
	require.NoError(t, err)
	assert.Equal(t, 10, got.TotalRuns)
	assert.Equal(t, 8, got.SuccessfulRuns)
	assert.Equal(t, 2, got.FailedRuns)
}

func TestLoopService_UpdateRuntimeState(t *testing.T) {
	svc, _ := newTestLoopService(t)
	ctx := context.Background()

	created, err := svc.Create(ctx, &CreateLoopRequest{
		OrganizationID: 1, CreatedByID: 1,
		Name: "Runtime", Slug: "runtime-test", PromptTemplate: "p",
	})
	require.NoError(t, err)

	err = svc.UpdateRuntimeState(ctx, created.ID, strPtr("/sandbox/path"), strPtr("pod-abc"))
	require.NoError(t, err)

	got, err := svc.GetBySlug(ctx, 1, "runtime-test")
	require.NoError(t, err)
	assert.NotNil(t, got.SandboxPath)
	assert.Equal(t, "/sandbox/path", *got.SandboxPath)
	assert.NotNil(t, got.LastPodKey)
	assert.Equal(t, "pod-abc", *got.LastPodKey)
}

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"lowercase name", "daily review", "daily-review"},
		{"mixed case", "My Cool Loop", "my-cool-loop"},
		{"special chars", "PR Review (v2)", "pr-review-v2"},
		{"short name padded", "ab", "ab-loop"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, generateSlug(tt.input))
		})
	}

	t.Run("chinese name generates valid slug", func(t *testing.T) {
		slug := generateSlug("每日代码审查")
		assert.True(t, slugRegex.MatchString(slug), "slug %q should match regex", slug)
		assert.True(t, len(slug) >= 3, "slug should be at least 3 chars")
	})

	t.Run("mixed chinese and ascii", func(t *testing.T) {
		slug := generateSlug("每日review任务")
		assert.True(t, slugRegex.MatchString(slug), "slug %q should match regex", slug)
		assert.Contains(t, slug, "review")
	})

	t.Run("emoji name generates valid slug", func(t *testing.T) {
		slug := generateSlug("🚀 deploy bot")
		assert.True(t, slugRegex.MatchString(slug), "slug %q should match regex", slug)
	})

	t.Run("pure unicode generates timestamp-based slug", func(t *testing.T) {
		slug := generateSlug("日本語テスト")
		assert.True(t, slugRegex.MatchString(slug), "slug %q should match regex", slug)
		assert.Contains(t, slug, "loop-")
	})
}

func TestLoopService_Create_ChineseName(t *testing.T) {
	svc, _ := newTestLoopService(t)
	ctx := context.Background()

	loop, err := svc.Create(ctx, &CreateLoopRequest{
		OrganizationID: 1,
		CreatedByID:    1,
		Name:           "每日代码审查",
		PromptTemplate: "Review code daily",
	})
	require.NoError(t, err)
	assert.NotZero(t, loop.ID)
	assert.True(t, slugRegex.MatchString(loop.Slug), "auto-generated slug %q should be valid", loop.Slug)
}
