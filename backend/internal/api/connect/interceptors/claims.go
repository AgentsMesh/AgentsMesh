package interceptors

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
)

type claimsCtxKey struct{}

func withClaims(ctx context.Context, claims *middleware.JWTClaims) context.Context {
	return context.WithValue(ctx, claimsCtxKey{}, claims)
}

// ClaimsFromContext returns the JWT claims attached by the auth
// interceptor, or nil if the request was unauthenticated. Service
// handlers prefer `middleware.GetTenant(ctx)` for the canonical UserID;
// use this only when full claim fields (Email / Username / Exp / ...)
// are needed.
func ClaimsFromContext(ctx context.Context) *middleware.JWTClaims {
	c, _ := ctx.Value(claimsCtxKey{}).(*middleware.JWTClaims)
	return c
}
