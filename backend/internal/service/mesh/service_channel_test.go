package mesh

import (
	"context"
	"testing"
)

func TestGetChannelPods(t *testing.T) {
	repo, db := setupTestRepo(t)
	service := NewService(repo, nil, nil, nil)
	ctx := context.Background()

	// Add some channel pods
	db.Exec(`INSERT INTO channel_pods (channel_id, pod_key) VALUES (1, 'pod-1'), (1, 'pod-2'), (2, 'pod-3')`)

	keys := service.getChannelPods(ctx, 1)
	if len(keys) != 2 {
		t.Errorf("len(keys) = %d, want 2", len(keys))
	}
}

func TestGetChannelMessageCount(t *testing.T) {
	repo, db := setupTestRepo(t)
	service := NewService(repo, nil, nil, nil)
	ctx := context.Background()

	// Add some messages
	db.Exec(`INSERT INTO channel_messages (channel_id, message_type, content) VALUES (1, 'text', 'msg1'), (1, 'text', 'msg2'), (1, 'text', 'msg3'), (2, 'text', 'msg4')`)

	count := service.getChannelMessageCount(ctx, 1)
	if count != 3 {
		t.Errorf("count = %d, want 3", count)
	}
}

func TestJoinChannel(t *testing.T) {
	repo, db := setupTestRepo(t)
	service := NewService(repo, nil, nil, nil)
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
	repo, db := setupTestRepo(t)
	service := NewService(repo, nil, nil, nil)
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
	repo, db := setupTestRepo(t)
	service := NewService(repo, nil, nil, nil)
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

func TestGetChannelPods_Empty(t *testing.T) {
	repo, _ := setupTestRepo(t)
	service := NewService(repo, nil, nil, nil)
	ctx := context.Background()

	keys := service.getChannelPods(ctx, 999)
	if len(keys) != 0 {
		t.Errorf("expected 0 keys for non-existent channel, got %d", len(keys))
	}
}

func TestGetChannelMessageCount_Empty(t *testing.T) {
	repo, _ := setupTestRepo(t)
	service := NewService(repo, nil, nil, nil)
	ctx := context.Background()

	count := service.getChannelMessageCount(ctx, 999)
	if count != 0 {
		t.Errorf("expected 0 messages for non-existent channel, got %d", count)
	}
}

func TestJoinChannel_Duplicate(t *testing.T) {
	repo, db := setupTestRepo(t)
	service := NewService(repo, nil, nil, nil)
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
	repo, _ := setupTestRepo(t)
	service := NewService(repo, nil, nil, nil)
	ctx := context.Background()

	// Leave a channel we never joined - should not error
	err := service.LeaveChannel(ctx, 999, "nonexistent")
	if err != nil {
		t.Errorf("LeaveChannel() should not error for non-existent: %v", err)
	}
}

func TestRecordChannelAccess_NilPod(t *testing.T) {
	repo, db := setupTestRepo(t)
	service := NewService(repo, nil, nil, nil)
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
	repo, db := setupTestRepo(t)
	service := NewService(repo, nil, nil, nil)
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
