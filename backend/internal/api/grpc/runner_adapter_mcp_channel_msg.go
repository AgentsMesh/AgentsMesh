package grpc

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	channelDomain "github.com/anthropics/agentsmesh/backend/internal/domain/channel"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
)

// mcpSendMessage handles the "send_message" MCP method.
// Agents send plain text + optional mentions; we build structured MessageContent server-side.
func (a *GRPCRunnerAdapter) mcpSendMessage(ctx context.Context, tc *middleware.TenantContext, podKey string, payload []byte) (interface{}, *mcpError) {
	var params struct {
		ChannelID         int64                        `json:"channel_id"`
		Content           string                       `json:"content"`
		StructuredContent *channelDomain.MessageContent `json:"structured_content"`
		Mentions          []string                     `json:"mentions"`
	}
	if err := unmarshalPayload(payload, &params); err != nil {
		return nil, err
	}
	if params.ChannelID == 0 {
		return nil, newMcpError(400, "channel_id is required")
	}
	if params.Content == "" && params.StructuredContent == nil {
		return nil, newMcpError(400, "content or structured_content is required")
	}

	ch, mcpErr := a.validateChannelAccess(ctx, tc, params.ChannelID)
	if mcpErr != nil {
		return nil, mcpErr
	}
	if ch.IsArchived {
		return nil, newMcpError(400, "cannot send messages to archived channel")
	}

	// Use structured content if provided; otherwise build from plain text + mentions
	var content channelDomain.MessageContent
	if params.StructuredContent != nil {
		content = *params.StructuredContent
	} else {
		mentionMap := make(map[string]struct{ typ, key string })
		for _, m := range params.Mentions {
			parts := strings.SplitN(m, ":", 2)
			if len(parts) == 2 && (parts[0] == "user" || parts[0] == "pod") {
				mentionMap[parts[1]] = struct{ typ, key string }{parts[0], parts[1]}
			}
		}
		content = buildTextContent(params.Content, mentionMap)
	}

	msg, err := a.channelService.SendMessage(ctx, params.ChannelID, &podKey, &tc.UserID, content, nil)
	if err != nil {
		return nil, newMcpError(400, "failed to send message: "+err.Error())
	}
	return map[string]interface{}{"message": messageToMCP(msg)}, nil
}

// messageToMCP transforms a backend Message into the format expected by runner/agent.
// Runner expects: content (string), mentions ([]string in "type:key" format).
// Backend has:    body (string), content (*MessageContent JSONB), mentions (MessageMentions).
func messageToMCP(msg *channelDomain.Message) map[string]interface{} {
	result := map[string]interface{}{
		"id":           msg.ID,
		"channel_id":   msg.ChannelID,
		"content":      msg.Body,
		"message_type": msg.MessageType,
		"created_at":   msg.CreatedAt.Format(time.RFC3339),
	}
	if msg.SenderPod != nil {
		result["sender_pod"] = *msg.SenderPod
	}
	if msg.SenderUserID != nil {
		result["sender_user_id"] = *msg.SenderUserID
	}
	if msg.ReplyTo != nil {
		result["reply_to"] = *msg.ReplyTo
	}
	if msg.EditedAt != nil {
		result["edited_at"] = msg.EditedAt.Format(time.RFC3339)
	}

	// Convert MessageMentions → []string in "type:key" format for agent compatibility
	var mentions []string
	for _, pk := range msg.Mentions.Pods {
		mentions = append(mentions, "pod:"+pk)
	}
	for _, uid := range msg.Mentions.Users {
		mentions = append(mentions, fmt.Sprintf("user:%d", uid))
	}
	if len(mentions) > 0 {
		result["mentions"] = mentions
	}
	return result
}

func messagesToMCP(msgs []*channelDomain.Message) []map[string]interface{} {
	result := make([]map[string]interface{}, len(msgs))
	for i, msg := range msgs {
		result[i] = messageToMCP(msg)
	}
	return result
}

