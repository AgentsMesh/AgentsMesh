package channel

import (
	"context"
	"testing"
)

func TestNewService(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	if svc == nil {
		t.Error("NewService returned nil")
	}
	if svc.repo == nil {
		t.Error("Service repo not set correctly")
	}
}

func TestCreateChannel(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()

	tests := []struct {
		name    string
		req     *CreateChannelRequest
		wantErr bool
	}{
		{
			name:    "basic channel",
			req:     &CreateChannelRequest{OrganizationID: 1, Name: "general"},
			wantErr: false,
		},
		{
			name:    "channel with description",
			req:     &CreateChannelRequest{OrganizationID: 1, Name: "dev", Description: strPtr("Dev discussions")},
			wantErr: false,
		},
		{
			name:    "channel with repository",
			req:     &CreateChannelRequest{OrganizationID: 1, Name: "repo-ch", RepositoryID: intPtr(42)},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch, err := svc.CreateChannel(ctx, tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateChannel() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && (ch == nil || ch.ID == 0 || ch.Name != tt.req.Name || ch.IsArchived) {
				t.Error("Channel validation failed")
			}
		})
	}
}

func TestCreateChannel_DuplicateName(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()

	req := &CreateChannelRequest{OrganizationID: 1, Name: "general"}
	if _, err := svc.CreateChannel(ctx, req); err != nil {
		t.Fatalf("First CreateChannel failed: %v", err)
	}

	if _, err := svc.CreateChannel(ctx, req); err != ErrDuplicateName {
		t.Errorf("Expected ErrDuplicateName, got %v", err)
	}
}

func TestGetChannel(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()

	created, _ := svc.CreateChannel(ctx, &CreateChannelRequest{OrganizationID: 1, Name: "test"})

	t.Run("existing channel", func(t *testing.T) {
		ch, err := svc.GetChannel(ctx, created.ID)
		if err != nil || ch.Name != created.Name {
			t.Errorf("GetChannel failed: %v", err)
		}
	})

	t.Run("non-existent channel", func(t *testing.T) {
		if _, err := svc.GetChannel(ctx, 99999); err != ErrChannelNotFound {
			t.Errorf("Expected ErrChannelNotFound, got %v", err)
		}
	})
}

func TestGetChannelByName(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()

	created, _ := svc.CreateChannel(ctx, &CreateChannelRequest{OrganizationID: 1, Name: "named"})

	t.Run("existing", func(t *testing.T) {
		ch, err := svc.GetChannelByName(ctx, 1, "named")
		if err != nil || ch.ID != created.ID {
			t.Errorf("GetChannelByName failed: %v", err)
		}
	})

	t.Run("non-existent", func(t *testing.T) {
		if _, err := svc.GetChannelByName(ctx, 1, "missing"); err == nil {
			t.Error("Expected error for non-existent channel")
		}
	})
}

func TestGetChannelsByTicket(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()

	ticketID := int64(42)
	for i := 0; i < 2; i++ {
		svc.CreateChannel(ctx, &CreateChannelRequest{
			OrganizationID: 1,
			Name:           "ticket-ch-" + string(rune('0'+i)),
			TicketID:       &ticketID,
		})
	}

	channels, err := svc.GetChannelsByTicket(ctx, ticketID)
	if err != nil || len(channels) != 2 {
		t.Errorf("GetChannelsByTicket failed: %v, count=%d", err, len(channels))
	}
}

func TestErrors(t *testing.T) {
	tests := []struct {
		err      error
		expected string
	}{
		{ErrChannelNotFound, "channel not found"},
		{ErrChannelArchived, "channel is archived"},
		{ErrDuplicateName, "channel name already exists"},
	}

	for _, tt := range tests {
		if tt.err.Error() != tt.expected {
			t.Errorf("Error = %s, want %s", tt.err.Error(), tt.expected)
		}
	}
}

func TestCreateChannelRequest(t *testing.T) {
	req := &CreateChannelRequest{
		OrganizationID:  1,
		Name:            "test",
		Description:     strPtr("desc"),
		RepositoryID:    intPtr(3),
		TicketID:        intPtr(4),
		CreatedByPod:    strPtr("pod"),
		CreatedByUserID: intPtr(5),
	}
	if req.OrganizationID != 1 || req.Name != "test" {
		t.Error("Request fields not set correctly")
	}
}
