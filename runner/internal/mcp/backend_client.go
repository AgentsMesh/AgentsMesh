package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/mcp/tools"
)

// BackendClient calls the AgentsMesh Backend API for collaboration operations.
type BackendClient struct {
	baseURL    string
	orgSlug    string // Organization slug for org-scoped API paths
	podKey     string
	httpClient *http.Client
}

// NewBackendClient creates a new backend API client.
// orgSlug is required for org-scoped API paths (/api/v1/orgs/:slug/pod/*)
func NewBackendClient(baseURL, orgSlug, podKey string) *BackendClient {
	return &BackendClient{
		baseURL: baseURL,
		orgSlug: orgSlug,
		podKey:  podKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetPodKey updates the pod key for the client.
func (c *BackendClient) SetPodKey(podKey string) {
	c.podKey = podKey
}

// GetPodKey returns the current pod key.
func (c *BackendClient) GetPodKey() string {
	return c.podKey
}

// SetOrgSlug updates the organization slug for the client.
func (c *BackendClient) SetOrgSlug(orgSlug string) {
	c.orgSlug = orgSlug
}

// GetOrgSlug returns the current organization slug.
func (c *BackendClient) GetOrgSlug() string {
	return c.orgSlug
}

// podAPIPath returns the org-scoped pod API path prefix for MCP tools.
// MCP tools use /api/v1/orgs/:slug/pod/* with X-Pod-Key authentication.
func (c *BackendClient) podAPIPath() string {
	return fmt.Sprintf("/api/v1/orgs/%s/pod", c.orgSlug)
}

// request makes an HTTP request to the backend.
func (c *BackendClient) request(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	fullURL := c.baseURL + path

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Pod-Key", c.podKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}

// Terminal Operations

// ObserveTerminal gets terminal output from another pod.
func (c *BackendClient) ObserveTerminal(ctx context.Context, podKey string, lines int, raw bool, includeScreen bool) (*tools.TerminalOutput, error) {
	params := url.Values{}
	params.Set("lines", strconv.Itoa(lines))
	params.Set("raw", strconv.FormatBool(raw))
	params.Set("include_screen", strconv.FormatBool(includeScreen))

	path := fmt.Sprintf("%s/pods/%s/terminal/observe?%s", c.podAPIPath(), url.PathEscape(podKey), params.Encode())

	var result tools.TerminalOutput
	err := c.request(ctx, http.MethodGet, path, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// SendTerminalText sends text input to a terminal.
func (c *BackendClient) SendTerminalText(ctx context.Context, podKey string, text string) error {
	body := map[string]interface{}{
		"input": text,
	}
	return c.request(ctx, http.MethodPost, fmt.Sprintf("%s/pods/%s/terminal/input", c.podAPIPath(), url.PathEscape(podKey)), body, nil)
}

// SendTerminalKey sends special keys to a terminal.
func (c *BackendClient) SendTerminalKey(ctx context.Context, podKey string, keys []string) error {
	// Convert keys to escape sequences and concatenate
	input := convertKeysToInput(keys)
	body := map[string]interface{}{
		"input": input,
	}
	return c.request(ctx, http.MethodPost, fmt.Sprintf("%s/pods/%s/terminal/input", c.podAPIPath(), url.PathEscape(podKey)), body, nil)
}

// convertKeysToInput converts key names to terminal escape sequences.
func convertKeysToInput(keys []string) string {
	var result string
	for _, key := range keys {
		switch key {
		case "enter":
			result += "\r"
		case "escape":
			result += "\x1b"
		case "tab":
			result += "\t"
		case "backspace":
			result += "\x7f"
		case "delete":
			result += "\x1b[3~"
		case "ctrl+c":
			result += "\x03"
		case "ctrl+d":
			result += "\x04"
		case "ctrl+u":
			result += "\x15"
		case "ctrl+l":
			result += "\x0c"
		case "ctrl+z":
			result += "\x1a"
		case "ctrl+a":
			result += "\x01"
		case "ctrl+e":
			result += "\x05"
		case "ctrl+k":
			result += "\x0b"
		case "ctrl+w":
			result += "\x17"
		case "up":
			result += "\x1b[A"
		case "down":
			result += "\x1b[B"
		case "left":
			result += "\x1b[D"
		case "right":
			result += "\x1b[C"
		case "home":
			result += "\x1b[H"
		case "end":
			result += "\x1b[F"
		case "pageup":
			result += "\x1b[5~"
		case "pagedown":
			result += "\x1b[6~"
		case "shift+tab":
			result += "\x1b[Z"
		default:
			// Single character keys
			if len(key) == 1 {
				result += key
			}
		}
	}
	return result
}

// Discovery Operations

// ListAvailablePods lists pods available for collaboration.
func (c *BackendClient) ListAvailablePods(ctx context.Context) ([]tools.AvailablePod, error) {
	var result struct {
		Pods []tools.AvailablePod `json:"pods"`
	}
	// Use pods endpoint with status filter
	err := c.request(ctx, http.MethodGet, c.podAPIPath()+"/pods?status=running", nil, &result)
	if err != nil {
		return nil, err
	}
	return result.Pods, nil
}

// ListRunners returns simplified Runner list with nested Agent info.
func (c *BackendClient) ListRunners(ctx context.Context) ([]tools.RunnerSummary, error) {
	var result struct {
		Runners []tools.RunnerSummary `json:"runners"`
	}
	err := c.request(ctx, http.MethodGet, c.podAPIPath()+"/runners", nil, &result)
	if err != nil {
		return nil, err
	}
	return result.Runners, nil
}

// ListRepositories lists repositories configured in the organization.
func (c *BackendClient) ListRepositories(ctx context.Context) ([]tools.Repository, error) {
	var result struct {
		Repositories []tools.Repository `json:"repositories"`
	}
	err := c.request(ctx, http.MethodGet, c.podAPIPath()+"/repositories", nil, &result)
	if err != nil {
		return nil, err
	}
	return result.Repositories, nil
}

// Binding Operations

// RequestBinding requests a binding with another pod.
func (c *BackendClient) RequestBinding(ctx context.Context, targetPod string, scopes []tools.BindingScope) (*tools.Binding, error) {
	body := map[string]interface{}{
		"target_pod": targetPod,
		"scopes":     scopes,
	}

	var result struct {
		Binding tools.Binding `json:"binding"`
	}
	err := c.request(ctx, http.MethodPost, c.podAPIPath()+"/bindings", body, &result)
	if err != nil {
		return nil, err
	}
	return &result.Binding, nil
}

// AcceptBinding accepts a binding request.
func (c *BackendClient) AcceptBinding(ctx context.Context, bindingID int) (*tools.Binding, error) {
	body := map[string]interface{}{
		"binding_id": bindingID,
	}

	var result struct {
		Binding tools.Binding `json:"binding"`
	}
	err := c.request(ctx, http.MethodPost, c.podAPIPath()+"/bindings/accept", body, &result)
	if err != nil {
		return nil, err
	}
	return &result.Binding, nil
}

// RejectBinding rejects a binding request.
func (c *BackendClient) RejectBinding(ctx context.Context, bindingID int, reason string) (*tools.Binding, error) {
	body := map[string]interface{}{
		"binding_id": bindingID,
		"reason":     reason,
	}

	var result struct {
		Binding tools.Binding `json:"binding"`
	}
	err := c.request(ctx, http.MethodPost, c.podAPIPath()+"/bindings/reject", body, &result)
	if err != nil {
		return nil, err
	}
	return &result.Binding, nil
}

// UnbindPod unbinds from another pod.
func (c *BackendClient) UnbindPod(ctx context.Context, targetPod string) error {
	body := map[string]interface{}{
		"target_pod": targetPod,
	}
	return c.request(ctx, http.MethodPost, c.podAPIPath()+"/bindings/unbind", body, nil)
}

// GetBindings gets all bindings for the current pod.
func (c *BackendClient) GetBindings(ctx context.Context, status *tools.BindingStatus) ([]tools.Binding, error) {
	path := c.podAPIPath() + "/bindings"
	if status != nil {
		path += "?status=" + url.QueryEscape(string(*status))
	}

	var result struct {
		Bindings []tools.Binding `json:"bindings"`
	}
	err := c.request(ctx, http.MethodGet, path, nil, &result)
	if err != nil {
		return nil, err
	}
	return result.Bindings, nil
}

// GetBoundPods gets pods that are bound to the current pod.
// Returns a list of pod keys (strings) that this pod has active bindings with.
func (c *BackendClient) GetBoundPods(ctx context.Context) ([]string, error) {
	var result struct {
		Pods []string `json:"pods"`
	}
	err := c.request(ctx, http.MethodGet, c.podAPIPath()+"/bindings/pods", nil, &result)
	if err != nil {
		return nil, err
	}
	return result.Pods, nil
}

// Channel Operations

// SearchChannels searches for collaboration channels.
func (c *BackendClient) SearchChannels(ctx context.Context, name string, projectID, ticketID *int, isArchived *bool, offset, limit int) ([]tools.Channel, error) {
	params := url.Values{}
	if name != "" {
		params.Set("name", name)
	}
	if projectID != nil {
		params.Set("project_id", strconv.Itoa(*projectID))
	}
	if ticketID != nil {
		params.Set("ticket_id", strconv.Itoa(*ticketID))
	}
	if isArchived != nil {
		params.Set("is_archived", strconv.FormatBool(*isArchived))
	}
	params.Set("offset", strconv.Itoa(offset))
	params.Set("limit", strconv.Itoa(limit))

	path := c.podAPIPath() + "/channels?" + params.Encode()

	var result struct {
		Channels []tools.Channel `json:"channels"`
	}
	err := c.request(ctx, http.MethodGet, path, nil, &result)
	if err != nil {
		return nil, err
	}
	return result.Channels, nil
}

// CreateChannel creates a new collaboration channel.
func (c *BackendClient) CreateChannel(ctx context.Context, name, description string, projectID, ticketID *int) (*tools.Channel, error) {
	body := map[string]interface{}{
		"name":        name,
		"description": description,
	}
	if projectID != nil {
		body["project_id"] = *projectID
	}
	if ticketID != nil {
		body["ticket_id"] = *ticketID
	}

	var result struct {
		Channel tools.Channel `json:"channel"`
	}
	err := c.request(ctx, http.MethodPost, c.podAPIPath()+"/channels", body, &result)
	if err != nil {
		return nil, err
	}
	return &result.Channel, nil
}

// GetChannel gets a channel by ID.
func (c *BackendClient) GetChannel(ctx context.Context, channelID int) (*tools.Channel, error) {
	var result struct {
		Channel tools.Channel `json:"channel"`
	}
	err := c.request(ctx, http.MethodGet, fmt.Sprintf("%s/channels/%d", c.podAPIPath(), channelID), nil, &result)
	if err != nil {
		return nil, err
	}
	return &result.Channel, nil
}

// SendMessage sends a message to a channel.
func (c *BackendClient) SendMessage(ctx context.Context, channelID int, content string, msgType tools.ChannelMessageType, mentions []string, replyTo *int) (*tools.ChannelMessage, error) {
	body := map[string]interface{}{
		"content":      content,
		"message_type": msgType,
	}
	if len(mentions) > 0 {
		body["mentions"] = mentions
	}
	if replyTo != nil {
		body["reply_to"] = *replyTo
	}

	var result struct {
		Message tools.ChannelMessage `json:"message"`
	}
	err := c.request(ctx, http.MethodPost, fmt.Sprintf("%s/channels/%d/messages", c.podAPIPath(), channelID), body, &result)
	if err != nil {
		return nil, err
	}
	return &result.Message, nil
}

// GetMessages gets messages from a channel.
func (c *BackendClient) GetMessages(ctx context.Context, channelID int, beforeTime, afterTime *string, mentionedPod *string, limit int) ([]tools.ChannelMessage, error) {
	params := url.Values{}
	if beforeTime != nil {
		params.Set("before_time", *beforeTime)
	}
	if afterTime != nil {
		params.Set("after_time", *afterTime)
	}
	if mentionedPod != nil {
		params.Set("mentioned_pod", *mentionedPod)
	}
	params.Set("limit", strconv.Itoa(limit))

	path := fmt.Sprintf("%s/channels/%d/messages?%s", c.podAPIPath(), channelID, params.Encode())

	var result struct {
		Messages []tools.ChannelMessage `json:"messages"`
	}
	err := c.request(ctx, http.MethodGet, path, nil, &result)
	if err != nil {
		return nil, err
	}
	return result.Messages, nil
}

// GetDocument gets the shared document from a channel.
func (c *BackendClient) GetDocument(ctx context.Context, channelID int) (string, error) {
	var result struct {
		Document string `json:"document"`
	}
	err := c.request(ctx, http.MethodGet, fmt.Sprintf("%s/channels/%d/document", c.podAPIPath(), channelID), nil, &result)
	if err != nil {
		return "", err
	}
	return result.Document, nil
}

// UpdateDocument updates the shared document in a channel.
func (c *BackendClient) UpdateDocument(ctx context.Context, channelID int, document string) error {
	body := map[string]interface{}{
		"document": document,
	}
	return c.request(ctx, http.MethodPut, fmt.Sprintf("%s/channels/%d/document", c.podAPIPath(), channelID), body, nil)
}

// Ticket Operations

// SearchTickets searches for tickets.
func (c *BackendClient) SearchTickets(ctx context.Context, productID *int, status *tools.TicketStatus, ticketType *tools.TicketType, priority *tools.TicketPriority, assigneeID, parentID *int, query string, limit, page int) ([]tools.Ticket, error) {
	params := url.Values{}
	params.Set("limit", strconv.Itoa(limit))
	params.Set("page", strconv.Itoa(page))

	if productID != nil {
		params.Set("product_id", strconv.Itoa(*productID))
	}
	if status != nil {
		params.Set("status", string(*status))
	}
	if ticketType != nil {
		params.Set("type", string(*ticketType))
	}
	if priority != nil {
		params.Set("priority", string(*priority))
	}
	if assigneeID != nil {
		params.Set("assignee_id", strconv.Itoa(*assigneeID))
	}
	if parentID != nil {
		params.Set("parent_id", strconv.Itoa(*parentID))
	}
	if query != "" {
		params.Set("query", query)
	}

	path := c.podAPIPath() + "/tickets?" + params.Encode()

	var result struct {
		Tickets []tools.Ticket `json:"tickets"`
	}
	err := c.request(ctx, http.MethodGet, path, nil, &result)
	if err != nil {
		return nil, err
	}
	return result.Tickets, nil
}

// GetTicket gets a ticket by ID or identifier (e.g., "AM-123").
func (c *BackendClient) GetTicket(ctx context.Context, ticketID string) (*tools.Ticket, error) {
	var result struct {
		Ticket tools.Ticket `json:"ticket"`
	}
	err := c.request(ctx, http.MethodGet, fmt.Sprintf("%s/tickets/%s", c.podAPIPath(), url.PathEscape(ticketID)), nil, &result)
	if err != nil {
		return nil, err
	}
	return &result.Ticket, nil
}

// CreateTicket creates a new ticket.
func (c *BackendClient) CreateTicket(ctx context.Context, productID int, title, description string, ticketType tools.TicketType, priority tools.TicketPriority, parentTicketID *int) (*tools.Ticket, error) {
	body := map[string]interface{}{
		"product_id":  productID,
		"title":       title,
		"description": description,
		"type":        ticketType,
		"priority":    priority,
	}
	if parentTicketID != nil {
		body["parent_ticket_id"] = *parentTicketID
	}

	var result struct {
		Ticket tools.Ticket `json:"ticket"`
	}
	err := c.request(ctx, http.MethodPost, c.podAPIPath()+"/tickets", body, &result)
	if err != nil {
		return nil, err
	}
	return &result.Ticket, nil
}

// UpdateTicket updates a ticket.
func (c *BackendClient) UpdateTicket(ctx context.Context, ticketID string, title, description *string, status *tools.TicketStatus, priority *tools.TicketPriority, ticketType *tools.TicketType) (*tools.Ticket, error) {
	body := make(map[string]interface{})
	if title != nil {
		body["title"] = *title
	}
	if description != nil {
		body["description"] = *description
	}
	if status != nil {
		body["status"] = *status
	}
	if priority != nil {
		body["priority"] = *priority
	}
	if ticketType != nil {
		body["type"] = *ticketType
	}

	var result struct {
		Ticket tools.Ticket `json:"ticket"`
	}
	err := c.request(ctx, http.MethodPut, fmt.Sprintf("%s/tickets/%s", c.podAPIPath(), url.PathEscape(ticketID)), body, &result)
	if err != nil {
		return nil, err
	}
	return &result.Ticket, nil
}

// Pod Operations

// CreatePod creates a new AgentPod.
func (c *BackendClient) CreatePod(ctx context.Context, req *tools.PodCreateRequest) (*tools.PodCreateResponse, error) {
	var result struct {
		Pod struct {
			PodKey string `json:"pod_key"`
			Status string `json:"status"`
		} `json:"pod"`
	}
	err := c.request(ctx, http.MethodPost, c.podAPIPath()+"/pods", req, &result)
	if err != nil {
		return nil, err
	}
	return &tools.PodCreateResponse{
		PodKey: result.Pod.PodKey,
		Status: result.Pod.Status,
	}, nil
}

// Verify BackendClient implements CollaborationClient interface.
var _ tools.CollaborationClient = (*BackendClient)(nil)
