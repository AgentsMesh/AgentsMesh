//go:build integration

package mcp

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/mcp/tools"
	"github.com/stretchr/testify/require"
)

// Integration tests for BackendClient
// These tests require a running backend server
//
// Environment variables:
// - BACKEND_URL: Backend server URL (e.g., http://localhost:8080)
// - POD_KEY: Valid pod key for authentication
//
// Run with: go test -tags=integration ./internal/mcp/... -v

func getTestClient(t *testing.T) *BackendClient {
	backendURL := os.Getenv("BACKEND_URL")
	podKey := os.Getenv("POD_KEY")

	if backendURL == "" || podKey == "" {
		t.Skip("BACKEND_URL and POD_KEY required for integration tests")
	}

	return NewBackendClient(backendURL, podKey)
}

func TestIntegration_ChannelOperations(t *testing.T) {
	client := getTestClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test CreateChannel
	channelName := "test-channel-" + time.Now().Format("20060102150405")
	ch, err := client.CreateChannel(ctx, channelName, "Integration test channel", nil, nil)
	require.NoError(t, err, "CreateChannel should succeed")
	require.NotZero(t, ch.ID, "Channel ID should not be zero")
	require.Equal(t, channelName, ch.Name, "Channel name should match")

	channelID := ch.ID

	// Test GetChannel
	ch, err = client.GetChannel(ctx, channelID)
	require.NoError(t, err, "GetChannel should succeed")
	require.Equal(t, channelID, ch.ID, "Channel ID should match")

	// Test SendMessage
	msg, err := client.SendMessage(ctx, channelID, "Hello from integration test", tools.ChannelMessageTypeText, nil, nil)
	require.NoError(t, err, "SendMessage should succeed")
	require.NotZero(t, msg.ID, "Message ID should not be zero")
	require.Equal(t, "Hello from integration test", msg.Content, "Message content should match")

	// Test GetMessages
	messages, err := client.GetMessages(ctx, channelID, nil, nil, nil, 10)
	require.NoError(t, err, "GetMessages should succeed")
	require.NotEmpty(t, messages, "Messages should not be empty")

	// Test GetDocument (initially empty)
	doc, err := client.GetDocument(ctx, channelID)
	require.NoError(t, err, "GetDocument should succeed")
	// Document may be empty for new channel

	// Test UpdateDocument
	testDoc := "# Integration Test Document\n\nThis is a test document."
	err = client.UpdateDocument(ctx, channelID, testDoc)
	require.NoError(t, err, "UpdateDocument should succeed")

	// Verify document was updated
	doc, err = client.GetDocument(ctx, channelID)
	require.NoError(t, err, "GetDocument after update should succeed")
	require.Equal(t, testDoc, doc, "Document content should match")

	// Test SearchChannels
	channels, err := client.SearchChannels(ctx, channelName, nil, nil, nil, 0, 10)
	require.NoError(t, err, "SearchChannels should succeed")
	require.NotEmpty(t, channels, "Search should find the channel")
}

func TestIntegration_PodOperations(t *testing.T) {
	client := getTestClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test ListAvailablePods
	pods, err := client.ListAvailablePods(ctx)
	require.NoError(t, err, "ListAvailablePods should succeed")
	// Pods list may be empty, that's OK

	t.Logf("Found %d available pods", len(pods))
}

func TestIntegration_BindingOperations(t *testing.T) {
	client := getTestClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test GetBindings
	bindings, err := client.GetBindings(ctx, nil)
	require.NoError(t, err, "GetBindings should succeed")
	// Bindings list may be empty, that's OK

	t.Logf("Found %d bindings", len(bindings))

	// Test GetBoundPods
	boundPods, err := client.GetBoundPods(ctx)
	require.NoError(t, err, "GetBoundPods should succeed")
	// Bound pods list may be empty, that's OK

	t.Logf("Found %d bound pods", len(boundPods))

	// Note: RequestBinding, AcceptBinding, RejectBinding require two pods
	// and more complex setup. They should be tested manually or with
	// a more comprehensive test environment.
}

func TestIntegration_TicketOperations(t *testing.T) {
	client := getTestClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test SearchTickets
	tickets, err := client.SearchTickets(ctx, nil, nil, nil, nil, nil, nil, "", 10, 1)
	require.NoError(t, err, "SearchTickets should succeed")
	// Tickets list may be empty, that's OK

	t.Logf("Found %d tickets", len(tickets))

	// Note: CreateTicket requires a valid product_id which depends on
	// the test environment setup. Skip for now unless we have test data.
	if len(tickets) > 0 {
		// Test GetTicket with an existing ticket
		ticketID := tickets[0].Identifier
		if ticketID == "" {
			ticketID = string(rune(tickets[0].ID))
		}

		ticket, err := client.GetTicket(ctx, ticketID)
		require.NoError(t, err, "GetTicket should succeed")
		require.NotZero(t, ticket.ID, "Ticket ID should not be zero")
	}
}
