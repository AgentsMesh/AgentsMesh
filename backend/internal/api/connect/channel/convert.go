package channelconnect

import (
	"encoding/json"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
	channelv1 "github.com/anthropics/agentsmesh/proto/gen/go/channel/v1"
)

// toProtoChannel converts the GORM-backed Channel to the protobuf wire shape.
// is_member / member_count are computed fields populated by the service layer
// for the GetChannelForUser path; ListChannels emits them when present.
func toProtoChannel(c *channel.Channel) *channelv1.Channel {
	if c == nil {
		return nil
	}
	out := &channelv1.Channel{
		Id:             c.ID,
		OrganizationId: c.OrganizationID,
		Name:           c.Name,
		Visibility:     c.Visibility,
		IsArchived:     c.IsArchived,
		IsMember:       c.IsMember,
		MemberCount:    c.MemberCount,
		CreatedAt:      c.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:      c.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if c.Description != nil {
		s := *c.Description
		out.Description = &s
	}
	if c.Document != nil {
		s := *c.Document
		out.Document = &s
	}
	if c.RepositoryID != nil {
		v := *c.RepositoryID
		out.RepositoryId = &v
	}
	if c.TicketID != nil {
		v := *c.TicketID
		out.TicketId = &v
	}
	if c.CreatedByPod != nil {
		s := *c.CreatedByPod
		out.CreatedByPod = &s
	}
	if c.CreatedByUserID != nil {
		v := *c.CreatedByUserID
		out.CreatedByUserId = &v
	}
	return out
}

// toProtoMessage converts a domain message to the wire shape. Rich content
// (Block AST) ships as opaque JSON in `content_json`. Sender enrichment
// (sender_user / sender_pod_info) is populated only when the service-layer
// loader joined the user / pod row in.
func toProtoMessage(m *channel.Message) *channelv1.ChannelMessage {
	if m == nil {
		return nil
	}
	out := &channelv1.ChannelMessage{
		Id:          m.ID,
		ChannelId:   m.ChannelID,
		MessageType: m.MessageType,
		Body:        m.Body,
		IsDeleted:   m.IsDeleted,
		CreatedAt:   m.CreatedAt.UTC().Format(time.RFC3339),
	}
	if m.SenderPod != nil {
		s := *m.SenderPod
		out.SenderPod = &s
	}
	if m.SenderUserID != nil {
		v := *m.SenderUserID
		out.SenderUserId = &v
	}
	if m.ReplyTo != nil {
		v := *m.ReplyTo
		out.ReplyTo = &v
	}
	if m.EditedAt != nil {
		s := m.EditedAt.UTC().Format(time.RFC3339)
		out.EditedAt = &s
	}
	if m.Content != nil {
		if b, err := json.Marshal(m.Content); err == nil {
			s := string(b)
			out.ContentJson = &s
		}
	}
	// Mentions has a default zero-value; marshal unconditionally so callers
	// reading `mentions_json` parse an empty `{}` rather than guessing.
	if b, err := json.Marshal(m.Mentions); err == nil {
		s := string(b)
		out.MentionsJson = &s
	}
	if m.SenderUser != nil {
		out.SenderUser = &channelv1.ChannelMessageSenderUser{
			Id:       m.SenderUser.ID,
			Username: m.SenderUser.Username,
		}
		if m.SenderUser.Name != nil {
			s := *m.SenderUser.Name
			out.SenderUser.Name = &s
		}
		if m.SenderUser.AvatarURL != nil {
			s := *m.SenderUser.AvatarURL
			out.SenderUser.AvatarUrl = &s
		}
	}
	if m.SenderPodInfo != nil {
		out.SenderPodInfo = &channelv1.ChannelMessageSenderPod{
			PodKey: m.SenderPodInfo.PodKey,
		}
		if m.SenderPodInfo.Alias != nil {
			s := *m.SenderPodInfo.Alias
			out.SenderPodInfo.Alias = &s
		}
	}
	return out
}

func toProtoMember(m channel.Member) *channelv1.ChannelMember {
	return &channelv1.ChannelMember{
		ChannelId: m.ChannelID,
		UserId:    m.UserID,
		Role:      m.Role,
		IsMuted:   m.IsMuted,
		JoinedAt:  m.JoinedAt.UTC().Format(time.RFC3339),
	}
}

func toProtoChannelPod(p *agentpod.Pod) *channelv1.ChannelPod {
	if p == nil {
		return nil
	}
	out := &channelv1.ChannelPod{
		Id:          p.ID,
		PodKey:      p.PodKey,
		Status:      p.Status,
		AgentStatus: p.AgentStatus,
	}
	if p.Alias != nil {
		s := *p.Alias
		out.Alias = &s
	}
	return out
}

// optionalString returns *p as an unaliased copy when non-nil. Matches the
// pattern used by REST handlers for converting *string PATCH inputs.
func optionalString(p *string) *string {
	if p == nil {
		return nil
	}
	s := *p
	return &s
}

// defaultListLimit / defaultListOffset preserve REST behavior — list
// channels paginates at 50 with offset 0. Conventions §5: explicit zero
// is preserved (only absent or non-positive defaults).
func defaultListLimit(p *int32, fallback int32) int32 {
	if p == nil || *p <= 0 {
		return fallback
	}
	return *p
}

func defaultListOffset(p *int32) int32 {
	if p == nil || *p < 0 {
		return 0
	}
	return *p
}
