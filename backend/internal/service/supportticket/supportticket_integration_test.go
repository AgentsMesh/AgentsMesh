package supportticket

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/domain/supportticket"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/testkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupService(t *testing.T) (*Service, supportticket.Repository, context.Context, int64, int64) {
	t.Helper()
	db := testkit.SetupTestDB(t)
	repo := infra.NewSupportTicketRepository(db)
	svc := NewService(repo, nil, config.StorageConfig{})

	user1 := testkit.CreateUser(t, db, "user1@test.com", "user1")
	user2 := testkit.CreateUser(t, db, "user2@test.com", "user2")

	return svc, repo, context.Background(), user1, user2
}

func TestSupportTicket_CreateAndList(t *testing.T) {
	svc, _, ctx, userID, _ := setupService(t)

	ticket, err := svc.Create(ctx, userID, &CreateRequest{
		Title:    "Cannot login",
		Category: supportticket.CategoryBug,
		Content:  "Login button does not respond.",
		Priority: supportticket.PriorityHigh,
	})
	require.NoError(t, err)
	require.NotNil(t, ticket)
	assert.Equal(t, "Cannot login", ticket.Title)
	assert.Equal(t, supportticket.StatusOpen, ticket.Status)
	assert.Equal(t, supportticket.CategoryBug, ticket.Category)

	resp, err := svc.ListByUser(ctx, userID, &ListQuery{Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Equal(t, int64(1), resp.Total)
	require.Len(t, resp.Data, 1)
	assert.Equal(t, ticket.ID, resp.Data[0].ID)
	assert.Equal(t, "Cannot login", resp.Data[0].Title)
}

func TestSupportTicket_AddMessageAndReopen(t *testing.T) {
	svc, repo, ctx, userID, _ := setupService(t)

	ticket, err := svc.Create(ctx, userID, &CreateRequest{
		Title:    "Reopen test",
		Category: supportticket.CategoryOther,
		Content:  "Initial message",
	})
	require.NoError(t, err)

	// Close the ticket via repo UpdateStatus
	_, err = repo.UpdateStatus(ctx, ticket.ID, supportticket.StatusOpen, supportticket.StatusClosed, map[string]interface{}{
		"status": supportticket.StatusClosed,
	})
	require.NoError(t, err)

	// Verify it is now closed
	got, err := svc.GetByID(ctx, ticket.ID, userID)
	require.NoError(t, err)
	assert.Equal(t, supportticket.StatusClosed, got.Status)

	// AddMessage should reopen the closed ticket
	msg, err := svc.AddMessage(ctx, ticket.ID, userID, &AddMessageRequest{
		Content: "I have more info to share.",
	})
	require.NoError(t, err)
	require.NotNil(t, msg)
	assert.Equal(t, "I have more info to share.", msg.Content)

	// Verify the ticket has been reopened
	reopened, err := svc.GetByID(ctx, ticket.ID, userID)
	require.NoError(t, err)
	assert.Equal(t, supportticket.StatusOpen, reopened.Status)
}

func TestSupportTicket_AccessControl(t *testing.T) {
	svc, _, ctx, user1, user2 := setupService(t)

	ticket, err := svc.Create(ctx, user1, &CreateRequest{
		Title:    "Private issue",
		Category: supportticket.CategoryAccount,
		Content:  "Account locked out.",
		Priority: supportticket.PriorityHigh,
	})
	require.NoError(t, err)

	// Owner can access
	got, err := svc.GetByID(ctx, ticket.ID, user1)
	require.NoError(t, err)
	assert.Equal(t, ticket.ID, got.ID)

	// Non-owner gets ErrTicketNotFound (nil returned from repo)
	_, err = svc.GetByID(ctx, ticket.ID, user2)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrTicketNotFound)
}
