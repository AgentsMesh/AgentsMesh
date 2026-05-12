package interceptors_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/anthropics/agentsmesh/backend/internal/api/connect/interceptors"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
)

const testSecret = "test-secret-do-not-use-in-prod"

type echoReq struct{ Msg string }
type echoRes struct{ Echo string }

func issueToken(t *testing.T, secret string, userID int64, exp time.Duration) string {
	t.Helper()
	tok, err := middleware.GenerateToken(userID, "u@example.com", "user", secret, 1)
	require.NoError(t, err)
	if exp == 0 {
		return tok
	}
	// Re-mint with custom expiration to support the "expired" case.
	claims := middleware.JWTClaims{
		UserID:   userID,
		Email:    "u@example.com",
		Username: "user",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(exp)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	require.NoError(t, err)
	return signed
}

// runInterceptor wraps a captureFunc with the auth interceptor and
// invokes it the way connect-go does for handler-side unary RPCs.
func runInterceptor(t *testing.T, header string, next connect.UnaryFunc) (connect.AnyResponse, error) {
	t.Helper()
	interceptor := interceptors.NewAuthInterceptor(testSecret)
	req := connect.NewRequest(&echoReq{Msg: "hi"})
	if header != "" {
		req.Header().Set("Authorization", header)
	}
	wrapped := interceptor(next)
	return wrapped(context.Background(), req)
}

func okHandler(t *testing.T, capture *context.Context) connect.UnaryFunc {
	t.Helper()
	return func(ctx context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
		*capture = ctx
		return connect.NewResponse(&echoRes{Echo: "ok"}), nil
	}
}

func TestAuthInterceptor_ValidToken_PopulatesContext(t *testing.T) {
	token := issueToken(t, testSecret, 42, 0)

	var captured context.Context
	resp, err := runInterceptor(t, "Bearer "+token, okHandler(t, &captured))

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, captured, "downstream handler must have been called")

	tenant := middleware.GetTenant(captured)
	require.NotNil(t, tenant, "TenantContext must be injected via middleware.SetTenant")
	assert.Equal(t, int64(42), tenant.UserID)

	claims := interceptors.ClaimsFromContext(captured)
	require.NotNil(t, claims)
	assert.Equal(t, int64(42), claims.UserID)
	assert.Equal(t, "u@example.com", claims.Email)
}

func TestAuthInterceptor_MissingHeader_ReturnsUnauthenticated(t *testing.T) {
	called := false
	resp, err := runInterceptor(t, "", func(context.Context, connect.AnyRequest) (connect.AnyResponse, error) {
		called = true
		return nil, nil
	})

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.False(t, called, "downstream handler must NOT be called on auth failure")

	var connectErr *connect.Error
	require.True(t, errors.As(err, &connectErr))
	assert.Equal(t, connect.CodeUnauthenticated, connectErr.Code())
}

func TestAuthInterceptor_InvalidTokens_ReturnUnauthenticated(t *testing.T) {
	cases := []struct {
		name   string
		header string
	}{
		{"malformed_token", "Bearer not-a-jwt"},
		{"wrong_signing_secret", "Bearer " + issueToken(t, "different-secret", 42, 0)},
		{"missing_bearer_scheme", issueToken(t, testSecret, 42, 0)},
		{"empty_bearer_value", "Bearer "},
		{"non_bearer_scheme", "Basic dXNlcjpwYXNz"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			called := false
			resp, err := runInterceptor(t, tc.header, func(context.Context, connect.AnyRequest) (connect.AnyResponse, error) {
				called = true
				return nil, nil
			})

			require.Error(t, err)
			assert.Nil(t, resp)
			assert.False(t, called)

			var connectErr *connect.Error
			require.True(t, errors.As(err, &connectErr))
			assert.Equal(t, connect.CodeUnauthenticated, connectErr.Code())
		})
	}
}

func TestAuthInterceptor_ExpiredToken_ReturnsUnauthenticated(t *testing.T) {
	expired := issueToken(t, testSecret, 42, -time.Hour)

	called := false
	resp, err := runInterceptor(t, "Bearer "+expired, func(context.Context, connect.AnyRequest) (connect.AnyResponse, error) {
		called = true
		return nil, nil
	})

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.False(t, called)

	var connectErr *connect.Error
	require.True(t, errors.As(err, &connectErr))
	assert.Equal(t, connect.CodeUnauthenticated, connectErr.Code())
}

func TestAuthInterceptor_PassesThroughHandlerErrors(t *testing.T) {
	token := issueToken(t, testSecret, 42, 0)
	downstreamErr := connect.NewError(connect.CodeNotFound, errors.New("resource missing"))

	resp, err := runInterceptor(t, "Bearer "+token, func(context.Context, connect.AnyRequest) (connect.AnyResponse, error) {
		return nil, downstreamErr
	})

	assert.Nil(t, resp)
	var connectErr *connect.Error
	require.True(t, errors.As(err, &connectErr))
	assert.Equal(t, connect.CodeNotFound, connectErr.Code(),
		"interceptor must not re-wrap downstream errors as Unauthenticated")
}