// buildTextContent converts plain text into structured MessageContent.
func buildTextContent(text string, mentions map[string]struct{ typ, key string }) channelDomain.MessageContent {
	lines := strings.Split(text, "\n")
	blocks := make([]channelDomain.Block, 0, len(lines))
	for _, line := range lines {
		elements := parseMCPLine(line, mentions)
		blocks = append(blocks, channelDomain.Block{Type: "paragraph", Elements: elements})
	}
	return channelDomain.MessageContent{Kind: "text", Blocks: blocks}
}

func parseMCPLine(line string, mentions map[string]struct{ typ, key string }) []channelDomain.InlineElement {
	if len(mentions) == 0 {
		return []channelDomain.InlineElement{{Type: channelDomain.InlineText, Text: line}}
	}
	// Sort keys longest-first to avoid prefix ambiguity (e.g., "alice" vs "alice-admin")
	keys := make([]string, 0, len(mentions))
	for k := range mentions {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return len(keys[i]) > len(keys[j]) })

	parts := strings.Split(line, "@")
	if len(parts) == 1 {
		return []channelDomain.InlineElement{{Type: channelDomain.InlineText, Text: line}}
	}
	var elements []channelDomain.InlineElement
	for i, part := range parts {
		if i == 0 {
			if part != "" {
				elements = append(elements, channelDomain.InlineElement{Type: channelDomain.InlineText, Text: part})
			}
			continue
		}
		matched := false
		for _, mKey := range keys {
			mRef := mentions[mKey]
			if strings.HasPrefix(part, mKey) {
				elements = append(elements, channelDomain.InlineElement{
					Type: channelDomain.InlineMention, EntityType: mRef.typ, EntityKey: mRef.key, Display: mKey,
				})
				rest := part[len(mKey):]
				if rest != "" {
					elements = append(elements, channelDomain.InlineElement{Type: channelDomain.InlineText, Text: rest})
				}
				matched = true
				break
			}
		}
		if !matched {
			elements = append(elements, channelDomain.InlineElement{Type: channelDomain.InlineText, Text: "@" + part})
		}
	}
	return elements
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
	if _, mcpErr := a.validateChannelAccess(ctx, tc, params.ChannelID); mcpErr != nil {
		return nil, mcpErr
	}

	limit := params.Limit
	if limit <= 0 {
		limit = 50
	} else if limit > 100 {
		limit = 100
	}

	if params.MentionedPod != nil && *params.MentionedPod != "" {
		messages, hasMore, err := a.channelService.GetMessagesMentioning(ctx, params.ChannelID, *params.MentionedPod, limit)
		if err != nil {
			return nil, newMcpError(500, "failed to get messages")
		}
		return map[string]interface{}{"messages": messagesToMCP(messages), "has_more": hasMore}, nil
	}

	var before, after *time.Time
	if params.BeforeTime != nil && *params.BeforeTime != "" {
		if t, err := time.Parse(time.RFC3339, *params.BeforeTime); err == nil {
			before = &t
		} else {
			return nil, newMcpError(400, "invalid before_time format, expected RFC3339")
		}
	}
	if params.AfterTime != nil && *params.AfterTime != "" {
		if t, err := time.Parse(time.RFC3339, *params.AfterTime); err == nil {
			after = &t
		} else {
			return nil, newMcpError(400, "invalid after_time format, expected RFC3339")
		}
	}

	messages, hasMore, err := a.channelService.GetMessages(ctx, params.ChannelID, before, after, limit)
	if err != nil {
		return nil, newMcpError(500, "failed to get messages")
	}
	return map[string]interface{}{"messages": messagesToMCP(messages), "has_more": hasMore}, nil
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
	ch, mcpErr := a.validateChannelAccess(ctx, tc, params.ChannelID)
	if mcpErr != nil {
		return nil, mcpErr
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
	if _, mcpErr := a.validateChannelAccess(ctx, tc, params.ChannelID); mcpErr != nil {
		return nil, mcpErr
	}

	_, err := a.channelService.UpdateChannel(ctx, params.ChannelID, nil, nil, &params.Document)
	if err != nil {
		return nil, newMcpError(500, "failed to update document")
	}
	return map[string]interface{}{"document": params.Document}, nil
}
