package channel

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
)

// SendMessage sends a message with structured content to a channel.
// The server extracts body (plain text) and mentions from the content AST.
func (s *Service) SendMessage(ctx context.Context, channelID int64, senderPod *string, senderUserID *int64, content channel.MessageContent, replyTo *int64) (*channel.Message, error) {
	ch, err := s.GetChannel(ctx, channelID)
	if err != nil {
		return nil, err
	}

	if ch.IsArchived {
		return nil, ErrChannelArchived
	}

	if err := content.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidContent, err)
	}

	content.SchemaVersion = 1
	body := extractBody(&content)
	hasAttachment := strings.TrimSpace(content.AttachmentKey) != ""
	if strings.TrimSpace(body) == "" && !hasAttachment {
		return nil, ErrEmptyContent
	}
	mentions := extractMentions(&content)

	var mentionResult *MentionResult
	if len(mentions.Pods) > 0 || len(mentions.Users) > 0 {
		mentionResult = &MentionResult{UserIDs: mentions.Users, PodKeys: mentions.Pods}
	}

	if senderUserID != nil {
		if ch.IsPublic() {
			_ = s.repo.AddMemberWithRole(ctx, channelID, *senderUserID, channel.RoleMember)
		} else {
			if err := s.requireMembership(ctx, channelID, *senderUserID); err != nil {
				return nil, err
			}
		}
	}

	messageType := channel.MessageTypeText
	if hasAttachment && strings.TrimSpace(body) == "" {
		messageType = channel.MessageTypeAttachment
	}

	msg := &channel.Message{
		ChannelID:    channelID,
		SenderPod:    senderPod,
		SenderUserID: senderUserID,
		MessageType:  messageType,
		Body:         body,
		Content:      &content,
		Mentions:     mentions,
		ReplyTo:      replyTo,
	}

	if err := s.repo.CreateMessage(ctx, msg); err != nil {
		return nil, err
	}

	_ = s.repo.TouchChannel(ctx, channelID)

	if len(s.postSendHooks) > 0 {
		mc := &MessageContext{Channel: ch, Message: msg, Mentions: mentionResult}
		for _, hook := range s.postSendHooks {
			if err := hook(ctx, mc); err != nil {
				slog.ErrorContext(ctx, "post-send hook failed", "error", err)
			}
		}
	}

	return msg, nil
}

func extractBody(c *channel.MessageContent) string {
	if c == nil || len(c.Blocks) == 0 {
		return ""
	}
	var paragraphs []string
	extractBlocksBody(c.Blocks, &paragraphs)
	return strings.Join(paragraphs, "\n")
}

func extractBlocksBody(blocks []channel.Block, out *[]string) {
	for _, block := range blocks {
		var sb strings.Builder
		writeInlineElements(&sb, block.Elements)
		for i, item := range block.Items {
			if sb.Len() > 0 || i > 0 {
				sb.WriteString("\n")
			}
			writeInlineElements(&sb, item)
		}
		if text := sb.String(); text != "" {
			*out = append(*out, text)
		}
		if len(block.Children) > 0 {
			extractBlocksBody(block.Children, out)
		}
	}
}

func writeInlineElements(sb *strings.Builder, elements []channel.InlineElement) {
	for _, el := range elements {
		switch el.Type {
		case channel.InlineText:
			sb.WriteString(el.Text)
		case channel.InlineMention:
			if el.Display != "" {
				sb.WriteString("@" + el.Display)
			} else {
				sb.WriteString("@" + el.EntityKey)
			}
		case channel.InlineLink:
			sb.WriteString(el.Text)
		case channel.InlineLinebreak:
			sb.WriteString("\n")
		}
	}
}

func extractMentions(c *channel.MessageContent) channel.MessageMentions {
	var m channel.MessageMentions
	if c == nil {
		return m
	}
	podsSeen := make(map[string]bool)
	usersSeen := make(map[int64]bool)
	extractBlocksMentions(c.Blocks, &m, podsSeen, usersSeen)
	return m
}

func extractBlocksMentions(blocks []channel.Block, m *channel.MessageMentions, podsSeen map[string]bool, usersSeen map[int64]bool) {
	for _, block := range blocks {
		collectMentionsFromElements(block.Elements, m, podsSeen, usersSeen)
		for _, item := range block.Items {
			collectMentionsFromElements(item, m, podsSeen, usersSeen)
		}
		if len(block.Children) > 0 {
			extractBlocksMentions(block.Children, m, podsSeen, usersSeen)
		}
	}
}

