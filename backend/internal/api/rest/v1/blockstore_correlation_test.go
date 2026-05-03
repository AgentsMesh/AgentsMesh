package v1

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace"

	"github.com/anthropics/agentsmesh/backend/internal/domain/blockstore"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
)

// REST-side correlation extraction. Mirrors gRPC actorFromTenant tests in
// backend/internal/api/grpc/runner_adapter_mcp_block_test.go — both
// transports must populate ActorContext audit fields the same way so
// BlockOp.Context lines up regardless of caller.

func TestActorFrom_PopulatesCorrelationFromGinCtx(t *testing.T) {
	traceID, _ := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	spanID, _ := trace.SpanIDFromHex("00f067aa0ba902b7")
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, "/blocks/apply-ops", nil)
	req = req.WithContext(trace.ContextWithSpanContext(context.Background(), sc))
	req.Header.Set("User-Agent", "agentsmesh-web/2.1")
	req.RemoteAddr = "203.0.113.42:54321"
	c.Request = req
	c.Set("tenant", &middleware.TenantContext{
		OrganizationID: 7,
		UserID:         100,
		UserRole:       "member",
	})

	actor, ok := actorFrom(c)
	assert.True(t, ok, "actorFrom must succeed when tenant ctx is present")
	assert.Equal(t, "4bf92f3577b34da6a3ce929d0e0e4736", actor.TraceID,
		"trace id must round-trip from otelgin span into ActorContext")
	assert.Equal(t, actor.TraceID, actor.RequestID,
		"request id aliases trace id until X-Request-Id middleware lands")
	assert.NotEmpty(t, actor.IP, "gin.ClientIP must populate IP from RemoteAddr")
	assert.Equal(t, "agentsmesh-web/2.1", actor.UserAgent)
	assert.Equal(t, blockstore.ActorUser, actor.ActorType)
	assert.Equal(t, int64(100), actor.UserID)
	assert.Equal(t, int64(100), actor.ActorID,
		"REST path attributes ActorID to the user, unlike gRPC agent path")
}

func TestActorFrom_EmptyCorrelationOnRawCtx(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, "/blocks/apply-ops", nil)
	c.Request = req
	c.Set("tenant", &middleware.TenantContext{
		OrganizationID: 7,
		UserID:         100,
	})

	actor, ok := actorFrom(c)
	assert.True(t, ok)
	assert.Empty(t, actor.TraceID, "no otel span ⇒ empty trace id, never a placeholder")
	assert.Empty(t, actor.RequestID)
	// IP/UserAgent may be set from gin's default client-IP heuristic; we
	// only assert TraceID stays empty so the buildOpContext omission contract
	// (no key when value is "") is exercised.
}

func TestActorFrom_AbortsWhenTenantMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, "/blocks/apply-ops", nil)
	c.Request = req
	// No tenant set — middleware bug or test misconfig. Handler must abort
	// with 401 rather than synthesise a phantom actor.

	_, ok := actorFrom(c)
	assert.False(t, ok, "actorFrom must refuse to proceed without tenant ctx")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
