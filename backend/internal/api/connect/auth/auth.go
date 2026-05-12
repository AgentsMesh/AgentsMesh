// Package authconnect hosts Connect-RPC handlers for the auth domain —
// public (login/register/refresh/oauth/verify/password-reset) entry points
// plus authenticated session control (logout). Mirrors REST handlers in
// backend/internal/api/rest/v1/auth*.go but exposes the data plane via
// Connect (binary protobuf wire, conventions §2.5). REST stays mounted in
// parallel; the migration runs dual-track until all 26 services have
// flipped.
//
// SENSITIVE: AuthService is user-scoped + PUBLIC per conventions §3.5
// exception #1 — the auth interceptor must NOT be applied to these RPCs.
// A user cannot present a bearer token before they have authenticated.
// `MountPublic` enforces that constraint; orchestration in
// cmd/server/connect_init.go calls `Mount` for the session service (which
// IS authenticated) and `MountPublic` for the public service.
package authconnect

import (
	"net/http"

	"connectrpc.com/connect"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/infra/email"
	authservice "github.com/anthropics/agentsmesh/backend/internal/service/auth"
	userservice "github.com/anthropics/agentsmesh/backend/internal/service/user"
)

const (
	ServiceName        = "proto.auth.v1.AuthService"
	SessionServiceName = "proto.auth.v1.AuthSessionService"
)

const (
	LoginProcedure              = "/" + ServiceName + "/Login"
	RegisterProcedure           = "/" + ServiceName + "/Register"
	RefreshTokenProcedure       = "/" + ServiceName + "/RefreshToken"
	ForgotPasswordProcedure     = "/" + ServiceName + "/ForgotPassword"
	ResetPasswordProcedure      = "/" + ServiceName + "/ResetPassword"
	VerifyEmailProcedure        = "/" + ServiceName + "/VerifyEmail"
	ResendVerificationProcedure = "/" + ServiceName + "/ResendVerification"
	OAuthRedirectProcedure      = "/" + ServiceName + "/OAuthRedirect"
	OAuthCallbackProcedure      = "/" + ServiceName + "/OAuthCallback"
)

const (
	LogoutProcedure = "/" + SessionServiceName + "/Logout"
)

// Server hosts the public AuthService. Mirrors REST's AuthHandler
// (auth.go:13) — same four service deps.
type Server struct {
	authSvc  *authservice.Service
	userSvc  *userservice.Service
	emailSvc email.Service
	config   *config.Config
}

func NewServer(
	authSvc *authservice.Service,
	userSvc *userservice.Service,
	emailSvc email.Service,
	cfg *config.Config,
) *Server {
	return &Server{
		authSvc:  authSvc,
		userSvc:  userSvc,
		emailSvc: emailSvc,
		config:   cfg,
	}
}

// SessionServer hosts AuthSessionService.Logout. Lives in its own struct
// so the mount path can apply the auth interceptor only to it without
// dragging the public RPCs through the bearer check.
type SessionServer struct {
	authSvc *authservice.Service
}

func NewSessionServer(authSvc *authservice.Service) *SessionServer {
	return &SessionServer{authSvc: authSvc}
}

// MountPublic registers AuthService procedures WITHOUT the auth interceptor.
// The caller (cmd/server/connect_init.go) passes only interceptors that
// don't require authentication (today: none).
func MountPublic(mux *http.ServeMux, srv *Server, opts ...connect.HandlerOption) {
	mux.Handle(LoginProcedure, connect.NewUnaryHandler(LoginProcedure, srv.Login, opts...))
	mux.Handle(RegisterProcedure, connect.NewUnaryHandler(RegisterProcedure, srv.Register, opts...))
	mux.Handle(RefreshTokenProcedure, connect.NewUnaryHandler(RefreshTokenProcedure, srv.RefreshToken, opts...))
	mux.Handle(ForgotPasswordProcedure, connect.NewUnaryHandler(ForgotPasswordProcedure, srv.ForgotPassword, opts...))
	mux.Handle(ResetPasswordProcedure, connect.NewUnaryHandler(ResetPasswordProcedure, srv.ResetPassword, opts...))
	mux.Handle(VerifyEmailProcedure, connect.NewUnaryHandler(VerifyEmailProcedure, srv.VerifyEmail, opts...))
	mux.Handle(ResendVerificationProcedure, connect.NewUnaryHandler(ResendVerificationProcedure, srv.ResendVerification, opts...))
	mux.Handle(OAuthRedirectProcedure, connect.NewUnaryHandler(OAuthRedirectProcedure, srv.OAuthRedirect, opts...))
	mux.Handle(OAuthCallbackProcedure, connect.NewUnaryHandler(OAuthCallbackProcedure, srv.OAuthCallback, opts...))
}

// MountSession registers AuthSessionService.Logout WITH the auth
// interceptor — Logout revokes the caller's bearer token, so the token
// must be present.
func MountSession(mux *http.ServeMux, srv *SessionServer, opts ...connect.HandlerOption) {
	mux.Handle(LogoutProcedure, connect.NewUnaryHandler(LogoutProcedure, srv.Logout, opts...))
}
