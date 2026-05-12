package supportticketconnect

import (
	"time"

	supportticketdomain "github.com/anthropics/agentsmesh/backend/internal/domain/supportticket"
	supportticketv1 "github.com/anthropics/agentsmesh/proto/gen/go/support_ticket/v1"
)

// toProtoTicket converts the GORM-backed domain model into the wire shape.
// AssignedAdminID is intentionally absent — it's not exposed on the
// user-facing endpoint (admin-only detail).
func toProtoTicket(t *supportticketdomain.SupportTicket) *supportticketv1.SupportTicket {
	if t == nil {
		return nil
	}
	out := &supportticketv1.SupportTicket{
		Id:        t.ID,
		UserId:    t.UserID,
		Title:     t.Title,
		Category:  t.Category,
		Status:    t.Status,
		Priority:  t.Priority,
		CreatedAt: t.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: t.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if t.ResolvedAt != nil {
		v := t.ResolvedAt.UTC().Format(time.RFC3339)
		out.ResolvedAt = &v
	}
	return out
}

// toProtoMessage converts a domain message + eager-loaded associations.
// Attachment.StorageKey is hidden from the wire (it's internal).
func toProtoMessage(m *supportticketdomain.SupportTicketMessage) *supportticketv1.SupportTicketMessage {
	if m == nil {
		return nil
	}
	out := &supportticketv1.SupportTicketMessage{
		Id:           m.ID,
		TicketId:     m.TicketID,
		UserId:       m.UserID,
		Content:      m.Content,
		IsAdminReply: m.IsAdminReply,
		CreatedAt:    m.CreatedAt.UTC().Format(time.RFC3339),
		Attachments:  make([]*supportticketv1.SupportTicketAttachment, 0, len(m.Attachments)),
	}
	if m.User != nil {
		out.User = &supportticketv1.SupportTicketUser{
			Id:    m.User.ID,
			Email: m.User.Email,
		}
		if m.User.Name != nil {
			out.User.Name = m.User.Name
		}
		if m.User.AvatarURL != nil {
			out.User.AvatarUrl = m.User.AvatarURL
		}
	}
	for i := range m.Attachments {
		out.Attachments = append(out.Attachments, toProtoAttachment(&m.Attachments[i]))
	}
	return out
}

func toProtoAttachment(a *supportticketdomain.SupportTicketAttachment) *supportticketv1.SupportTicketAttachment {
	if a == nil {
		return nil
	}
	out := &supportticketv1.SupportTicketAttachment{
		Id:           a.ID,
		TicketId:     a.TicketID,
		UploaderId:   a.UploaderID,
		OriginalName: a.OriginalName,
		MimeType:     a.MimeType,
		Size:         a.Size,
		CreatedAt:    a.CreatedAt.UTC().Format(time.RFC3339),
	}
	if a.MessageID != nil {
		out.MessageId = a.MessageID
	}
	return out
}

// normalizeListArgs picks defaults that match the REST handler's
// `normalizePagination(page=1, page_size=20)` — server-side default
// PageSize is 20 with a 100-row ceiling. Connect's `{offset, limit}`
// envelope (conventions §8) translates: offset=0 means page 1.
func normalizeListArgs(offset, limit int32) (int32, int32) {
	if limit < 1 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return offset, limit
}