func collectMentionsFromElements(elements []channel.InlineElement, m *channel.MessageMentions, podsSeen map[string]bool, usersSeen map[int64]bool) {
	for _, el := range elements {
		if el.Type != channel.InlineMention {
			continue
		}
		switch el.EntityType {
		case channel.EntityPod:
			if !podsSeen[el.EntityKey] {
				podsSeen[el.EntityKey] = true
				m.Pods = append(m.Pods, el.EntityKey)
			}
		case channel.EntityUser:
			if id, err := strconv.ParseInt(el.EntityKey, 10, 64); err == nil && !usersSeen[id] {
				usersSeen[id] = true
				m.Users = append(m.Users, id)
			}
		case channel.EntityChannel:
			m.Channel = true
		}
	}
}

// GetMessages returns messages for a channel.
func (s *Service) GetMessages(ctx context.Context, channelID int64, before *time.Time, after *time.Time, limit int) ([]*channel.Message, bool, error) {
	messages, err := s.repo.GetMessages(ctx, channelID, before, after, limit+1)
	if err != nil {
		return nil, false, err
	}
	hasMore := len(messages) > limit
	if hasMore {
		messages = messages[:limit]
	}
	if after == nil || before != nil {
		slices.Reverse(messages)
	}
	return messages, hasMore, nil
}

// SendSystemMessage sends a system message (body-only, no structured content).
func (s *Service) SendSystemMessage(ctx context.Context, channelID int64, body string) (*channel.Message, error) {
	ch, err := s.GetChannel(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if ch.IsArchived {
		return nil, ErrChannelArchived
	}

	msg := &channel.Message{
		ChannelID:   channelID,
		MessageType: channel.MessageTypeSystem,
		Body:        body,
	}
	if err := s.repo.CreateMessage(ctx, msg); err != nil {
		return nil, err
	}
	_ = s.repo.TouchChannel(ctx, channelID)

	if len(s.postSendHooks) > 0 {
		mc := &MessageContext{Channel: ch, Message: msg}
		for _, hook := range s.postSendHooks {
			if err := hook(ctx, mc); err != nil {
				slog.Error("post-send hook failed", "error", err)
			}
		}
	}
	return msg, nil
}

// SendMessageAsUser sends a message as a user (human) to a channel.
func (s *Service) SendMessageAsUser(ctx context.Context, channelID int64, userID int64, content channel.MessageContent) (*channel.Message, error) {
	return s.SendMessage(ctx, channelID, nil, &userID, content, nil)
}

// SendMessageAsPod sends a message as a pod (agent) to a channel.
func (s *Service) SendMessageAsPod(ctx context.Context, channelID int64, podKey string, content channel.MessageContent) (*channel.Message, error) {
	return s.SendMessage(ctx, channelID, &podKey, nil, content, nil)
}

// GetMessagesMentioning returns messages mentioning a specific pod.
func (s *Service) GetMessagesMentioning(ctx context.Context, channelID int64, podKey string, limit int) ([]*channel.Message, bool, error) {
	messages, hasMore, err := s.repo.GetMessagesMentioning(ctx, channelID, podKey, limit)
	if err != nil {
		return nil, false, err
	}
	slices.Reverse(messages)
	return messages, hasMore, nil
}

// GetMessagesByCursor returns messages before a given message ID (cursor-based pagination).
func (s *Service) GetMessagesByCursor(ctx context.Context, channelID int64, beforeID int64, limit int) ([]*channel.Message, bool, error) {
	messages, err := s.repo.GetMessagesBefore(ctx, channelID, beforeID, limit+1)
	if err != nil {
		return nil, false, err
	}
	hasMore := len(messages) > limit
	if hasMore {
		messages = messages[:limit]
	}
	slices.Reverse(messages)
	return messages, hasMore, nil
}

// GetRecentMessages returns the most recent messages from a channel.
func (s *Service) GetRecentMessages(ctx context.Context, channelID int64, limit int) ([]*channel.Message, error) {
	messages, err := s.repo.GetRecentMessages(ctx, channelID, limit)
	if err != nil {
		return nil, err
	}
	slices.Reverse(messages)
	return messages, nil
}

// SearchMessages searches channel messages by full-text query on body.
func (s *Service) SearchMessages(ctx context.Context, channelID int64, query string, limit int) ([]*channel.Message, error) {
	if limit <= 0 {
		limit = 20
	} else if limit > 100 {
		limit = 100
	}
	return s.repo.SearchMessages(ctx, channelID, query, limit)
}