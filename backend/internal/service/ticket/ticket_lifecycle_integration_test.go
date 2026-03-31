package ticket

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newIntegrationService creates a ticket Service backed by the shared testutil DB.
func newIntegrationService(t *testing.T) (*Service, context.Context, int64, int64) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	svc := NewService(infra.NewTicketRepository(db))
	ctx := context.Background()

	userID := testutil.CreateUser(t, db, "alice@test.com", "alice")
	orgID := testutil.CreateOrg(t, db, "test-org", userID)
	return svc, ctx, orgID, userID
}

func TestTicket_CreateAndGet(t *testing.T) {
	svc, ctx, orgID, userID := newIntegrationService(t)

	tkt, err := svc.CreateTicket(ctx, &CreateTicketRequest{
		OrganizationID: orgID,
		ReporterID:     userID,
		Title:          "Login page returns 500",
		Priority:       ticket.TicketPriorityHigh,
	})
	require.NoError(t, err)
	require.NotNil(t, tkt)
	assert.Equal(t, "Login page returns 500", tkt.Title)
	assert.Equal(t, ticket.TicketStatusBacklog, tkt.Status)
	assert.Equal(t, ticket.TicketPriorityHigh, tkt.Priority)
	assert.Equal(t, orgID, tkt.OrganizationID)
	assert.Equal(t, userID, tkt.ReporterID)
	assert.NotZero(t, tkt.ID)
	assert.NotEmpty(t, tkt.Slug)
	assert.Equal(t, 1, tkt.Number)

	// Get by ID
	got, err := svc.GetTicket(ctx, tkt.ID)
	require.NoError(t, err)
	assert.Equal(t, tkt.ID, got.ID)
	assert.Equal(t, tkt.Title, got.Title)
	assert.Equal(t, tkt.Slug, got.Slug)
}

func TestTicket_StatusTransition(t *testing.T) {
	svc, ctx, orgID, userID := newIntegrationService(t)

	tkt, err := svc.CreateTicket(ctx, &CreateTicketRequest{
		OrganizationID: orgID,
		ReporterID:     userID,
		Title:          "Implement OAuth",
		Priority:       ticket.TicketPriorityMedium,
	})
	require.NoError(t, err)
	assert.Equal(t, ticket.TicketStatusBacklog, tkt.Status)
	assert.Nil(t, tkt.StartedAt)
	assert.Nil(t, tkt.CompletedAt)

	// Transition to in_progress → sets StartedAt
	err = svc.UpdateStatus(ctx, tkt.ID, ticket.TicketStatusInProgress)
	require.NoError(t, err)

	got, err := svc.GetTicket(ctx, tkt.ID)
	require.NoError(t, err)
	assert.Equal(t, ticket.TicketStatusInProgress, got.Status)
	assert.NotNil(t, got.StartedAt, "StartedAt should be set when status = in_progress")

	// Transition to done → sets CompletedAt
	err = svc.UpdateStatus(ctx, tkt.ID, ticket.TicketStatusDone)
	require.NoError(t, err)

	got, err = svc.GetTicket(ctx, tkt.ID)
	require.NoError(t, err)
	assert.Equal(t, ticket.TicketStatusDone, got.Status)
	assert.NotNil(t, got.CompletedAt, "CompletedAt should be set when status = done")
	assert.True(t, got.IsCompleted())
}

func TestTicket_CommentCRUD(t *testing.T) {
	svc, ctx, orgID, userID := newIntegrationService(t)

	tkt, err := svc.CreateTicket(ctx, &CreateTicketRequest{
		OrganizationID: orgID,
		ReporterID:     userID,
		Title:          "Bug report",
	})
	require.NoError(t, err)

	// Create comment
	comment, err := svc.CreateComment(ctx, tkt.ID, userID, "First comment", nil, nil)
	require.NoError(t, err)
	require.NotNil(t, comment)
	assert.Equal(t, "First comment", comment.Content)
	assert.Equal(t, tkt.ID, comment.TicketID)
	assert.Equal(t, userID, comment.UserID)

	// List comments
	comments, total, err := svc.ListComments(ctx, tkt.ID, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, comments, 1)

	// Edit comment
	edited, err := svc.UpdateComment(ctx, tkt.ID, comment.ID, userID, "Edited comment", nil)
	require.NoError(t, err)
	assert.Equal(t, "Edited comment", edited.Content)

	// Delete comment
	err = svc.DeleteComment(ctx, tkt.ID, comment.ID, userID)
	require.NoError(t, err)

	comments, total, err = svc.ListComments(ctx, tkt.ID, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Empty(t, comments)
}

func TestTicket_LabelManagement(t *testing.T) {
	svc, ctx, orgID, userID := newIntegrationService(t)

	// Create label
	label, err := svc.CreateLabel(ctx, orgID, nil, "enhancement", "#00FF00")
	require.NoError(t, err)
	require.NotNil(t, label)
	assert.Equal(t, "enhancement", label.Name)
	assert.Equal(t, "#00FF00", label.Color)

	// Create ticket
	tkt, err := svc.CreateTicket(ctx, &CreateTicketRequest{
		OrganizationID: orgID,
		ReporterID:     userID,
		Title:          "Add dark mode",
	})
	require.NoError(t, err)

	// Attach label
	err = svc.AddLabel(ctx, tkt.ID, label.ID)
	require.NoError(t, err)

	// List ticket labels
	labels, err := svc.GetTicketLabels(ctx, tkt.ID)
	require.NoError(t, err)
	require.Len(t, labels, 1)
	assert.Equal(t, label.ID, labels[0].ID)
	assert.Equal(t, "enhancement", labels[0].Name)

	// Detach label
	err = svc.RemoveLabel(ctx, tkt.ID, label.ID)
	require.NoError(t, err)

	labels, err = svc.GetTicketLabels(ctx, tkt.ID)
	require.NoError(t, err)
	assert.Empty(t, labels)
}

func TestTicket_SlugLookup(t *testing.T) {
	svc, ctx, orgID, userID := newIntegrationService(t)

	tkt, err := svc.CreateTicket(ctx, &CreateTicketRequest{
		OrganizationID: orgID,
		ReporterID:     userID,
		Title:          "Slug lookup test",
	})
	require.NoError(t, err)
	require.NotEmpty(t, tkt.Slug)

	// GetBySlug
	got, err := svc.GetTicketBySlug(ctx, orgID, tkt.Slug)
	require.NoError(t, err)
	assert.Equal(t, tkt.ID, got.ID)
	assert.Equal(t, tkt.Slug, got.Slug)

	// GetByIDOrSlug — slug path
	got2, err := svc.GetTicketByIDOrSlug(ctx, orgID, tkt.Slug)
	require.NoError(t, err)
	assert.Equal(t, tkt.ID, got2.ID)

	// Wrong org → not found
	_, err = svc.GetTicketBySlug(ctx, orgID+999, tkt.Slug)
	assert.ErrorIs(t, err, ErrTicketNotFound)
}
