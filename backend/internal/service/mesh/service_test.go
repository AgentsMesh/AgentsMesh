package mesh

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	// Create tables
	db.Exec(`CREATE TABLE IF NOT EXISTS pods (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		pod_key TEXT NOT NULL UNIQUE,
		organization_id INTEGER NOT NULL,
		ticket_id INTEGER,
		repository_id INTEGER,
		runner_id INTEGER,
		agent_type_id INTEGER,
		custom_agent_type_id INTEGER,
		created_by_id INTEGER NOT NULL,
		pty_pid INTEGER,
		status TEXT NOT NULL DEFAULT 'pending',
		agent_status TEXT NOT NULL DEFAULT 'unknown',
		agent_pid INTEGER,
		started_at DATETIME,
		finished_at DATETIME,
		last_activity DATETIME,
		initial_prompt TEXT NOT NULL DEFAULT '',
		branch_name TEXT,
		worktree_path TEXT,
		model TEXT,
		permission_mode TEXT,
		think_level TEXT,
		title TEXT,
		config_overrides TEXT DEFAULT '{}',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)

	db.Exec(`CREATE TABLE IF NOT EXISTS channel_pods (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		channel_id INTEGER NOT NULL,
		pod_key TEXT NOT NULL,
		joined_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)

	db.Exec(`CREATE TABLE IF NOT EXISTS channel_messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		channel_id INTEGER NOT NULL,
		content TEXT NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)

	db.Exec(`CREATE TABLE IF NOT EXISTS channel_access (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		channel_id INTEGER NOT NULL,
		pod_key TEXT,
		user_id INTEGER,
		last_access DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)

	return db
}

func TestNewService(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db, nil, nil, nil)

	if service == nil {
		t.Fatal("expected non-nil service")
	}
	if service.db != db {
		t.Error("expected service.db to be the provided db")
	}
}

func TestPodToNode(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db, nil, nil, nil)

	ticketID := int64(100)
	repoID := int64(200)
	model := "claude-3-sonnet"
	pod := &agentpod.Pod{
		PodKey:       "test-pod-key",
		Status:       "running",
		AgentStatus:  "working",
		Model:        &model,
		TicketID:     &ticketID,
		RepositoryID: &repoID,
		CreatedByID:  1,
		RunnerID:     5,
	}

	node := service.podToNode(pod)

	if node.PodKey != "test-pod-key" {
		t.Errorf("PodKey = %s, want test-pod-key", node.PodKey)
	}
	if node.Status != "running" {
		t.Errorf("Status = %s, want running", node.Status)
	}
	if node.AgentStatus != "working" {
		t.Errorf("AgentStatus = %s, want working", node.AgentStatus)
	}
	if node.Model == nil || *node.Model != "claude-3-sonnet" {
		t.Errorf("Model mismatch")
	}
	if node.TicketID == nil || *node.TicketID != 100 {
		t.Error("TicketID mismatch")
	}
	if node.RepositoryID == nil || *node.RepositoryID != 200 {
		t.Error("RepositoryID mismatch")
	}
}

func TestGetChannelPods(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db, nil, nil, nil)
	ctx := context.Background()

	// Add some channel pods
	db.Exec(`INSERT INTO channel_pods (channel_id, pod_key) VALUES (1, 'pod-1'), (1, 'pod-2'), (2, 'pod-3')`)

	keys := service.getChannelPods(ctx, 1)
	if len(keys) != 2 {
		t.Errorf("len(keys) = %d, want 2", len(keys))
	}
}

func TestGetChannelMessageCount(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db, nil, nil, nil)
	ctx := context.Background()

	// Add some messages
	db.Exec(`INSERT INTO channel_messages (channel_id, content) VALUES (1, 'msg1'), (1, 'msg2'), (1, 'msg3'), (2, 'msg4')`)

	count := service.getChannelMessageCount(ctx, 1)
	if count != 3 {
		t.Errorf("count = %d, want 3", count)
	}
}

func TestJoinChannel(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db, nil, nil, nil)
	ctx := context.Background()

	err := service.JoinChannel(ctx, 1, "test-pod")
	if err != nil {
		t.Fatalf("JoinChannel() error = %v", err)
	}

	// Verify
	var cp ChannelPod
	if err := db.First(&cp).Error; err != nil {
		t.Fatalf("failed to find channel pod: %v", err)
	}
	if cp.PodKey != "test-pod" {
		t.Errorf("PodKey = %s, want test-pod", cp.PodKey)
	}
}

