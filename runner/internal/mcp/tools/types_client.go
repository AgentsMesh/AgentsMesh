// Package tools provides MCP tools for agent collaboration.
package tools

import (
	"context"
)

// PodInteractionClient defines the interface for pod interaction operations.
type PodInteractionClient interface {
	GetPodSnapshot(ctx context.Context, podKey string, lines int, raw bool, includeScreen bool) (*PodSnapshot, error)
	SendPodInput(ctx context.Context, podKey string, text string, keys []string) error
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
	SearchChannels(ctx context.Context, name string, repositoryID *int, ticketSlug *string, isArchived *bool, offset, limit int) ([]Channel, error)
	CreateChannel(ctx context.Context, name, description string, repositoryID *int, ticketSlug *string) (*Channel, error)
	GetChannel(ctx context.Context, channelID int) (*Channel, error)
	SendMessage(ctx context.Context, channelID int, content string, msgType ChannelMessageType, mentions []string, replyTo *int) (*ChannelMessage, error)
	GetMessages(ctx context.Context, channelID int, beforeTime, afterTime *string, mentionedPod *string, limit int) ([]ChannelMessage, error)
	GetDocument(ctx context.Context, channelID int) (string, error)
	UpdateDocument(ctx context.Context, channelID int, document string) error
}

// TicketClient defines the interface for ticket operations.
type TicketClient interface {
	SearchTickets(ctx context.Context, repositoryID *int, status *TicketStatus, priority *TicketPriority, assigneeID *int, parentTicketSlug *string, query string, limit, page int) ([]Ticket, error)
	GetTicket(ctx context.Context, ticketSlug string, contentOffset, contentLimit *int) (*Ticket, error)
	CreateTicket(ctx context.Context, repositoryID *int64, title, content string, priority TicketPriority, parentTicketSlug *string) (*Ticket, error)
	UpdateTicket(ctx context.Context, ticketSlug string, title, content *string, status *TicketStatus, priority *TicketPriority) (*Ticket, error)
	PostComment(ctx context.Context, ticketSlug, content string, parentID *int64) (*TicketComment, error)
}

// PodClient defines the interface for pod creation.
type PodClient interface {
	CreatePod(ctx context.Context, req *PodCreateRequest) (*PodCreateResponse, error)
}

// LoopClient defines the interface for loop operations.
type LoopClient interface {
	ListLoops(ctx context.Context, status, query string, limit, offset int) ([]LoopSummary, error)
	TriggerLoop(ctx context.Context, loopSlug string, variables map[string]interface{}) (*LoopTriggerResult, error)
}

// CollaborationClient combines all collaboration interfaces.
type CollaborationClient interface {
	PodInteractionClient
	DiscoveryClient
	BindingClient
	ChannelClient
	TicketClient
	PodClient
	LoopClient

	// GetPodKey returns the current pod's key.
	GetPodKey() string
}
