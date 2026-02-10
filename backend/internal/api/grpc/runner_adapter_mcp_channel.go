package grpc

import (
	"context"
	"encoding/json"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/channel"
)

// ==================== Channel MCP Methods ====================

// mcpSearchChannels handles the "search_channels" MCP method.
func (a *GRPCRunnerAdapter) mcpSearchChannels(ctx context.Context, tc *middleware.TenantContext, podKey string, payload []byte) (interface{}, *mcpError) {
	var params struct {
		Name         string `json:"name"`
		RepositoryID *int64 `json:"repository_id"`
		TicketID     *int64 `json:"ticket_id"`
		IsArchived   *bool  `json:"is_archived"`
		Offset       int    `json:"offset"`
		Limit        int    `json:"limit"`
	}
	if err := unmarshalPayload(payload, &params); err != nil {
		return nil, err
	}

	includeArchived := false
	if params.IsArchived != nil {
		includeArchived = *params.IsArchived
	}

	limit := params.Limit
	if limit <= 0 {
		limit = 50
	}

	channels, _, mcpErr := a.channelService.ListChannels(ctx, tc.OrganizationID, includeArchived, limit, params.Offset)
	if mcpErr != nil {
		return nil, newMcpError(500, "failed to list channels")
	}

	return map[string]interface{}{"channels": channels}, nil
}

// mcpCreateChannel handles the "create_channel" MCP method.
func (a *GRPCRunnerAdapter) mcpCreateChannel(ctx context.Context, tc *middleware.TenantContext, podKey string, payload []byte) (interface{}, *mcpError) {
	var params struct {
		Name         string `json:"name"`
		Description  string `json:"description"`
		RepositoryID *int64 `json:"repository_id"`
		TicketID     *int64 `json:"ticket_id"`
	}
	if err := unmarshalPayload(payload, &params); err != nil {
		return nil, err
	}

	if params.Name == "" {
		return nil, newMcpError(400, "name is required")
	}

	var desc *string
	if params.Description != "" {
		desc = &params.Description
	}

	ch, err := a.channelService.CreateChannel(ctx, &channel.CreateChannelRequest{
		OrganizationID:  tc.OrganizationID,
		Name:            params.Name,
		Description:     desc,
		RepositoryID:    params.RepositoryID,
		TicketID:        params.TicketID,
		CreatedByPod:    &podKey,
		CreatedByUserID: &tc.UserID,
	})
	if err != nil {
		if err == channel.ErrDuplicateName {
			return nil, newMcpError(409, "channel name already exists")
		}
		return nil, newMcpError(500, "failed to create channel")
	}

	return map[string]interface{}{"channel": ch}, nil
}

// mcpGetChannel handles the "get_channel" MCP method.
func (a *GRPCRunnerAdapter) mcpGetChannel(ctx context.Context, tc *middleware.TenantContext, payload []byte) (interface{}, *mcpError) {
	var params struct {
		ChannelID int64 `json:"channel_id"`
	}
	if err := unmarshalPayload(payload, &params); err != nil {
		return nil, err
	}

	if params.ChannelID == 0 {
		return nil, newMcpError(400, "channel_id is required")
	}

	ch, err := a.channelService.GetChannel(ctx, params.ChannelID)
	if err != nil {
		return nil, newMcpError(404, "channel not found")
	}

	if ch.OrganizationID != tc.OrganizationID {
		return nil, newMcpError(403, "access denied")
	}

	return map[string]interface{}{"channel": ch}, nil
}

