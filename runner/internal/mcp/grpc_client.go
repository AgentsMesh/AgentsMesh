package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/mcp/tools"
)

// GRPCCollaborationClient implements tools.CollaborationClient using gRPC bidirectional stream.
// Instead of HTTP REST calls, it serializes requests to JSON and sends them via RPCClient
// as McpRequest messages over the existing gRPC connection.
type GRPCCollaborationClient struct {
	rpc    *client.RPCClient
	podKey string
}

// NewGRPCCollaborationClient creates a new gRPC-based collaboration client.
func NewGRPCCollaborationClient(rpc *client.RPCClient, podKey string) *GRPCCollaborationClient {
	return &GRPCCollaborationClient{
		rpc:    rpc,
		podKey: podKey,
	}
}

// GetPodKey returns the current pod's key.
func (c *GRPCCollaborationClient) GetPodKey() string {
	return c.podKey
}

// call is a generic helper that sends an MCP request and unmarshals the response.
func (c *GRPCCollaborationClient) call(ctx context.Context, method string, params interface{}, result interface{}) error {
	log := logger.MCP()

	if c.rpc == nil {
		log.Error("RPC client not available", "method", method, "pod_key", c.podKey)
		return fmt.Errorf("RPC client not available")
	}

	log.Debug("Calling backend via gRPC", "method", method, "pod_key", c.podKey)

	respBytes, err := c.rpc.Call(ctx, c.podKey, method, params)
	if err != nil {
		// Already logged in rpc.Call, no need to duplicate
		return err
	}
	if result != nil && len(respBytes) > 0 {
		if err := json.Unmarshal(respBytes, result); err != nil {
			log.Error("Failed to unmarshal MCP response",
				"method", method,
				"pod_key", c.podKey,
				"response_len", len(respBytes),
				"error", err,
			)
			return fmt.Errorf("failed to unmarshal MCP response for %s: %w", method, err)
		}
	}
	return nil
}

// ==================== TerminalClient ====================

