package agentpod

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
)

func TestCreatePod(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestPodService(db)
	ctx := context.Background()

	tests := []struct {
		name    string
		req     *CreatePodRequest
		wantErr bool
	}{
		{
			name: "basic pod",
			req: &CreatePodRequest{
				OrganizationID: 1,
				RunnerID:       1,
				CreatedByID:    1,
				Prompt:         "Test prompt",
			},
			wantErr: false,
		},
		{
			name: "pod with all options",
			req: &CreatePodRequest{
				OrganizationID: 1,
				RunnerID:       1,
				CreatedByID:    1,
				Prompt:         "Test prompt",
				Model:          "sonnet",
				PermissionMode: "default",
			},
			wantErr: false,
		},
		{
			name: "pod with ticket",
			req: &CreatePodRequest{
				OrganizationID: 1,
				RunnerID:       1,
				CreatedByID:    1,
				TicketID:       intPtr(42),
				Prompt:         "Working on ticket",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sess, err := svc.CreatePod(ctx, tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreatePod() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if sess == nil {
					t.Error("Pod is nil")
					return
				}
				if sess.ID == 0 {
					t.Error("Pod ID should be set")
				}
				if sess.PodKey == "" {
					t.Error("PodKey should not be empty")
				}
				if sess.Status != agentpod.StatusInitializing {
					t.Errorf("Status = %s, want initializing", sess.Status)
				}
			}
		})
	}
}

func TestCreatePod_CredentialProfileID(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestPodService(db)
	ctx := context.Background()

	t.Run("stores credential_profile_id when provided", func(t *testing.T) {
		profileID := int64(42)
		pod, err := svc.CreatePod(ctx, &CreatePodRequest{
			OrganizationID:      1,
			RunnerID:            1,
			CreatedByID:         1,
			CredentialProfileID: &profileID,
		})
		if err != nil {
			t.Fatalf("CreatePod failed: %v", err)
		}
		if pod.CredentialProfileID == nil {
			t.Fatal("CredentialProfileID should not be nil")
		}
		if *pod.CredentialProfileID != 42 {
			t.Errorf("CredentialProfileID = %d, want 42", *pod.CredentialProfileID)
		}
	})

	t.Run("stores nil credential_profile_id when not provided", func(t *testing.T) {
		pod, err := svc.CreatePod(ctx, &CreatePodRequest{
			OrganizationID:      1,
			RunnerID:            1,
			CreatedByID:         1,
			CredentialProfileID: nil,
		})
		if err != nil {
			t.Fatalf("CreatePod failed: %v", err)
		}
		if pod.CredentialProfileID != nil {
			t.Errorf("CredentialProfileID should be nil, got %d", *pod.CredentialProfileID)
		}
	})
}

func TestCreatePod_Alias(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestPodService(db)
	ctx := context.Background()

	t.Run("stores alias when provided", func(t *testing.T) {
		alias := "my-feature-pod"
		pod, err := svc.CreatePod(ctx, &CreatePodRequest{
			OrganizationID: 1,
			RunnerID:       1,
			CreatedByID:    1,
			Alias:          &alias,
		})
		if err != nil {
			t.Fatalf("CreatePod failed: %v", err)
		}
		if pod.Alias == nil {
			t.Fatal("Alias should not be nil")
		}
		if *pod.Alias != "my-feature-pod" {
			t.Errorf("Alias = %q, want %q", *pod.Alias, "my-feature-pod")
		}

		// Verify persisted to DB via GetPod
		fetched, err := svc.GetPod(ctx, pod.PodKey)
		if err != nil {
			t.Fatalf("GetPod failed: %v", err)
		}
		if fetched.Alias == nil || *fetched.Alias != "my-feature-pod" {
			t.Errorf("Persisted Alias = %v, want %q", fetched.Alias, "my-feature-pod")
		}
	})

	t.Run("stores nil alias when not provided", func(t *testing.T) {
		pod, err := svc.CreatePod(ctx, &CreatePodRequest{
			OrganizationID: 1,
			RunnerID:       1,
			CreatedByID:    1,
			Alias:          nil,
		})
		if err != nil {
			t.Fatalf("CreatePod failed: %v", err)
		}
		if pod.Alias != nil {
			t.Errorf("Alias should be nil, got %q", *pod.Alias)
		}
	})
}