func TestLeaveChannel(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db, nil, nil, nil)
	ctx := context.Background()

	// Join first
	service.JoinChannel(ctx, 1, "test-pod")

	// Leave
	err := service.LeaveChannel(ctx, 1, "test-pod")
	if err != nil {
		t.Fatalf("LeaveChannel() error = %v", err)
	}

	// Verify removed
	var count int64
	db.Model(&ChannelPod{}).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 channel pods, got %d", count)
	}
}

func TestRecordChannelAccess(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db, nil, nil, nil)
	ctx := context.Background()

	podKey := "test-pod"
	userID := int64(1)

	err := service.RecordChannelAccess(ctx, 1, &podKey, &userID)
	if err != nil {
		t.Fatalf("RecordChannelAccess() error = %v", err)
	}

	// Verify
	var ca ChannelAccess
	if err := db.First(&ca).Error; err != nil {
		t.Fatalf("failed to find channel access: %v", err)
	}
	if ca.ChannelID != 1 {
		t.Errorf("ChannelID = %d, want 1", ca.ChannelID)
	}
}

func TestBatchGetTicketPods(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db, nil, nil, nil)
	ctx := context.Background()

	// Create pods
	ticketID1 := int64(100)
	ticketID2 := int64(200)

	db.Create(&agentpod.Pod{PodKey: "pod-1", OrganizationID: 1, TicketID: &ticketID1, Status: "running", CreatedByID: 1})
	db.Create(&agentpod.Pod{PodKey: "pod-2", OrganizationID: 1, TicketID: &ticketID1, Status: "terminated", CreatedByID: 1})
	db.Create(&agentpod.Pod{PodKey: "pod-3", OrganizationID: 1, TicketID: &ticketID2, Status: "running", CreatedByID: 1})

	result, err := service.BatchGetTicketPods(ctx, []int64{100, 200, 300})
	if err != nil {
		t.Fatalf("BatchGetTicketPods() error = %v", err)
	}

	if len(result.TicketPods[100]) != 2 {
		t.Errorf("ticket 100 pods = %d, want 2", len(result.TicketPods[100]))
	}
	if len(result.TicketPods[200]) != 1 {
		t.Errorf("ticket 200 pods = %d, want 1", len(result.TicketPods[200]))
	}
	if len(result.TicketPods[300]) != 0 {
		t.Errorf("ticket 300 pods = %d, want 0", len(result.TicketPods[300]))
	}
}

func TestErrorVariables(t *testing.T) {
	if ErrTicketNotFound == nil {
		t.Error("ErrTicketNotFound should not be nil")
	}
	if ErrRunnerNotFound == nil {
		t.Error("ErrRunnerNotFound should not be nil")
	}
}

// ChannelPod for testing (local type to avoid import cycle)
type ChannelPod struct {
	ID        int64  `gorm:"primaryKey"`
	ChannelID int64  `gorm:"not null"`
	PodKey    string `gorm:"not null"`
}

func (ChannelPod) TableName() string {
	return "channel_pods"
}

// ChannelAccess for testing
type ChannelAccess struct {
	ID        int64   `gorm:"primaryKey"`
	ChannelID int64   `gorm:"not null"`
	PodKey    *string
	UserID    *int64
}

func (ChannelAccess) TableName() string {
	return "channel_access"
}

// Mock the channel.Message for count query
func init() {
	// Register table name mapping if needed
}

func TestPodToNode_NilValues(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db, nil, nil, nil)

	// Test with minimal pod (nil optional fields)
	pod := &agentpod.Pod{
		PodKey:      "minimal-pod",
		Status:      "pending",
		AgentStatus: "unknown",
		CreatedByID: 1,
	}

	node := service.podToNode(pod)

	if node.PodKey != "minimal-pod" {
		t.Errorf("PodKey = %s, want minimal-pod", node.PodKey)
	}
	if node.Model != nil {
		t.Error("expected Model to be nil")
	}
	if node.TicketID != nil {
		t.Error("expected TicketID to be nil")
	}
	if node.RepositoryID != nil {
		t.Error("expected RepositoryID to be nil")
	}
}

func TestGetChannelPods_Empty(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db, nil, nil, nil)
	ctx := context.Background()

	keys := service.getChannelPods(ctx, 999)
	if len(keys) != 0 {
		t.Errorf("expected 0 keys for non-existent channel, got %d", len(keys))
	}
}

