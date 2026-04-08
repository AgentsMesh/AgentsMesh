package ticket

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/testkit"
	"gorm.io/gorm"
)

// newTestService creates a Service backed by an in-memory DB for testing.
func newTestService(db *gorm.DB) *Service {
	return NewService(infra.NewTicketRepository(db))
}

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return testkit.SetupTestDB(t)
}

func TestNewService(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)

	if service == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestNewServiceWithContext(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	// Verify service can be used with context
	_, _, err := service.ListTickets(ctx, &ticket.TicketListFilter{
		OrganizationID: 1,
		Limit:          10,
	})
	if err != nil {
		t.Fatalf("service should work with context: %v", err)
	}
}
