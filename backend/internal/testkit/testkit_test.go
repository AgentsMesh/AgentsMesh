package testkit_test

import (
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/testkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupTestDB_CreatesAllTables(t *testing.T) {
	db := testkit.SetupTestDB(t)

	// Verify key tables exist by counting them
	var count int64
	tables := []string{
		"users", "user_identities", "user_git_credentials", "user_repository_providers",
		"organizations", "organization_members", "agents", "repositories", "git_providers",
		"runners", "runner_certificates", "runner_logs", "pods",
		"autopilot_controllers", "autopilot_iterations",
		"channels", "channel_messages", "channel_members",
		"tickets", "ticket_comments", "ticket_merge_requests",
		"loops", "loop_runs",
		"subscription_plans", "subscriptions", "payment_orders", "licenses",
		"api_keys", "invitations", "sso_configs", "support_tickets",
		"token_usages", "custom_agents",
		"agent_messages", "agent_message_dead_letters",
	}

	for _, table := range tables {
		err := db.Raw("SELECT COUNT(*) FROM " + table).Scan(&count).Error
		require.NoError(t, err, "table %s should exist", table)
	}
}

func TestFactory_CreateUserAndOrg(t *testing.T) {
	db := testkit.SetupTestDB(t)

	userID := testkit.CreateUser(t, db, "test@example.com", "testuser")
	assert.Greater(t, userID, int64(0))

	orgID := testkit.CreateOrg(t, db, "test-org", userID)
	assert.Greater(t, orgID, int64(0))

	// Verify org member
	var role string
	db.Raw("SELECT role FROM organization_members WHERE organization_id = ? AND user_id = ?", orgID, userID).Scan(&role)
	assert.Equal(t, "owner", role)
}

func TestFactory_CreateRunner(t *testing.T) {
	db := testkit.SetupTestDB(t)
	userID := testkit.CreateUser(t, db, "u@e.com", "u")
	orgID := testkit.CreateOrg(t, db, "org1", userID)

	runnerID := testkit.CreateRunner(t, db, orgID, "node-001")
	assert.Greater(t, runnerID, int64(0))
}

func TestFactory_CreatePod(t *testing.T) {
	db := testkit.SetupTestDB(t)
	userID := testkit.CreateUser(t, db, "u@e.com", "u")
	orgID := testkit.CreateOrg(t, db, "org1", userID)
	runnerID := testkit.CreateRunner(t, db, orgID, "node-001")

	podKey := testkit.CreatePod(t, db, orgID, runnerID, userID)
	assert.NotEmpty(t, podKey)
}

func TestCaptureEventBus(t *testing.T) {
	bus := testkit.NewCaptureEventBus()

	bus.Publish("pod.created", map[string]string{"key": "pod-1"})
	bus.Publish("pod.created", map[string]string{"key": "pod-2"})
	bus.Publish("channel.message", "hello")

	assert.True(t, bus.HasEvent("pod.created"))
	assert.Equal(t, 2, bus.EventCount("pod.created"))
	assert.Equal(t, 1, bus.EventCount("channel.message"))
	assert.False(t, bus.HasEvent("nonexistent"))

	bus.Reset()
	assert.Equal(t, 0, bus.EventCount("pod.created"))
}
