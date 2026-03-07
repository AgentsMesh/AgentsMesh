package ticket

import (
	"context"
	"testing"
)

func TestLabel_CRUD(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	t.Run("create and get", func(t *testing.T) {
		label, err := service.CreateLabel(ctx, 1, nil, "critical", "#FF0000")
		if err != nil {
			t.Fatalf("CreateLabel() error = %v", err)
		}

		got, err := service.GetLabel(ctx, label.ID)
		if err != nil {
			t.Fatalf("GetLabel() error = %v", err)
		}
		if got.Name != "critical" {
			t.Errorf("Name = %s, want critical", got.Name)
		}
	})

	t.Run("update", func(t *testing.T) {
		label, _ := service.CreateLabel(ctx, 1, nil, "minor", "#808080")

		updated, err := service.UpdateLabel(ctx, 1, label.ID, map[string]interface{}{
			"name":  "major",
			"color": "#FF6600",
		})
		if err != nil {
			t.Fatalf("UpdateLabel() error = %v", err)
		}
		if updated.Name != "major" {
			t.Errorf("Name = %s, want major", updated.Name)
		}
	})

	t.Run("delete", func(t *testing.T) {
		label, _ := service.CreateLabel(ctx, 1, nil, "temp", "#000000")

		err := service.DeleteLabel(ctx, 1, label.ID)
		if err != nil {
			t.Fatalf("DeleteLabel() error = %v", err)
		}

		_, err = service.GetLabel(ctx, label.ID)
		if err != ErrLabelNotFound {
			t.Errorf("expected ErrLabelNotFound, got %v", err)
		}
	})
}

func TestLabel_Errors(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	tests := []struct {
		name    string
		op      func() error
		wantErr error
	}{
		{
			name:    "get not found",
			op:      func() error { _, err := service.GetLabel(ctx, 99999); return err },
			wantErr: ErrLabelNotFound,
		},
		{
			name:    "update not found",
			op:      func() error { _, err := service.UpdateLabel(ctx, 1, 99999, map[string]interface{}{"name": "x"}); return err },
			wantErr: ErrLabelNotFound,
		},
		// Note: DeleteLabel does not check if label exists, it just deletes (no-op if not found)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.op(); err != tt.wantErr {
				t.Errorf("error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestDeleteLabel_NoError(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	// DeleteLabel should not error even if label doesn't exist
	err := service.DeleteLabel(ctx, 1, 99999)
	if err != nil {
		t.Errorf("DeleteLabel() should not error for non-existent label, got %v", err)
	}
}

func TestListLabels_WithRepositoryScope(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	// Org-level label
	service.CreateLabel(ctx, 1, nil, "org-label", "#FF0000")

	// Repo-level label
	repoID := int64(100)
	service.CreateLabel(ctx, 1, &repoID, "repo-label", "#00FF00")

	tests := []struct {
		name      string
		repoID    *int64
		wantCount int
	}{
		{"org only", nil, 1},
		{"org + repo", &repoID, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			labels, err := service.ListLabels(ctx, 1, tt.repoID)
			if err != nil {
				t.Fatalf("ListLabels() error = %v", err)
			}
			if len(labels) != tt.wantCount {
				t.Errorf("len = %d, want %d", len(labels), tt.wantCount)
			}
		})
	}
}
