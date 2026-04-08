package ticket

import (
	"context"
	"testing"
)

// TestCreateTicket_SlugScopedToOrg verifies that slug generation
// is scoped to organization_id, allowing different orgs to have the same slug.
// This was the root cause of a production 500 bug: the old code had a global unique
// constraint on slug but generated numbers per-org, causing conflicts.
func TestCreateTicket_SlugScopedToOrg(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	// Create ticket in org 1
	tkt1, err := service.CreateTicket(ctx, &CreateTicketRequest{
		OrganizationID: 1,
		ReporterID:     1,

		Title:          "Org1 Ticket",
		Priority:       "medium",
	})
	if err != nil {
		t.Fatalf("failed to create ticket in org 1: %v", err)
	}
	if tkt1.Slug != "TICKET-1" {
		t.Errorf("org 1 first ticket: expected Slug 'TICKET-1', got %s", tkt1.Slug)
	}

	// Create ticket in org 2 — should also get TICKET-1, not conflict
	tkt2, err := service.CreateTicket(ctx, &CreateTicketRequest{
		OrganizationID: 2,
		ReporterID:     2,

		Title:          "Org2 Ticket",
		Priority:       "medium",
	})
	if err != nil {
		t.Fatalf("failed to create ticket in org 2: %v", err)
	}
	if tkt2.Slug != "TICKET-1" {
		t.Errorf("org 2 first ticket: expected Slug 'TICKET-1', got %s", tkt2.Slug)
	}

	// Create second ticket in org 1 — should get TICKET-2
	tkt3, err := service.CreateTicket(ctx, &CreateTicketRequest{
		OrganizationID: 1,
		ReporterID:     1,

		Title:          "Org1 Second Ticket",
		Priority:       "medium",
	})
	if err != nil {
		t.Fatalf("failed to create second ticket in org 1: %v", err)
	}
	if tkt3.Slug != "TICKET-2" {
		t.Errorf("org 1 second ticket: expected Slug 'TICKET-2', got %s", tkt3.Slug)
	}
	if tkt3.Number != 2 {
		t.Errorf("org 1 second ticket: expected Number 2, got %d", tkt3.Number)
	}
}

// TestCreateTicket_PrefixScopedToOrg verifies that repository-prefix slug
// generation is also scoped to organization, not globally.
func TestCreateTicket_PrefixScopedToOrg(t *testing.T) {
	db := setupTestDB(t)
	// Seed repositories with ticket prefix
	db.Exec(`INSERT INTO repositories (organization_id, external_id, name, slug, ticket_prefix) VALUES (1, 'ext-1', 'repo1', 'repo1', 'PROJ')`)
	db.Exec(`INSERT INTO repositories (organization_id, external_id, name, slug, ticket_prefix) VALUES (2, 'ext-2', 'repo2', 'repo2', 'PROJ')`)

	service := newTestService(db)
	ctx := context.Background()

	repoID1 := int64(1)
	repoID2 := int64(2)

	// Org 1, repo 1, prefix PROJ
	tkt1, err := service.CreateTicket(ctx, &CreateTicketRequest{
		OrganizationID: 1,
		ReporterID:     1,

		Title:          "Org1 Repo1",
		Priority:       "medium",
		RepositoryID:   &repoID1,
	})
	if err != nil {
		t.Fatalf("failed to create ticket for org1/repo1: %v", err)
	}
	if tkt1.Slug != "PROJ-1" {
		t.Errorf("expected 'PROJ-1', got %s", tkt1.Slug)
	}

	// Org 2, repo 2, same prefix PROJ — should also get PROJ-1
	tkt2, err := service.CreateTicket(ctx, &CreateTicketRequest{
		OrganizationID: 2,
		ReporterID:     2,

		Title:          "Org2 Repo2",
		Priority:       "medium",
		RepositoryID:   &repoID2,
	})
	if err != nil {
		t.Fatalf("failed to create ticket for org2/repo2: %v", err)
	}
	if tkt2.Slug != "PROJ-1" {
		t.Errorf("expected 'PROJ-1', got %s", tkt2.Slug)
	}

	// Org 1, repo 1 again — should get PROJ-2
	tkt3, err := service.CreateTicket(ctx, &CreateTicketRequest{
		OrganizationID: 1,
		ReporterID:     1,

		Title:          "Org1 Repo1 Second",
		Priority:       "medium",
		RepositoryID:   &repoID1,
	})
	if err != nil {
		t.Fatalf("failed to create second ticket for org1/repo1: %v", err)
	}
	if tkt3.Slug != "PROJ-2" {
		t.Errorf("expected 'PROJ-2', got %s", tkt3.Slug)
	}
}

// TestCreateTicket_MixedPrefixesInSameOrg verifies that different prefixes
// within the same organization maintain independent numbering.
func TestCreateTicket_MixedPrefixesInSameOrg(t *testing.T) {
	db := setupTestDB(t)
	// Seed repositories with different ticket prefixes
	db.Exec(`INSERT INTO repositories (organization_id, external_id, name, slug, ticket_prefix) VALUES (1, 'ext-1', 'repo1', 'repo1', 'PROJ')`)
	db.Exec(`INSERT INTO repositories (organization_id, external_id, name, slug, ticket_prefix) VALUES (1, 'ext-2', 'repo2', 'repo2', 'BUG')`)

	service := newTestService(db)
	ctx := context.Background()

	repoID1 := int64(1)
	repoID2 := int64(2)

	// Create PROJ-1
	tkt1, err := service.CreateTicket(ctx, &CreateTicketRequest{
		OrganizationID: 1,
		ReporterID:     1,

		Title:          "Project Task",
		Priority:       "medium",
		RepositoryID:   &repoID1,
	})
	if err != nil {
		t.Fatalf("failed to create PROJ ticket: %v", err)
	}
	if tkt1.Slug != "PROJ-1" {
		t.Errorf("expected 'PROJ-1', got %s", tkt1.Slug)
	}

	// Create BUG-1 (different prefix, same org, independent numbering)
	tkt2, err := service.CreateTicket(ctx, &CreateTicketRequest{
		OrganizationID: 1,
		ReporterID:     1,

		Title:          "Bug Report",
		Priority:       "high",
		RepositoryID:   &repoID2,
	})
	if err != nil {
		t.Fatalf("failed to create BUG ticket: %v", err)
	}
	if tkt2.Slug != "BUG-1" {
		t.Errorf("expected 'BUG-1', got %s", tkt2.Slug)
	}

	// Create TICKET-1 (no repo, default prefix, independent numbering)
	tkt3, err := service.CreateTicket(ctx, &CreateTicketRequest{
		OrganizationID: 1,
		ReporterID:     1,

		Title:          "No Repo Task",
		Priority:       "medium",
	})
	if err != nil {
		t.Fatalf("failed to create TICKET ticket: %v", err)
	}
	if tkt3.Slug != "TICKET-1" {
		t.Errorf("expected 'TICKET-1', got %s", tkt3.Slug)
	}

	// Create PROJ-2
	tkt4, err := service.CreateTicket(ctx, &CreateTicketRequest{
		OrganizationID: 1,
		ReporterID:     1,

		Title:          "Project Task 2",
		Priority:       "medium",
		RepositoryID:   &repoID1,
	})
	if err != nil {
		t.Fatalf("failed to create second PROJ ticket: %v", err)
	}
	if tkt4.Slug != "PROJ-2" {
		t.Errorf("expected 'PROJ-2', got %s", tkt4.Slug)
	}
}
