// Package interceptors holds Connect-RPC interceptors shared across the
// per-service handlers in `backend/internal/api/connect/...`. The auth
// interceptor adapts the existing REST JWT middleware
// (`backend/internal/middleware/auth.go`) into the Connect handler
// pipeline so business code reads the same `TenantContext` regardless of
// transport.
package interceptors

import (
	"context"
	"errors"
	"strings"

	"connectrpc.com/connect"
	"github.com/golang-jwt/jwt/v5"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
)

// NewAuthInterceptor returns a Connect unary interceptor that validates
// the JWT in the `Authorization: Bearer <token>` request header and
// injects a `*middleware.TenantContext` (populated with UserID) into the
// downstream context using `middleware.SetTenant`.
//
// The interceptor only fills `TenantContext.UserID`; per-RPC tenant
// resolution (org slug → org ID + membership check) is the service
// handler's job because the RPC procedure path carries no org slug.
//
// Tokens are parsed with the same HS256 logic as
// `middleware.AuthMiddleware`; behavioural drift between the two paths
// would be a security bug.
func NewAuthInterceptor(jwtSecret string) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if req.Spec().IsClient {
				return next(ctx, req)
			}

			claims, err := parseBearerToken(req.Header().Get("Authorization"), jwtSecret)
			if err != nil {
				return nil, err
			}

			ctx = middleware.SetTenant(ctx, &middleware.TenantContext{UserID: claims.UserID})
			ctx = withClaims(ctx, claims)
			return next(ctx, req)
		}
	}
}

func parseBearerToken(header, secret string) (*middleware.JWTClaims, error) {
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" || parts[1] == "" {
		return nil, connect.NewError(
			connect.CodeUnauthenticated,
			errors.New("authorization bearer token is required"),
		)
	}

	claims := &middleware.JWTClaims{}
	token, err := jwt.ParseWithClaims(parts[1], claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return nil, connect.NewError(
			connect.CodeUnauthenticated,
			errors.New("invalid or expired token"),
		)
	}
	return claims, nil
}