func TestGetChannelMessageCount_Empty(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db, nil, nil, nil)
	ctx := context.Background()

	count := service.getChannelMessageCount(ctx, 999)
	if count != 0 {
		t.Errorf("expected 0 messages for non-existent channel, got %d", count)
	}
}

func TestJoinChannel_Duplicate(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db, nil, nil, nil)
	ctx := context.Background()

	// Join twice - should create two records (no unique constraint in test)
	service.JoinChannel(ctx, 1, "test-pod")
	err := service.JoinChannel(ctx, 1, "test-pod")
	if err != nil {
		t.Fatalf("JoinChannel() duplicate error = %v", err)
	}

	var count int64
	db.Model(&ChannelPod{}).Where("channel_id = ?", 1).Count(&count)
	if count != 2 {
		t.Errorf("expected 2 channel pods, got %d", count)
	}
}

func TestLeaveChannel_NotExists(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db, nil, nil, nil)
	ctx := context.Background()

	// Leave a channel we never joined - should not error
	err := service.LeaveChannel(ctx, 999, "nonexistent")
	if err != nil {
		t.Errorf("LeaveChannel() should not error for non-existent: %v", err)
	}
}

func TestRecordChannelAccess_NilPod(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db, nil, nil, nil)
	ctx := context.Background()

	userID := int64(1)
	err := service.RecordChannelAccess(ctx, 1, nil, &userID)
	if err != nil {
		t.Fatalf("RecordChannelAccess() error = %v", err)
	}

	var ca ChannelAccess
	db.First(&ca)
	if ca.PodKey != nil {
		t.Error("expected PodKey to be nil")
	}
	if ca.UserID == nil || *ca.UserID != 1 {
		t.Error("expected UserID to be 1")
	}
}

func TestRecordChannelAccess_NilUser(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db, nil, nil, nil)
	ctx := context.Background()

	podKey := "pod-key"
	err := service.RecordChannelAccess(ctx, 2, &podKey, nil)
	if err != nil {
		t.Fatalf("RecordChannelAccess() error = %v", err)
	}

	var ca ChannelAccess
	db.Where("channel_id = ?", 2).First(&ca)
	if ca.PodKey == nil || *ca.PodKey != "pod-key" {
		t.Error("expected PodKey to be 'pod-key'")
	}
	if ca.UserID != nil {
		t.Error("expected UserID to be nil")
	}
}

func TestBatchGetTicketPods_Empty(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db, nil, nil, nil)
	ctx := context.Background()

	result, err := service.BatchGetTicketPods(ctx, []int64{})
	if err != nil {
		t.Fatalf("BatchGetTicketPods() error = %v", err)
	}
	if len(result.TicketPods) != 0 {
		t.Errorf("expected 0 entries, got %d", len(result.TicketPods))
	}
}

func TestBatchGetTicketPods_NoPods(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db, nil, nil, nil)
	ctx := context.Background()

	result, err := service.BatchGetTicketPods(ctx, []int64{1, 2, 3})
	if err != nil {
		t.Fatalf("BatchGetTicketPods() error = %v", err)
	}
	// Should have entries for all requested IDs even if empty
	for _, id := range []int64{1, 2, 3} {
		if _, ok := result.TicketPods[id]; !ok {
			t.Errorf("expected entry for ticket ID %d", id)
		}
		if len(result.TicketPods[id]) != 0 {
			t.Errorf("expected 0 pods for ticket %d, got %d", id, len(result.TicketPods[id]))
		}
	}
}

func TestBatchGetTicketPods_PodWithNilTicket(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db, nil, nil, nil)
	ctx := context.Background()

	// Create pod with nil ticket_id
	db.Create(&agentpod.Pod{PodKey: "no-ticket", OrganizationID: 1, Status: "running", CreatedByID: 1})

	result, err := service.BatchGetTicketPods(ctx, []int64{100})
	if err != nil {
		t.Fatalf("BatchGetTicketPods() error = %v", err)
	}
	// Pod with nil ticket_id should not be included
	if len(result.TicketPods[100]) != 0 {
		t.Errorf("expected 0 pods for ticket 100, got %d", len(result.TicketPods[100]))
	}
}

func TestServiceFields(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db, nil, nil, nil)

	// Verify nil services are accepted
	if service.podService != nil {
		t.Error("expected podService to be nil")
	}
	if service.channelService != nil {
		t.Error("expected channelService to be nil")
	}
	if service.bindingService != nil {
		t.Error("expected bindingService to be nil")
	}
}
