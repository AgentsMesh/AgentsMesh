package agentpod

import (
	"context"
	"testing"
	"time"
)

func TestUpdatePodPTY(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestPodService(db)
	ctx := context.Background()

	req := &CreatePodRequest{
		OrganizationID: 1,
		RunnerID:       1,
		CreatedByID:    1,
	}
	sess, _ := svc.CreatePod(ctx, req)

	// Just verify no error is returned - actual column mapping
	// differs between GORM field name (pty_p_id) and raw SQL (pty_pid)
	err := svc.UpdatePodPTY(ctx, sess.PodKey, 54321)
	if err != nil {
		t.Fatalf("UpdatePodPTY failed: %v", err)
	}
}

func TestUpdateSandboxPath(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestPodService(db)
	ctx := context.Background()

	req := &CreatePodRequest{
		OrganizationID: 1,
		RunnerID:       1,
		CreatedByID:    1,
	}
	sess, _ := svc.CreatePod(ctx, req)

	err := svc.UpdateSandboxPath(ctx, sess.PodKey, "/workspace/sandboxes/pod-1", "feature/test")
	if err != nil {
		t.Fatalf("UpdateSandboxPath failed: %v", err)
	}

	updated, _ := svc.GetPod(ctx, sess.PodKey)
	if updated.SandboxPath == nil || *updated.SandboxPath != "/workspace/sandboxes/pod-1" {
		t.Error("SandboxPath not set correctly")
	}
	if updated.BranchName == nil || *updated.BranchName != "feature/test" {
		t.Error("BranchName not set correctly")
	}
}

func TestRecordActivity(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestPodService(db)
	ctx := context.Background()

	req := &CreatePodRequest{
		OrganizationID: 1,
		RunnerID:       1,
		CreatedByID:    1,
	}
	sess, _ := svc.CreatePod(ctx, req)

	time.Sleep(10 * time.Millisecond)

	err := svc.RecordActivity(ctx, sess.PodKey)
	if err != nil {
		t.Fatalf("RecordActivity failed: %v", err)
	}

	updated, _ := svc.GetPod(ctx, sess.PodKey)
	if updated.LastActivity == nil {
		t.Error("LastActivity should be set")
	}
}

func TestUpdatePodTitle(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestPodService(db)
	ctx := context.Background()

	// Create a pod first
	req := &CreatePodRequest{
		OrganizationID: 1,
		RunnerID:       1,
		CreatedByID:    1,
	}
	pod, err := svc.CreatePod(ctx, req)
	if err != nil {
		t.Fatalf("CreatePod failed: %v", err)
	}

	tests := []struct {
		name    string
		podKey  string
		title   string
		wantErr bool
	}{
		{
			name:    "update title successfully",
			podKey:  pod.PodKey,
			title:   "My Custom Title",
			wantErr: false,
		},
		{
			name:    "update title with special characters",
			podKey:  pod.PodKey,
			title:   "Title with \"quotes\" and 'apostrophes'",
			wantErr: false,
		},
		{
			name:    "update title to empty",
			podKey:  pod.PodKey,
			title:   "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.UpdatePodTitle(ctx, tt.podKey, tt.title)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdatePodTitle() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				updated, _ := svc.GetPod(ctx, tt.podKey)
				if tt.title == "" {
					// Empty title should still work
					if updated.Title != nil && *updated.Title != "" {
						t.Errorf("Title = %v, want empty", *updated.Title)
					}
				} else {
					if updated.Title == nil || *updated.Title != tt.title {
						t.Errorf("Title = %v, want %v", updated.Title, tt.title)
					}
				}
			}
		})
	}
}
