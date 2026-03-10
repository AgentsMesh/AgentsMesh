package mcp

import (
	"context"

	"github.com/anthropics/agentsmesh/runner/internal/mcp/tools"
)

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
