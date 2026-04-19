package channel

import (
	"testing"
	"time"
)

// --- Test Channel ---

func TestChannelTableName(t *testing.T) {
	c := Channel{}
	if c.TableName() != "channels" {
		t.Errorf("expected 'channels', got %s", c.TableName())
	}
}

func TestChannelStruct(t *testing.T) {
	now := time.Now()
	desc := "Test channel"
	doc := "Shared doc content"
	repoID := int64(5)
	ticketID := int64(20)
	createdByPod := "pod-123"
	createdByUserID := int64(50)

	c := Channel{
		ID:              1,
		OrganizationID:  100,
		Name:            "Development",
		Description:     &desc,
		Document:        &doc,
		RepositoryID:    &repoID,
		TicketID:        &ticketID,
		CreatedByPod:    &createdByPod,
		CreatedByUserID: &createdByUserID,
		IsArchived:      false,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if c.ID != 1 {
		t.Errorf("expected ID 1, got %d", c.ID)
	}
	if c.Name != "Development" {
		t.Errorf("expected Name 'Development', got %s", c.Name)
	}
	if *c.Description != "Test channel" {
		t.Errorf("expected Description 'Test channel', got %s", *c.Description)
	}
}

// --- Test Message Type Constants ---

func TestMessageTypeConstants(t *testing.T) {
	tests := []struct {
		constant string
		expected string
	}{
		{MessageTypeText, "text"},
		{MessageTypeSystem, "system"},
	}

	for _, tt := range tests {
		if tt.constant != tt.expected {
			t.Errorf("expected '%s', got '%s'", tt.expected, tt.constant)
		}
	}
}
