package v1

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/anthropics/agentsmesh/backend/internal/domain/blockstore"
	blockstoreservice "github.com/anthropics/agentsmesh/backend/internal/service/blockstore"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
)

type BlockstoreHandler struct {
	service *blockstoreservice.Service
}

func NewBlockstoreHandler(svc *blockstoreservice.Service) *BlockstoreHandler {
	return &BlockstoreHandler{service: svc}
}

func actorFrom(c *gin.Context) (blockstoreservice.ActorContext, bool) {
	tc := middleware.GetTenant(c)
	if tc == nil {
		apierr.AbortUnauthorized(c, apierr.AUTH_REQUIRED, "tenant context missing")
		return blockstoreservice.ActorContext{}, false
	}
	traceID := traceIDFromContext(c.Request.Context())
	return blockstoreservice.ActorContext{
		UserID:    tc.UserID,
		OrgID:     tc.OrganizationID,
		ActorType: blockstore.ActorUser,
		ActorID:   tc.UserID,
		TraceID:   traceID,
		RequestID: traceID,
		IP:        c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
	}, true
}

func traceIDFromContext(ctx context.Context) string {
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		return sc.TraceID().String()
	}
	return ""
}

// errors.Is required: wrapped errors (service %w-formatted) MUST still map correctly,
// else switch-by-identity falls through to InternalError, masking 4xx as 500.
// Callers MUST return immediately after a non-nil return.
func translateErr(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, blockstore.ErrWorkspaceNotFound),
		errors.Is(err, blockstore.ErrBlockNotFound),
		errors.Is(err, blockstore.ErrRefNotFound):
		apierr.AbortNotFound(c, apierr.RESOURCE_NOT_FOUND, err.Error())
	case errors.Is(err, blockstore.ErrOrgMismatch),
		errors.Is(err, blockstore.ErrBlockForbidden):
		apierr.AbortForbidden(c, apierr.INSUFFICIENT_PERMISSIONS, err.Error())
	case errors.Is(err, blockstore.ErrUnknownBlockType),
		errors.Is(err, blockstore.ErrUnknownOpKind),
		errors.Is(err, blockstore.ErrInvalidRel),
		errors.Is(err, blockstore.ErrOrderKeyRequired),
		errors.Is(err, blockstore.ErrMissingRequiredKey),
		errors.Is(err, blockstore.ErrColumnValueInvalid),
		errors.Is(err, blockstore.ErrChildNotAllowed),
		errors.Is(err, blockstore.ErrCrossWorkspaceRef),
		errors.Is(err, blockstore.ErrApplyOpsEmpty),
		errors.Is(err, blockstore.ErrEmbeddingDisabled):
		apierr.AbortBadRequest(c, apierr.VALIDATION_FAILED, err.Error())
	case errors.Is(err, blockstore.ErrSingleNestParent),
		errors.Is(err, blockstore.ErrNestCycle),
		errors.Is(err, blockstore.ErrStaleUpdate),
		errors.Is(err, blockstore.ErrWorkspaceAlreadyExists):
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{
			"error": apierr.VALIDATION_FAILED, "message": err.Error(),
		})
	default:
		slog.Warn("blockstore.internal_error", "err", err.Error())
		apierr.InternalError(c, "internal error")
	}
	return true
}
