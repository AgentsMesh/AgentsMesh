package channel

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
)

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

	// Reload with preloaded SenderUser + SenderPodInfo.Agent so the immediate
	// RPC response and downstream event broadcast carry the same shape as
	// list endpoints (which all preload). Falling back on error preserves
	// the message create — losing the join is bad, losing the message worse.
	if loaded, err := s.repo.GetMessageByID(ctx, msg.ID); err == nil && loaded != nil {
		msg = loaded
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

func (s *Service) SendMessageAsUser(ctx context.Context, channelID int64, userID int64, content channel.MessageContent) (*channel.Message, error) {
	return s.SendMessage(ctx, channelID, nil, &userID, content, nil)
}

func (s *Service) SendMessageAsPod(ctx context.Context, channelID int64, podKey string, content channel.MessageContent) (*channel.Message, error) {
	return s.SendMessage(ctx, channelID, &podKey, nil, content, nil)
}

func (s *Service) GetMessagesMentioning(ctx context.Context, channelID int64, podKey string, limit int) ([]*channel.Message, bool, error) {
	messages, hasMore, err := s.repo.GetMessagesMentioning(ctx, channelID, podKey, limit)
	if err != nil {
		return nil, false, err
	}
	slices.Reverse(messages)
	return messages, hasMore, nil
}

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

func (s *Service) GetRecentMessages(ctx context.Context, channelID int64, limit int) ([]*channel.Message, error) {
	messages, err := s.repo.GetRecentMessages(ctx, channelID, limit)
	if err != nil {
		return nil, err
	}
	slices.Reverse(messages)
	return messages, nil
}

func (s *Service) SearchMessages(ctx context.Context, channelID int64, query string, limit int) ([]*channel.Message, error) {
	if limit <= 0 {
		limit = 20
	} else if limit > 100 {
		limit = 100
	}
	return s.repo.SearchMessages(ctx, channelID, query, limit)
}
