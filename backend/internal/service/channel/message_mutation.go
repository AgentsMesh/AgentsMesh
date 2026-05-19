package channel

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
	"github.com/anthropics/agentsmesh/backend/internal/infra/eventbus"
)

func (s *Service) EditMessage(ctx context.Context, channelID, messageID, senderUserID int64, newContent channel.MessageContent) (*channel.Message, error) {
	ch, err := s.GetChannel(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if ch.IsArchived {
		return nil, ErrChannelArchived
	}

	if err := newContent.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidContent, err)
	}

	newContent.SchemaVersion = 1

	msg, err := s.repo.GetMessageByID(ctx, messageID)
	if err != nil {
		return nil, err
	}
	if msg == nil || msg.ChannelID != channelID {
		return nil, ErrMessageNotFound
	}
	if msg.SenderUserID == nil || *msg.SenderUserID != senderUserID {
		return nil, ErrNotMessageSender
	}

	newBody := extractBody(&newContent)
	newMentions := extractMentions(&newContent)

	edit := &channel.MessageEdit{
		MessageID:       messageID,
		EditorUserID:    &senderUserID,
		PreviousBody:    msg.Body,
		PreviousContent: msg.Content,
	}
	if err := s.repo.SaveMessageEdit(ctx, edit); err != nil {
		slog.Error("failed to save message edit history", "message_id", messageID, "error", err)
	}

	if err := s.repo.UpdateMessage(ctx, messageID, newBody, &newContent, newMentions); err != nil {
		return nil, err
	}

	editedData := eventbus.ChannelMessageEditedData{
		ID:        messageID,
		ChannelID: channelID,
		Body:      newBody,
		Content:   &newContent,
		Mentions:  newMentions,
		EditedAt:  time.Now().Format(time.RFC3339),
	}
	s.publishChannelEvent(ch.OrganizationID, ch.ID, eventbus.EventChannelMessageEdited, editedData)

	return s.repo.GetMessageByID(ctx, messageID)
}

func (s *Service) DeleteMessage(ctx context.Context, channelID, messageID, senderUserID int64) error {
	ch, err := s.GetChannel(ctx, channelID)
	if err != nil {
		return err
	}
	if ch.IsArchived {
		return ErrChannelArchived
	}

	msg, err := s.repo.GetMessageByID(ctx, messageID)
	if err != nil {
		return err
	}
	if msg == nil || msg.ChannelID != channelID {
		return ErrMessageNotFound
	}
	if msg.SenderUserID == nil || *msg.SenderUserID != senderUserID {
		return ErrNotMessageSender
	}

	if err := s.repo.SoftDeleteMessage(ctx, messageID); err != nil {
		return err
	}

	deletedData := eventbus.ChannelMessageDeletedData{
		ID:        messageID,
		ChannelID: channelID,
	}
	s.publishChannelEvent(ch.OrganizationID, ch.ID, eventbus.EventChannelMessageDeleted, deletedData)

	return nil
}

func (s *Service) publishChannelEvent(orgID, channelID int64, eventType eventbus.EventType, data interface{}) {
	if s.eventBus == nil {
		return
	}
	payload, err := json.Marshal(data)
	if err != nil {
		slog.Error("failed to marshal message event", "error", err)
		return
	}

	var targetUserIDs []int64
	targetUserIDs, _ = s.repo.GetMemberUserIDs(context.Background(), channelID)

	ctx := context.Background()
	s.eventBus.Publish(ctx, &eventbus.Event{
		Type:           eventType,
		Category:       eventbus.CategoryEntity,
		OrganizationID: orgID,
		EntityType:     "channel",
		EntityID:       fmt.Sprintf("%d", channelID),
		Data:           payload,
		Timestamp:      time.Now().UnixMilli(),
		TargetUserIDs:  targetUserIDs,
	})
}
