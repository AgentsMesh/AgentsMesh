// Package tools provides MCP tools for agent collaboration.
package tools

import (
	"context"
)

// TerminalClient defines the interface for terminal operations.
type TerminalClient interface {
	ObserveTerminal(ctx context.Context, podKey string, lines int, raw bool, includeScreen bool) (*TerminalOutput, error)
	SendTerminalText(ctx context.Context, podKey string, text string) error
	SendTerminalKey(ctx context.Context, podKey string, keys []string) error
}

// DiscoveryClient defines the interface for pod discovery.
type DiscoveryClient interface {
	ListAvailablePods(ctx context.Context) ([]AvailablePod, error)
	ListRunners(ctx context.Context) ([]RunnerSummary, error)
	ListRepositories(ctx context.Context) ([]Repository, error)
}

// BindingClient defines the interface for pod binding operations.
type BindingClient interface {
	RequestBinding(ctx context.Context, targetPod string, scopes []BindingScope) (*Binding, error)
	AcceptBinding(ctx context.Context, bindingID int) (*Binding, error)
	RejectBinding(ctx context.Context, bindingID int, reason string) (*Binding, error)
	UnbindPod(ctx context.Context, targetPod string) error
	GetBindings(ctx context.Context, status *BindingStatus) ([]Binding, error)
	GetBoundPods(ctx context.Context) ([]string, error)
}

// ChannelClient defines the interface for channel operations.
type ChannelClient interface {
	SearchChannels(ctx context.Context, name string, repositoryID, ticketID *int, isArchived *bool, offset, limit int) ([]Channel, error)
	CreateChannel(ctx context.Context, name, description string, repositoryID, ticketID *int) (*Channel, error)
	GetChannel(ctx context.Context, channelID int) (*Channel, error)
	SendMessage(ctx context.Context, channelID int, content string, msgType ChannelMessageType, mentions []string, replyTo *int) (*ChannelMessage, error)
	GetMessages(ctx context.Context, channelID int, beforeTime, afterTime *string, mentionedPod *string, limit int) ([]ChannelMessage, error)
	GetDocument(ctx context.Context, channelID int) (string, error)
	UpdateDocument(ctx context.Context, channelID int, document string) error
}

// TicketClient defines the interface for ticket operations.
type TicketClient interface {
	SearchTickets(ctx context.Context, repositoryID *int, status *TicketStatus, ticketType *TicketType, priority *TicketPriority, assigneeID, parentID *int, query string, limit, page int) ([]Ticket, error)
	GetTicket(ctx context.Context, ticketID string) (*Ticket, error)
	CreateTicket(ctx context.Context, repositoryID *int64, title string, ticketType TicketType, priority TicketPriority, parentTicketID *int64) (*Ticket, error)
	UpdateTicket(ctx context.Context, ticketID string, title *string, status *TicketStatus, priority *TicketPriority, ticketType *TicketType) (*Ticket, error)
}

// PodClient defines the interface for pod creation.
type PodClient interface {
	CreatePod(ctx context.Context, req *PodCreateRequest) (*PodCreateResponse, error)
}

// CollaborationClient combines all collaboration interfaces.
type CollaborationClient interface {
	TerminalClient
	DiscoveryClient
	BindingClient
	ChannelClient
	TicketClient
	PodClient

	// GetPodKey returns the current pod's key.
	GetPodKey() string
}