// ObserveTerminal gets terminal output from another pod.
func (c *GRPCCollaborationClient) ObserveTerminal(ctx context.Context, podKey string, lines int, raw bool, includeScreen bool) (*tools.TerminalOutput, error) {
	params := map[string]interface{}{
		"pod_key":        podKey,
		"lines":          lines,
		"raw":            raw,
		"include_screen": includeScreen,
	}
	var result tools.TerminalOutput
	if err := c.call(ctx, "observe_terminal", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SendTerminalText sends text input to a terminal.
func (c *GRPCCollaborationClient) SendTerminalText(ctx context.Context, podKey string, text string) error {
	params := map[string]interface{}{
		"pod_key": podKey,
		"text":    text,
	}
	return c.call(ctx, "send_terminal_text", params, nil)
}

// SendTerminalKey sends special keys to a terminal.
func (c *GRPCCollaborationClient) SendTerminalKey(ctx context.Context, podKey string, keys []string) error {
	params := map[string]interface{}{
		"pod_key": podKey,
		"keys":    keys,
	}
	return c.call(ctx, "send_terminal_key", params, nil)
}

// ==================== DiscoveryClient ====================

// ListAvailablePods lists pods available for collaboration.
func (c *GRPCCollaborationClient) ListAvailablePods(ctx context.Context) ([]tools.AvailablePod, error) {
	var result struct {
		Pods []tools.AvailablePod `json:"pods"`
	}
	if err := c.call(ctx, "list_available_pods", nil, &result); err != nil {
		return nil, err
	}
	return result.Pods, nil
}

// ListRunners returns simplified Runner list with nested Agent info.
func (c *GRPCCollaborationClient) ListRunners(ctx context.Context) ([]tools.RunnerSummary, error) {
	var result struct {
		Runners []tools.RunnerSummary `json:"runners"`
	}
	if err := c.call(ctx, "list_runners", nil, &result); err != nil {
		return nil, err
	}
	return result.Runners, nil
}

// ListRepositories lists repositories configured in the organization.
func (c *GRPCCollaborationClient) ListRepositories(ctx context.Context) ([]tools.Repository, error) {
	var result struct {
		Repositories []tools.Repository `json:"repositories"`
	}
	if err := c.call(ctx, "list_repositories", nil, &result); err != nil {
		return nil, err
	}
	return result.Repositories, nil
}

// ==================== BindingClient ====================

// RequestBinding requests a binding with another pod.
func (c *GRPCCollaborationClient) RequestBinding(ctx context.Context, targetPod string, scopes []tools.BindingScope) (*tools.Binding, error) {
	params := map[string]interface{}{
		"target_pod": targetPod,
		"scopes":     scopes,
	}
	var result struct {
		Binding tools.Binding `json:"binding"`
	}
	if err := c.call(ctx, "request_binding", params, &result); err != nil {
		return nil, err
	}
	return &result.Binding, nil
}

// AcceptBinding accepts a binding request.
func (c *GRPCCollaborationClient) AcceptBinding(ctx context.Context, bindingID int) (*tools.Binding, error) {
	params := map[string]interface{}{
		"binding_id": bindingID,
	}
	var result struct {
		Binding tools.Binding `json:"binding"`
	}
	if err := c.call(ctx, "accept_binding", params, &result); err != nil {
		return nil, err
	}
	return &result.Binding, nil
}

// RejectBinding rejects a binding request.
func (c *GRPCCollaborationClient) RejectBinding(ctx context.Context, bindingID int, reason string) (*tools.Binding, error) {
	params := map[string]interface{}{
		"binding_id": bindingID,
		"reason":     reason,
	}
	var result struct {
		Binding tools.Binding `json:"binding"`
	}
	if err := c.call(ctx, "reject_binding", params, &result); err != nil {
		return nil, err
	}
	return &result.Binding, nil
}

// UnbindPod unbinds from another pod.
func (c *GRPCCollaborationClient) UnbindPod(ctx context.Context, targetPod string) error {
	params := map[string]interface{}{
		"target_pod": targetPod,
	}
	return c.call(ctx, "unbind_pod", params, nil)
}

// GetBindings gets all bindings for the current pod.
func (c *GRPCCollaborationClient) GetBindings(ctx context.Context, status *tools.BindingStatus) ([]tools.Binding, error) {
	params := map[string]interface{}{}
	if status != nil {
		params["status"] = string(*status)
	}
	var result struct {
		Bindings []tools.Binding `json:"bindings"`
	}
	if err := c.call(ctx, "get_bindings", params, &result); err != nil {
		return nil, err
	}
	return result.Bindings, nil
}

// GetBoundPods gets pods that are bound to the current pod.
func (c *GRPCCollaborationClient) GetBoundPods(ctx context.Context) ([]string, error) {
	var result struct {
		Pods []string `json:"pods"`
	}
	if err := c.call(ctx, "get_bound_pods", nil, &result); err != nil {
		return nil, err
	}
	return result.Pods, nil
}

// ==================== ChannelClient ====================

// SearchChannels searches for collaboration channels.
func (c *GRPCCollaborationClient) SearchChannels(ctx context.Context, name string, repositoryID *int, ticketSlug *string, isArchived *bool, offset, limit int) ([]tools.Channel, error) {
	params := map[string]interface{}{
		"offset": offset,
		"limit":  limit,
	}
	if name != "" {
		params["name"] = name
	}
	if repositoryID != nil {
		params["repository_id"] = *repositoryID
	}
	if ticketSlug != nil {
		params["ticket_slug"] = *ticketSlug
	}
	if isArchived != nil {
		params["is_archived"] = *isArchived
	}
	var result struct {
		Channels []tools.Channel `json:"channels"`
	}
	if err := c.call(ctx, "search_channels", params, &result); err != nil {
		return nil, err
	}
	return result.Channels, nil
}

// CreateChannel creates a new collaboration channel.
func (c *GRPCCollaborationClient) CreateChannel(ctx context.Context, name, description string, repositoryID *int, ticketSlug *string) (*tools.Channel, error) {
	params := map[string]interface{}{
		"name":        name,
		"description": description,
	}
	if repositoryID != nil {
		params["repository_id"] = *repositoryID
	}
	if ticketSlug != nil {
		params["ticket_slug"] = *ticketSlug
	}
	var result struct {
		Channel tools.Channel `json:"channel"`
	}
	if err := c.call(ctx, "create_channel", params, &result); err != nil {
		return nil, err
	}
	return &result.Channel, nil
}

// GetChannel gets a channel by ID.
func (c *GRPCCollaborationClient) GetChannel(ctx context.Context, channelID int) (*tools.Channel, error) {
	params := map[string]interface{}{
		"channel_id": channelID,
	}
	var result struct {
		Channel tools.Channel `json:"channel"`
	}
	if err := c.call(ctx, "get_channel", params, &result); err != nil {
		return nil, err
	}
	return &result.Channel, nil
}

// SendMessage sends a message to a channel.
func (c *GRPCCollaborationClient) SendMessage(ctx context.Context, channelID int, content string, msgType tools.ChannelMessageType, mentions []string, replyTo *int) (*tools.ChannelMessage, error) {
	params := map[string]interface{}{
		"channel_id":   channelID,
		"content":      content,
		"message_type": msgType,
	}
	if len(mentions) > 0 {
		params["mentions"] = mentions
	}
	if replyTo != nil {
		params["reply_to"] = *replyTo
	}
	var result struct {
		Message tools.ChannelMessage `json:"message"`
	}
	if err := c.call(ctx, "send_message", params, &result); err != nil {
		return nil, err
	}
	return &result.Message, nil
}

// GetMessages gets messages from a channel.
func (c *GRPCCollaborationClient) GetMessages(ctx context.Context, channelID int, beforeTime, afterTime *string, mentionedPod *string, limit int) ([]tools.ChannelMessage, error) {
	params := map[string]interface{}{
		"channel_id": channelID,
		"limit":      limit,
	}
	if beforeTime != nil {
		params["before_time"] = *beforeTime
	}
	if afterTime != nil {
		params["after_time"] = *afterTime
	}
	if mentionedPod != nil {
		params["mentioned_pod"] = *mentionedPod
	}
	var result struct {
		Messages []tools.ChannelMessage `json:"messages"`
	}
	if err := c.call(ctx, "get_messages", params, &result); err != nil {
		return nil, err
	}
	return result.Messages, nil
}

// GetDocument gets the shared document from a channel.
func (c *GRPCCollaborationClient) GetDocument(ctx context.Context, channelID int) (string, error) {
	params := map[string]interface{}{
		"channel_id": channelID,
	}
	var result struct {
		Document string `json:"document"`
	}
	if err := c.call(ctx, "get_document", params, &result); err != nil {
		return "", err
	}
	return result.Document, nil
}

// UpdateDocument updates the shared document in a channel.
func (c *GRPCCollaborationClient) UpdateDocument(ctx context.Context, channelID int, document string) error {
	params := map[string]interface{}{
		"channel_id": channelID,
		"document":   document,
	}
	return c.call(ctx, "update_document", params, nil)
}

// ==================== TicketClient ====================

// SearchTickets searches for tickets.
func (c *GRPCCollaborationClient) SearchTickets(ctx context.Context, repositoryID *int, status *tools.TicketStatus, priority *tools.TicketPriority, assigneeID *int, parentTicketSlug *string, query string, limit, page int) ([]tools.Ticket, error) {
	params := map[string]interface{}{
		"limit": limit,
		"page":  page,
	}
	if repositoryID != nil {
		params["repository_id"] = *repositoryID
	}
	if status != nil {
		params["status"] = string(*status)
	}
	if priority != nil {
		params["priority"] = string(*priority)
	}
	if assigneeID != nil {
		params["assignee_id"] = *assigneeID
	}
	if parentTicketSlug != nil {
		params["parent_ticket_slug"] = *parentTicketSlug
	}
	if query != "" {
		params["query"] = query
	}
	var result struct {
		Tickets []tools.Ticket `json:"tickets"`
	}
	if err := c.call(ctx, "search_tickets", params, &result); err != nil {
		return nil, err
	}
	return result.Tickets, nil
}

// GetTicket gets a ticket by slug with optional content pagination.
func (c *GRPCCollaborationClient) GetTicket(ctx context.Context, ticketSlug string, contentOffset, contentLimit *int) (*tools.Ticket, error) {
	params := map[string]interface{}{
		"ticket_slug": ticketSlug,
	}
	if contentOffset != nil {
		params["content_offset"] = *contentOffset
	}
	if contentLimit != nil {
		params["content_limit"] = *contentLimit
	}
	var result struct {
		Ticket tools.Ticket `json:"ticket"`
	}
	if err := c.call(ctx, "get_ticket", params, &result); err != nil {
		return nil, err
	}
	return &result.Ticket, nil
}

// CreateTicket creates a new ticket.
func (c *GRPCCollaborationClient) CreateTicket(ctx context.Context, repositoryID *int64, title, content string, priority tools.TicketPriority, parentTicketSlug *string) (*tools.Ticket, error) {
	params := map[string]interface{}{
		"title":    title,
		"priority": priority,
	}
	if content != "" {
		params["content"] = content
	}
	if repositoryID != nil {
		params["repository_id"] = *repositoryID
	}
	if parentTicketSlug != nil {
		params["parent_ticket_slug"] = *parentTicketSlug
	}
	var result struct {
		Ticket tools.Ticket `json:"ticket"`
	}
	if err := c.call(ctx, "create_ticket", params, &result); err != nil {
		return nil, err
	}
	return &result.Ticket, nil
}

// UpdateTicket updates a ticket.
func (c *GRPCCollaborationClient) UpdateTicket(ctx context.Context, ticketSlug string, title, content *string, status *tools.TicketStatus, priority *tools.TicketPriority) (*tools.Ticket, error) {
	params := map[string]interface{}{
		"ticket_slug": ticketSlug,
	}
	if title != nil {
		params["title"] = *title
	}
	if content != nil {
		params["content"] = *content
	}
	if status != nil {
		params["status"] = *status
	}
	if priority != nil {
		params["priority"] = *priority
	}
	var result struct {
		Ticket tools.Ticket `json:"ticket"`
	}
	if err := c.call(ctx, "update_ticket", params, &result); err != nil {
		return nil, err
	}
	return &result.Ticket, nil
}

// PostComment posts a comment on a ticket.
func (c *GRPCCollaborationClient) PostComment(ctx context.Context, ticketSlug, content string, parentID *int64) (*tools.TicketComment, error) {
	params := map[string]interface{}{
		"ticket_slug": ticketSlug,
		"content":     content,
	}
	if parentID != nil {
		params["parent_id"] = *parentID
	}
	var result struct {
		Comment tools.TicketComment `json:"comment"`
	}
	if err := c.call(ctx, "post_comment", params, &result); err != nil {
		return nil, err
	}
	return &result.Comment, nil
}

// ==================== PodClient ====================

// CreatePod creates a new AgentPod.
func (c *GRPCCollaborationClient) CreatePod(ctx context.Context, req *tools.PodCreateRequest) (*tools.PodCreateResponse, error) {
	var result struct {
		Pod struct {
			PodKey string `json:"pod_key"`
			Status string `json:"status"`
		} `json:"pod"`
	}
	if err := c.call(ctx, "create_pod", req, &result); err != nil {
		return nil, err
	}
	return &tools.PodCreateResponse{
		PodKey: result.Pod.PodKey,
		Status: result.Pod.Status,
	}, nil
}

// Verify GRPCCollaborationClient implements CollaborationClient interface.
var _ tools.CollaborationClient = (*GRPCCollaborationClient)(nil)