func TestCreatePod_DefaultValues(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestPodService(db)
	ctx := context.Background()

	req := &CreatePodRequest{
		OrganizationID: 1,
		RunnerID:       1,
		CreatedByID:    1,
	}

	sess, err := svc.CreatePod(ctx, req)
	if err != nil {
		t.Fatalf("CreatePod failed: %v", err)
	}

	// Check defaults
	if sess.Model == nil || *sess.Model != "opus" {
		t.Error("Default model should be opus")
	}
	if sess.PermissionMode == nil || *sess.PermissionMode != agentpod.PermissionModePlan {
		t.Error("Default permission mode should be plan")
	}
}

func TestGetPod(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestPodService(db)
	ctx := context.Background()

	// Create a pod first
	req := &CreatePodRequest{
		OrganizationID: 1,
		RunnerID:       1,
		CreatedByID:    1,
	}
	created, err := svc.CreatePod(ctx, req)
	if err != nil {
		t.Fatalf("Failed to create pod: %v", err)
	}

	t.Run("existing pod", func(t *testing.T) {
		sess, err := svc.GetPod(ctx, created.PodKey)
		if err != nil {
			t.Errorf("GetPod failed: %v", err)
		}
		if sess.ID != created.ID {
			t.Errorf("Pod ID = %d, want %d", sess.ID, created.ID)
		}
	})

	t.Run("non-existent pod", func(t *testing.T) {
		_, err := svc.GetPod(ctx, "non-existent-key")
		if err == nil {
			t.Error("Expected error for non-existent pod")
		}
		if err != ErrPodNotFound {
			t.Errorf("Expected ErrPodNotFound, got %v", err)
		}
	})
}

func TestGetPodByID(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestPodService(db)
	ctx := context.Background()

	// Create a pod first
	req := &CreatePodRequest{
		OrganizationID: 1,
		RunnerID:       1,
		CreatedByID:    1,
	}
	created, _ := svc.CreatePod(ctx, req)

	t.Run("existing pod", func(t *testing.T) {
		sess, err := svc.GetPodByID(ctx, created.ID)
		if err != nil {
			t.Errorf("GetPodByID failed: %v", err)
		}
		if sess.PodKey != created.PodKey {
			t.Errorf("PodKey mismatch")
		}
	})

	t.Run("non-existent pod", func(t *testing.T) {
		_, err := svc.GetPodByID(ctx, 99999)
		if err == nil {
			t.Error("Expected error for non-existent pod")
		}
	})
}

func TestGetPodOrganizationAndCreator(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestPodService(db)
	ctx := context.Background()

	// Create a pod first
	req := &CreatePodRequest{
		OrganizationID: 42,
		RunnerID:       1,
		CreatedByID:    99,
	}
	pod, err := svc.CreatePod(ctx, req)
	if err != nil {
		t.Fatalf("CreatePod failed: %v", err)
	}

	tests := []struct {
		name          string
		podKey        string
		wantOrgID     int64
		wantCreatorID int64
		wantErr       bool
	}{
		{
			name:          "existing pod",
			podKey:        pod.PodKey,
			wantOrgID:     42,
			wantCreatorID: 99,
			wantErr:       false,
		},
		{
			name:          "non-existent pod",
			podKey:        "non-existent-pod-key",
			wantOrgID:     0,
			wantCreatorID: 0,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orgID, creatorID, err := svc.GetPodOrganizationAndCreator(ctx, tt.podKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetPodOrganizationAndCreator() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if orgID != tt.wantOrgID {
				t.Errorf("orgID = %v, want %v", orgID, tt.wantOrgID)
			}
			if creatorID != tt.wantCreatorID {
				t.Errorf("creatorID = %v, want %v", creatorID, tt.wantCreatorID)
			}
		})
	}
}
