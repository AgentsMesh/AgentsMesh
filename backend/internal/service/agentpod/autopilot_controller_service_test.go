package agentpod

import (
	"context"
	"strings"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ==================== Test Helpers ====================

// mockCommandSender implements AutopilotCommandSender for testing.
type mockCommandSender struct {
	called   bool
	runnerID int64
	cmd      *runnerv1.CreateAutopilotCommand
	err      error
}

func (m *mockCommandSender) SendCreateAutopilot(runnerID int64, cmd *runnerv1.CreateAutopilotCommand) error {
	m.called = true
	m.runnerID = runnerID
	m.cmd = cmd
	return m.err
}

func setupAutopilotTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	// autopilot_controllers table (SQLite-compatible schema)
	db.Exec(`CREATE TABLE IF NOT EXISTS autopilot_controllers (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		organization_id INTEGER NOT NULL,
		autopilot_controller_key TEXT NOT NULL UNIQUE,
		pod_key TEXT NOT NULL,
		pod_id INTEGER NOT NULL,
		runner_id INTEGER NOT NULL,
		prompt TEXT,
		phase TEXT NOT NULL DEFAULT 'initializing',
		current_iteration INTEGER NOT NULL DEFAULT 0,
		max_iterations INTEGER NOT NULL DEFAULT 10,
		iteration_timeout_sec INTEGER NOT NULL DEFAULT 300,
		circuit_breaker_state TEXT NOT NULL DEFAULT 'closed',
		circuit_breaker_reason TEXT,
		no_progress_threshold INTEGER NOT NULL DEFAULT 3,
		same_error_threshold INTEGER NOT NULL DEFAULT 5,
		approval_timeout_min INTEGER NOT NULL DEFAULT 30,
		control_agent_slug TEXT,
		control_prompt_template TEXT,
		mcp_config_json TEXT,
		user_takeover INTEGER NOT NULL DEFAULT 0,
		started_at DATETIME,
		last_iteration_at DATETIME,
		completed_at DATETIME,
		approval_request_at DATETIME,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)

	return db
}

func newTestPod() *agentpod.Pod {
	return &agentpod.Pod{
		ID:             42,
		OrganizationID: 1,
		PodKey:         "pod-abc123",
		RunnerID:       7,
		Status:         agentpod.StatusRunning,
	}
}

// ==================== CreateAndStart Tests ====================

func TestCreateAndStart_Success(t *testing.T) {
	db := setupAutopilotTestDB(t)
	sender := &mockCommandSender{}
	svc := newTestAutopilotService(db)
	svc.SetCommandSender(sender)

	pod := newTestPod()
	controller, err := svc.CreateAndStart(context.Background(), &CreateAndStartRequest{
		OrganizationID: 1,
		Pod:            pod,
		Prompt:  "Review the code",
	})

	require.NoError(t, err)
	require.NotNil(t, controller)

	// Verify DB record
	assert.Equal(t, int64(1), controller.OrganizationID)
	assert.Equal(t, pod.PodKey, controller.PodKey)
	assert.Equal(t, pod.ID, controller.PodID)
	assert.Equal(t, pod.RunnerID, controller.RunnerID)
	assert.Equal(t, "Review the code", controller.Prompt)
	assert.Equal(t, agentpod.AutopilotPhaseInitializing, controller.Phase)
	assert.Equal(t, agentpod.CircuitBreakerClosed, controller.CircuitBreakerState)
	assert.True(t, controller.ID > 0, "should have auto-generated ID")

	// Verify key format: "autopilot-{podKey}-{nanoTimestamp}"
	assert.True(t, strings.HasPrefix(controller.AutopilotControllerKey, "autopilot-pod-abc123-"))

	// Verify gRPC command was sent
	assert.True(t, sender.called)
	assert.Equal(t, pod.RunnerID, sender.runnerID)
	assert.Equal(t, controller.AutopilotControllerKey, sender.cmd.AutopilotKey)
	assert.Equal(t, pod.PodKey, sender.cmd.PodKey)
	assert.Equal(t, "Review the code", sender.cmd.Config.Prompt)
}

func TestCreateAndStart_NilPod(t *testing.T) {
	db := setupAutopilotTestDB(t)
	svc := newTestAutopilotService(db)

	controller, err := svc.CreateAndStart(context.Background(), &CreateAndStartRequest{
		OrganizationID: 1,
		Pod:            nil,
		Prompt:  "test",
	})

	assert.Nil(t, controller)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "target pod is required")
}

func TestCreateAndStart_DefaultValues(t *testing.T) {
	db := setupAutopilotTestDB(t)
	svc := newTestAutopilotService(db)
	// No command sender — verifies nil sender is safe

	controller, err := svc.CreateAndStart(context.Background(), &CreateAndStartRequest{
		OrganizationID: 1,
		Pod:            newTestPod(),
		Prompt:  "test",
		// All config fields left at zero → should get domain defaults
	})

	require.NoError(t, err)
	assert.Equal(t, agentpod.DefaultMaxIterations, controller.MaxIterations)
	assert.Equal(t, agentpod.DefaultIterationTimeoutSec, controller.IterationTimeoutSec)
	assert.Equal(t, agentpod.DefaultNoProgressThreshold, controller.NoProgressThreshold)
	assert.Equal(t, agentpod.DefaultSameErrorThreshold, controller.SameErrorThreshold)
	assert.Equal(t, agentpod.DefaultApprovalTimeoutMin, controller.ApprovalTimeoutMin)
}

func TestCreateAndStart_CustomValues(t *testing.T) {
	db := setupAutopilotTestDB(t)
	sender := &mockCommandSender{}
	svc := newTestAutopilotService(db)
	svc.SetCommandSender(sender)

	controller, err := svc.CreateAndStart(context.Background(), &CreateAndStartRequest{
		OrganizationID:      1,
		Pod:                 newTestPod(),
		Prompt:              "test",
		MaxIterations:       25,
		IterationTimeoutSec: 600,
		NoProgressThreshold: 5,
		SameErrorThreshold:  10,
		ApprovalTimeoutMin:  60,
	})

	require.NoError(t, err)
	assert.Equal(t, int32(25), controller.MaxIterations)
	assert.Equal(t, int32(600), controller.IterationTimeoutSec)
	assert.Equal(t, int32(5), controller.NoProgressThreshold)
	assert.Equal(t, int32(10), controller.SameErrorThreshold)
	assert.Equal(t, int32(60), controller.ApprovalTimeoutMin)

	// Verify gRPC command also carries custom values
	assert.Equal(t, int32(25), sender.cmd.Config.MaxIterations)
	assert.Equal(t, int32(600), sender.cmd.Config.IterationTimeoutSeconds)
}

func TestCreateAndStart_CustomKeyPrefix(t *testing.T) {
	db := setupAutopilotTestDB(t)
	svc := newTestAutopilotService(db)

	controller, err := svc.CreateAndStart(context.Background(), &CreateAndStartRequest{
		OrganizationID: 1,
		Pod:            newTestPod(),
		Prompt:  "test",
		KeyPrefix:      "loop-daily-review-run3",
	})

	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(controller.AutopilotControllerKey, "loop-daily-review-run3-pod-abc123-"))
}

func TestCreateAndStart_DefaultKeyPrefix(t *testing.T) {
	db := setupAutopilotTestDB(t)
	svc := newTestAutopilotService(db)

	controller, err := svc.CreateAndStart(context.Background(), &CreateAndStartRequest{
		OrganizationID: 1,
		Pod:            newTestPod(),
		Prompt:  "test",
		// KeyPrefix empty → defaults to "autopilot"
	})

	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(controller.AutopilotControllerKey, "autopilot-"))
}

func TestCreateAndStart_OptionalConfigFields(t *testing.T) {
	db := setupAutopilotTestDB(t)
	sender := &mockCommandSender{}
	svc := newTestAutopilotService(db)
	svc.SetCommandSender(sender)

	controller, err := svc.CreateAndStart(context.Background(), &CreateAndStartRequest{
		OrganizationID:        1,
		Pod:                   newTestPod(),
		Prompt:                "test",
		ControlAgentSlug:      "custom-agent",
		ControlPromptTemplate: "You are a reviewer...",
		MCPConfigJSON:         `{"servers":["s1"]}`,
	})

	require.NoError(t, err)
	require.NotNil(t, controller.ControlAgentSlug)
	assert.Equal(t, "custom-agent", *controller.ControlAgentSlug)
	require.NotNil(t, controller.ControlPromptTemplate)
	assert.Equal(t, "You are a reviewer...", *controller.ControlPromptTemplate)
	require.NotNil(t, controller.MCPConfigJSON)
	assert.Equal(t, `{"servers":["s1"]}`, *controller.MCPConfigJSON)

	// Verify gRPC command carries optional fields
	assert.Equal(t, "custom-agent", sender.cmd.Config.ControlAgentSlug)
	assert.Equal(t, "You are a reviewer...", sender.cmd.Config.ControlPromptTemplate)
	assert.Equal(t, `{"servers":["s1"]}`, sender.cmd.Config.McpConfigJson)
}

func TestCreateAndStart_OptionalFieldsOmitted(t *testing.T) {
	db := setupAutopilotTestDB(t)
	svc := newTestAutopilotService(db)

	controller, err := svc.CreateAndStart(context.Background(), &CreateAndStartRequest{
		OrganizationID: 1,
		Pod:            newTestPod(),
		Prompt:  "test",
		// No optional config fields
	})

	require.NoError(t, err)
	assert.Nil(t, controller.ControlAgentSlug)
	assert.Nil(t, controller.ControlPromptTemplate)
	assert.Nil(t, controller.MCPConfigJSON)
}

func TestCreateAndStart_NilCommandSender(t *testing.T) {
	db := setupAutopilotTestDB(t)
	svc := newTestAutopilotService(db)
	// commandSender is nil (not set)

	controller, err := svc.CreateAndStart(context.Background(), &CreateAndStartRequest{
		OrganizationID: 1,
		Pod:            newTestPod(),
		Prompt:  "test",
	})

	// Should succeed — DB record created, gRPC skipped
	require.NoError(t, err)
	assert.True(t, controller.ID > 0)
}

func TestCreateAndStart_CommandSenderFailure(t *testing.T) {
	db := setupAutopilotTestDB(t)
	sender := &mockCommandSender{err: assert.AnError}
	svc := newTestAutopilotService(db)
	svc.SetCommandSender(sender)

	controller, err := svc.CreateAndStart(context.Background(), &CreateAndStartRequest{
		OrganizationID: 1,
		Pod:            newTestPod(),
		Prompt:  "test",
	})

	// Error returned, BUT controller is also returned (DB record exists)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "autopilot created in DB but failed to send command to runner")
	require.NotNil(t, controller, "controller should be returned even on gRPC failure")
	assert.True(t, controller.ID > 0, "DB record should exist")
}

func TestCreateAndStart_DBPersistence(t *testing.T) {
	db := setupAutopilotTestDB(t)
	svc := newTestAutopilotService(db)

	created, err := svc.CreateAndStart(context.Background(), &CreateAndStartRequest{
		OrganizationID: 1,
		Pod:            newTestPod(),
		Prompt:  "persisted prompt",
	})
	require.NoError(t, err)

	// Verify we can retrieve the record from DB
	fetched, err := svc.GetAutopilotControllerByKey(context.Background(), created.AutopilotControllerKey)
	require.NoError(t, err)
	assert.Equal(t, created.ID, fetched.ID)
	assert.Equal(t, "persisted prompt", fetched.Prompt)
	assert.Equal(t, agentpod.AutopilotPhaseInitializing, fetched.Phase)
	assert.Equal(t, agentpod.CircuitBreakerClosed, fetched.CircuitBreakerState)
}
