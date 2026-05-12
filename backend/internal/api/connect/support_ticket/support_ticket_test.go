package supportticketconnect

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	supportticketsvc "github.com/anthropics/agentsmesh/backend/internal/service/supportticket"
	supportticketv1 "github.com/anthropics/agentsmesh/proto/gen/go/support_ticket/v1"
)

func ctxAsUser(userID int64) context.Context {
	return middleware.SetTenant(context.Background(),
		&middleware.TenantContext{UserID: userID})
}

func connectCodeOf(t *testing.T, err error) connect.Code {
	t.Helper()
	var ce *connect.Error
	require.True(t, errors.As(err, &ce), "expected *connect.Error, got %v", err)
	return ce.Code()
}

// --- Auth guards ---

func TestListSupportTickets_NoAuth_Unauthenticated(t *testing.T) {
	srv := NewServer(nil)
	_, err := srv.ListSupportTickets(context.Background(),
		connect.NewRequest(&supportticketv1.ListSupportTicketsRequest{}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeUnauthenticated, connectCodeOf(t, err))
}

func TestGetSupportTicket_NoAuth_Unauthenticated(t *testing.T) {
	srv := NewServer(nil)
	_, err := srv.GetSupportTicket(context.Background(),
		connect.NewRequest(&supportticketv1.GetSupportTicketRequest{Id: 1}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeUnauthenticated, connectCodeOf(t, err))
}

func TestGetSupportTicket_MissingID_InvalidArgument(t *testing.T) {
	srv := NewServer(nil)
	_, err := srv.GetSupportTicket(ctxAsUser(42),
		connect.NewRequest(&supportticketv1.GetSupportTicketRequest{}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connectCodeOf(t, err))
}

func TestGetAttachmentURL_NoAuth_Unauthenticated(t *testing.T) {
	srv := NewServer(nil)
	_, err := srv.GetAttachmentURL(context.Background(),
		connect.NewRequest(&supportticketv1.GetAttachmentUrlRequest{AttachmentId: 1}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeUnauthenticated, connectCodeOf(t, err))
}

func TestGetAttachmentURL_MissingID_InvalidArgument(t *testing.T) {
	srv := NewServer(nil)
	_, err := srv.GetAttachmentURL(ctxAsUser(42),
		connect.NewRequest(&supportticketv1.GetAttachmentUrlRequest{}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connectCodeOf(t, err))
}

// --- mapSupportTicketError table ---

func TestMapSupportTicketError(t *testing.T) {
	cases := []struct {
		name string
		in   error
		want connect.Code
	}{
		{"ticket_not_found", supportticketsvc.ErrTicketNotFound, connect.CodeNotFound},
		{"attachment_not_found", supportticketsvc.ErrAttachmentNotFound, connect.CodeNotFound},
		{"access_denied", supportticketsvc.ErrAccessDenied, connect.CodePermissionDenied},
		{"invalid_category", supportticketsvc.ErrInvalidCategory, connect.CodeInvalidArgument},
		{"invalid_priority", supportticketsvc.ErrInvalidPriority, connect.CodeInvalidArgument},
		{"invalid_status", supportticketsvc.ErrInvalidStatus, connect.CodeInvalidArgument},
		{"invalid_transition", supportticketsvc.ErrInvalidTransition, connect.CodeInvalidArgument},
		{"file_too_large", supportticketsvc.ErrFileTooLarge, connect.CodeResourceExhausted},
		{"storage_error", supportticketsvc.ErrStorageError, connect.CodeUnavailable},
		{"generic", errors.New("boom"), connect.CodeInternal},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, connectCodeOf(t, mapSupportTicketError(tc.in)))
		})
	}
}

// --- userIDFromCtx ---

func TestUserIDFromCtx_NoTenant_Unauthenticated(t *testing.T) {
	_, err := userIDFromCtx(context.Background())
	require.Error(t, err)
	assert.Equal(t, connect.CodeUnauthenticated, connectCodeOf(t, err))
}

func TestUserIDFromCtx_ZeroUserID_Unauthenticated(t *testing.T) {
	ctx := middleware.SetTenant(context.Background(), &middleware.TenantContext{UserID: 0})
	_, err := userIDFromCtx(ctx)
	require.Error(t, err)
	assert.Equal(t, connect.CodeUnauthenticated, connectCodeOf(t, err))
}

func TestUserIDFromCtx_Ok(t *testing.T) {
	uid, err := userIDFromCtx(ctxAsUser(42))
	require.NoError(t, err)
	assert.Equal(t, int64(42), uid)
}

// --- normalizeListArgs ---

func TestNormalizeListArgs(t *testing.T) {
	cases := []struct {
		name             string
		inOff, inLim     int32
		wantOff, wantLim int32
	}{
		{"defaults", 0, 0, 0, 20},
		{"explicit_offset_zero", 0, 30, 0, 30},
		{"caps_to_100", 0, 200, 0, 20}, // out-of-range falls back to default
		{"valid_50_at_offset_60", 60, 50, 60, 50},
		{"negative_offset_clamped", -10, 30, 0, 30},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			off, lim := normalizeListArgs(tc.inOff, tc.inLim)
			assert.Equal(t, tc.wantOff, off)
			assert.Equal(t, tc.wantLim, lim)
		})
	}
}
