package authconnect

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/anthropics/agentsmesh/proto/gen/go/auth/v1"
)

func connectCodeOf(t *testing.T, err error) connect.Code {
	t.Helper()
	var ce *connect.Error
	require.True(t, errors.As(err, &ce), "expected *connect.Error, got %v", err)
	return ce.Code()
}

// --- input validation (handlers must reject empty inputs before touching
// the service layer, conventions §10 — InvalidArgument, not Internal) ---

func TestLogin_EmptyEmail_InvalidArgument(t *testing.T) {
	srv := &Server{}
	_, err := srv.Login(context.Background(),
		connect.NewRequest(&authv1.LoginRequest{Password: "p"}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connectCodeOf(t, err))
}

func TestLogin_EmptyPassword_InvalidArgument(t *testing.T) {
	srv := &Server{}
	_, err := srv.Login(context.Background(),
		connect.NewRequest(&authv1.LoginRequest{Email: "a@b.com"}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connectCodeOf(t, err))
}

func TestRegister_EmptyEmail_InvalidArgument(t *testing.T) {
	srv := &Server{}
	_, err := srv.Register(context.Background(),
		connect.NewRequest(&authv1.RegisterRequest{Username: "u", Password: "12345678"}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connectCodeOf(t, err))
}

func TestRegister_ShortPassword_InvalidArgument(t *testing.T) {
	srv := &Server{}
	_, err := srv.Register(context.Background(),
		connect.NewRequest(&authv1.RegisterRequest{
			Email: "a@b.com", Username: "user", Password: "short",
		}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connectCodeOf(t, err))
}

func TestRefreshToken_EmptyToken_InvalidArgument(t *testing.T) {
	srv := &Server{}
	_, err := srv.RefreshToken(context.Background(),
		connect.NewRequest(&authv1.RefreshTokenRequest{}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connectCodeOf(t, err))
}

func TestVerifyEmail_EmptyToken_InvalidArgument(t *testing.T) {
	srv := &Server{}
	_, err := srv.VerifyEmail(context.Background(),
		connect.NewRequest(&authv1.VerifyEmailRequest{}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connectCodeOf(t, err))
}

func TestResendVerification_EmptyEmail_InvalidArgument(t *testing.T) {
	srv := &Server{}
	_, err := srv.ResendVerification(context.Background(),
		connect.NewRequest(&authv1.ResendVerificationRequest{}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connectCodeOf(t, err))
}

func TestForgotPassword_EmptyEmail_InvalidArgument(t *testing.T) {
	srv := &Server{}
	_, err := srv.ForgotPassword(context.Background(),
		connect.NewRequest(&authv1.ForgotPasswordRequest{}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connectCodeOf(t, err))
}

func TestResetPassword_EmptyToken_InvalidArgument(t *testing.T) {
	srv := &Server{}
	_, err := srv.ResetPassword(context.Background(),
		connect.NewRequest(&authv1.ResetPasswordRequest{NewPassword: "12345678"}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connectCodeOf(t, err))
}

func TestResetPassword_ShortPassword_InvalidArgument(t *testing.T) {
	srv := &Server{}
	_, err := srv.ResetPassword(context.Background(),
		connect.NewRequest(&authv1.ResetPasswordRequest{
			Token: "abc", NewPassword: "short",
		}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connectCodeOf(t, err))
}

func TestOAuthRedirect_EmptyProvider_InvalidArgument(t *testing.T) {
	srv := &Server{}
	_, err := srv.OAuthRedirect(context.Background(),
		connect.NewRequest(&authv1.OAuthRedirectRequest{}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connectCodeOf(t, err))
}

func TestOAuthCallback_EmptyFields_InvalidArgument(t *testing.T) {
	srv := &Server{}
	cases := []struct {
		name string
		in   *authv1.OAuthCallbackRequest
	}{
		{"empty provider", &authv1.OAuthCallbackRequest{Code: "c", State: "s"}},
		{"empty code", &authv1.OAuthCallbackRequest{Provider: "github", State: "s"}},
		{"empty state", &authv1.OAuthCallbackRequest{Provider: "github", Code: "c"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := srv.OAuthCallback(context.Background(), connect.NewRequest(tc.in))
			require.Error(t, err)
			assert.Equal(t, connect.CodeInvalidArgument, connectCodeOf(t, err))
		})
	}
}

// --- Logout — token-blacklist path is no-op when service is nil ---

func TestLogout_NoBearer_Succeeds(t *testing.T) {
	// Without an Authorization header, Logout returns success without
	// calling RevokeToken. The interceptor would've already rejected the
	// request if auth was actually required — this exercises the handler's
	// own guard logic.
	srv := &SessionServer{}
	req := connect.NewRequest(&authv1.LogoutRequest{})
	resp, err := srv.Logout(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "Logged out successfully", resp.Msg.GetMessage())
}

func TestLogout_MalformedHeader_NoCrash(t *testing.T) {
	srv := &SessionServer{}
	req := connect.NewRequest(&authv1.LogoutRequest{})
	req.Header().Set("Authorization", "Token abc") // wrong scheme
	resp, err := srv.Logout(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "Logged out successfully", resp.Msg.GetMessage())
}

// --- Procedure URL constants align with proto package + service name ---

func TestProcedureConstants(t *testing.T) {
	assert.Equal(t, "/proto.auth.v1.AuthService/Login", LoginProcedure)
	assert.Equal(t, "/proto.auth.v1.AuthService/Register", RegisterProcedure)
	assert.Equal(t, "/proto.auth.v1.AuthService/RefreshToken", RefreshTokenProcedure)
	assert.Equal(t, "/proto.auth.v1.AuthService/ForgotPassword", ForgotPasswordProcedure)
	assert.Equal(t, "/proto.auth.v1.AuthService/ResetPassword", ResetPasswordProcedure)
	assert.Equal(t, "/proto.auth.v1.AuthService/VerifyEmail", VerifyEmailProcedure)
	assert.Equal(t, "/proto.auth.v1.AuthService/ResendVerification", ResendVerificationProcedure)
	assert.Equal(t, "/proto.auth.v1.AuthService/OAuthRedirect", OAuthRedirectProcedure)
	assert.Equal(t, "/proto.auth.v1.AuthService/OAuthCallback", OAuthCallbackProcedure)
	assert.Equal(t, "/proto.auth.v1.AuthSessionService/Logout", LogoutProcedure)
}

func TestMount_DoesNotPanic(t *testing.T) {
	// Smoke test: Mount with nil service. Construction should not panic;
	// real handlers reject at the first nil deref, but Mount itself only
	// wires the http.ServeMux.
	pubMux := http.NewServeMux()
	MountPublic(pubMux, &Server{})

	sessMux := http.NewServeMux()
	MountSession(sessMux, &SessionServer{})
}