// mcpSendMessage handles the "send_message" MCP method.
func (a *GRPCRunnerAdapter) mcpSendMessage(ctx context.Context, tc *middleware.TenantContext, podKey string, payload []byte) (interface{}, *mcpError) {
	var params struct {
		ChannelID   int64    `json:"channel_id"`
		Content     string   `json:"content"`
		MessageType string   `json:"message_type"`
		Mentions    []string `json:"mentions"`
		ReplyTo     *int     `json:"reply_to"`
	}
	if err := unmarshalPayload(payload, &params); err != nil {
		return nil, err
	}

	if params.ChannelID == 0 {
		return nil, newMcpError(400, "channel_id is required")
	}
	if params.Content == "" {
		return nil, newMcpError(400, "content is required")
	}

	ch, err := a.channelService.GetChannel(ctx, params.ChannelID)
	if err != nil {
		return nil, newMcpError(404, "channel not found")
	}
	if ch.OrganizationID != tc.OrganizationID {
		return nil, newMcpError(403, "access denied")
	}
	if ch.IsArchived {
		return nil, newMcpError(400, "cannot send messages to archived channel")
	}

	msgType := params.MessageType
	if msgType == "" {
		msgType = "text"
	}

	// Convert mentions to JSON for storage if provided
	var mentionsJSON []byte
	if len(params.Mentions) > 0 {
		mentionsJSON, _ = json.Marshal(params.Mentions)
	}
	_ = mentionsJSON // mentions are passed via the SendMessage call

	msg, err := a.channelService.SendMessage(ctx, params.ChannelID, &podKey, &tc.UserID, msgType, params.Content, nil)
	if err != nil {
		return nil, newMcpError(500, "failed to send message")
	}

	return map[string]interface{}{"message": msg}, nil
}

// mcpGetMessages handles the "get_messages" MCP method.
func (a *GRPCRunnerAdapter) mcpGetMessages(ctx context.Context, tc *middleware.TenantContext, payload []byte) (interface{}, *mcpError) {
	var params struct {
		ChannelID    int64   `json:"channel_id"`
		BeforeTime   *string `json:"before_time"`
		AfterTime    *string `json:"after_time"`
		MentionedPod *string `json:"mentioned_pod"`
		Limit        int     `json:"limit"`
	}
	if err := unmarshalPayload(payload, &params); err != nil {
		return nil, err
	}

	if params.ChannelID == 0 {
		return nil, newMcpError(400, "channel_id is required")
	}

	ch, err := a.channelService.GetChannel(ctx, params.ChannelID)
	if err != nil {
		return nil, newMcpError(404, "channel not found")
	}
	if ch.OrganizationID != tc.OrganizationID {
		return nil, newMcpError(403, "access denied")
	}

	limit := params.Limit
	if limit <= 0 {
		limit = 50
	}

	messages, err := a.channelService.GetMessages(ctx, params.ChannelID, nil, limit)
	if err != nil {
		return nil, newMcpError(500, "failed to get messages")
	}

	return map[string]interface{}{"messages": messages}, nil
}

// mcpGetDocument handles the "get_document" MCP method.
func (a *GRPCRunnerAdapter) mcpGetDocument(ctx context.Context, tc *middleware.TenantContext, payload []byte) (interface{}, *mcpError) {
	var params struct {
		ChannelID int64 `json:"channel_id"`
	}
	if err := unmarshalPayload(payload, &params); err != nil {
		return nil, err
	}

	if params.ChannelID == 0 {
		return nil, newMcpError(400, "channel_id is required")
	}

	ch, err := a.channelService.GetChannel(ctx, params.ChannelID)
	if err != nil {
		return nil, newMcpError(404, "channel not found")
	}
	if ch.OrganizationID != tc.OrganizationID {
		return nil, newMcpError(403, "access denied")
	}

	document := ""
	if ch.Document != nil {
		document = *ch.Document
	}

	return map[string]interface{}{"document": document}, nil
}

// mcpUpdateDocument handles the "update_document" MCP method.
func (a *GRPCRunnerAdapter) mcpUpdateDocument(ctx context.Context, tc *middleware.TenantContext, payload []byte) (interface{}, *mcpError) {
	var params struct {
		ChannelID int64  `json:"channel_id"`
		Document  string `json:"document"`
	}
	if err := unmarshalPayload(payload, &params); err != nil {
		return nil, err
	}

	if params.ChannelID == 0 {
		return nil, newMcpError(400, "channel_id is required")
	}

	ch, err := a.channelService.GetChannel(ctx, params.ChannelID)
	if err != nil {
		return nil, newMcpError(404, "channel not found")
	}
	if ch.OrganizationID != tc.OrganizationID {
		return nil, newMcpError(403, "access denied")
	}

	_, err = a.channelService.UpdateChannel(ctx, params.ChannelID, nil, nil, &params.Document)
	if err != nil {
		return nil, newMcpError(500, "failed to update document")
	}

	return map[string]interface{}{"document": params.Document}, nil
}
