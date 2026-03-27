package mcp

import (
	"context"
	"fmt"

	"github.com/anthropics/agentsmesh/runner/internal/mcp/tools"
)

// Channel Messaging & Document Tools

func (s *HTTPServer) createSendChannelMessageTool() *MCPTool {
	return &MCPTool{
		Name:        "send_channel_message",
		Description: "Send a message to a collaboration channel.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"channel_id": map[string]interface{}{
					"type":        "integer",
					"description": "The ID of the channel to send to",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "The message content",
				},
				"message_type": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"text", "system"},
					"description": "Type of message (default: text)",
				},
				"mentions": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "Pod keys to mention in the message",
				},
				"reply_to": map[string]interface{}{
					"type":        "integer",
					"description": "Message ID to reply to (optional)",
				},
			},
			"required": []string{"channel_id", "content"},
		},
		Handler: func(ctx context.Context, client tools.CollaborationClient, args map[string]interface{}) (interface{}, error) {
			channelID := getIntArg(args, "channel_id")
			content := getStringArg(args, "content")
			msgType := getStringArg(args, "message_type")
			mentions := getStringSliceArg(args, "mentions")
			replyTo := getIntPtrArg(args, "reply_to")

			if channelID == 0 || content == "" {
				return nil, fmt.Errorf("channel_id and content are required")
			}

			if msgType == "" {
				msgType = "text"
			}

			return client.SendMessage(ctx, channelID, content, tools.ChannelMessageType(msgType), mentions, replyTo)
		},
	}
}

func (s *HTTPServer) createGetChannelMessagesTool() *MCPTool {
	return &MCPTool{
		Name:        "get_channel_messages",
		Description: "Get messages from a collaboration channel.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"channel_id": map[string]interface{}{
					"type":        "integer",
					"description": "The ID of the channel",
				},
				"before_time": map[string]interface{}{
					"type":        "string",
					"description": "Get messages before this timestamp (ISO 8601)",
				},
				"after_time": map[string]interface{}{
					"type":        "string",
					"description": "Get messages after this timestamp (ISO 8601)",
				},
				"mentioned_pod": map[string]interface{}{
					"type":        "string",
					"description": "Filter to messages mentioning this pod",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum messages to return (default: 50)",
				},
			},
			"required": []string{"channel_id"},
		},
		Handler: func(ctx context.Context, client tools.CollaborationClient, args map[string]interface{}) (interface{}, error) {
			channelID := getIntArg(args, "channel_id")
			if channelID == 0 {
				return nil, fmt.Errorf("channel_id is required")
			}

			var beforeTime, afterTime, mentionedPod *string
			if v := getStringArg(args, "before_time"); v != "" {
				beforeTime = &v
			}
			if v := getStringArg(args, "after_time"); v != "" {
				afterTime = &v
			}
			if v := getStringArg(args, "mentioned_pod"); v != "" {
				mentionedPod = &v
			}

			limit := getIntArg(args, "limit")
			if limit == 0 {
				limit = 50
			}

			result, err := client.GetMessages(ctx, channelID, beforeTime, afterTime, mentionedPod, limit)
			if err != nil {
				return nil, err
			}
			return tools.ChannelMessageList(result), nil
		},
	}
}

func (s *HTTPServer) createGetChannelDocumentTool() *MCPTool {
	return &MCPTool{
		Name:        "get_channel_document",
		Description: "Get the shared document from a channel. Channels can have a collaborative document that all members can view and edit.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"channel_id": map[string]interface{}{
					"type":        "integer",
					"description": "The ID of the channel",
				},
			},
			"required": []string{"channel_id"},
		},
		Handler: func(ctx context.Context, client tools.CollaborationClient, args map[string]interface{}) (interface{}, error) {
			channelID := getIntArg(args, "channel_id")
			if channelID == 0 {
				return nil, fmt.Errorf("channel_id is required")
			}
			return client.GetDocument(ctx, channelID)
		},
	}
}

func (s *HTTPServer) createUpdateChannelDocumentTool() *MCPTool {
	return &MCPTool{
		Name:        "update_channel_document",
		Description: "Update the shared document in a channel. This replaces the entire document content.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"channel_id": map[string]interface{}{
					"type":        "integer",
					"description": "The ID of the channel",
				},
				"document": map[string]interface{}{
					"type":        "string",
					"description": "The new document content",
				},
			},
			"required": []string{"channel_id", "document"},
		},
		Handler: func(ctx context.Context, client tools.CollaborationClient, args map[string]interface{}) (interface{}, error) {
			channelID := getIntArg(args, "channel_id")
			document := getStringArg(args, "document")

			if channelID == 0 {
				return nil, fmt.Errorf("channel_id is required")
			}

			err := client.UpdateDocument(ctx, channelID, document)
			if err != nil {
				return nil, err
			}
			return "Document updated successfully", nil
		},
	}
}
