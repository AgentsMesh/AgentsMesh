// Package ssoconnect hosts Connect-RPC handlers for the public SSO
// discovery + LDAP auth surface. Mirrors REST handlers in
//   backend/internal/api/rest/v1/auth_sso.go        (Discover)
//   backend/internal/api/rest/v1/auth_sso_ldap.go   (LDAPAuth)
// but exposes the data plane via Connect (binary protobuf wire,
// conventions §2.5). REST stays mounted in parallel; the migration runs
// dual-track until all 26 services have flipped.
//
// The OIDC/SAML browser-redirect endpoints (auth_sso_oidc.go,
// auth_sso_saml.go) stay on REST permanently — Connect's unary contract
// cannot return `Location:` redirects. The admin SSO config CRUD
// endpoints (admin/sso.go) are out of scope here; web-admin owns them
// and they are not consumed by the user-facing `getSSOService()` wasm
// bridge.
//
// SENSITIVE: SSOService is user-scoped + PUBLIC per conventions §3.5
// exception #1 — the auth interceptor must NOT be applied. Users who
// hit Discover do not have a bearer token (that is the goal of the
// SSO flow they are starting); LdapAuth issues the bearer token after
// authenticating via the IdP. MountPublic enforces that constraint;
// orchestration in cmd/server/connect_init.go calls MountPublic
// without the auth interceptor.
package ssoconnect

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"connectrpc.com/connect"

	"github.com/anthropics/agentsmesh/backend/internal/domain/sso"
	authservice "github.com/anthropics/agentsmesh/backend/internal/service/auth"
	ssoservice "github.com/anthropics/agentsmesh/backend/internal/service/sso"
	ssov1 "github.com/anthropics/agentsmesh/proto/gen/go/sso/v1"
)

const ServiceName = "proto.sso.v1.SSOService"

const (
	DiscoverProcedure = "/" + ServiceName + "/Discover"
	LdapAuthProcedure = "/" + ServiceName + "/LdapAuth"
)

// Server hosts the public SSOService. Mirrors REST's SSOAuthHandler
// (auth_sso.go:24) — same two service deps the user-facing RPCs need.
// The OIDC/SAML/metadata REST endpoints take a third dep (config.Config)
// for FrontendURL redirects; those endpoints stay on REST and are
// untouched here.
type Server struct {
	ssoSvc  *ssoservice.Service
	authSvc *authservice.Service
}

func NewServer(ssoSvc *ssoservice.Service, authSvc *authservice.Service) *Server {
	return &Server{ssoSvc: ssoSvc, authSvc: authSvc}
}

// Discover returns the SSO configs registered for the email's domain.
// Mirrors REST's GET /api/v1/auth/sso/discover?email=… (auth_sso.go:52).
// On an unknown domain or repo error the REST handler returns an empty
// list with 200 — we keep the same contract: the frontend's login page
// renders the password form when no configs come back, never a banner.
func (s *Server) Discover(
	ctx context.Context, req *connect.Request[ssov1.DiscoverRequest],
) (*connect.Response[ssov1.DiscoverResponse], error) {
	email := req.Msg.GetEmail()
	if email == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("email is required"))
	}
	domain := extractEmailDomain(email)
	if domain == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("invalid email format"))
	}

	configs, err := s.ssoSvc.GetEnabledConfigs(ctx, domain)
	if err != nil {
		// Mirror REST behavior: log + return empty list so the login
		// page can render the password form. Errors here are typically
		// transient DB issues and should not block the password flow.
		slog.ErrorContext(ctx, "failed to discover SSO configs",
			"domain", domain, "error", err)
		return connect.NewResponse(&ssov1.DiscoverResponse{
			Items: []*ssov1.SSODiscoverConfig{}, Total: 0,
		}), nil
	}

	items := make([]*ssov1.SSODiscoverConfig, 0, len(configs))
	for _, cfg := range configs {
		items = append(items, toProtoDiscoverConfig(cfg))
	}
	return connect.NewResponse(&ssov1.DiscoverResponse{
		Items:  items,
		Total:  int64(len(items)),
		Limit:  0,
		Offset: 0,
	}), nil
}

// LdapAuth authenticates the caller against the LDAP IdP registered for
// the email's domain and returns session tokens + user info. Mirrors
// REST's POST /api/v1/auth/sso/:domain/ldap (auth_sso_ldap.go:21).
func (s *Server) LdapAuth(
	ctx context.Context, req *connect.Request[ssov1.LdapAuthRequest],
) (*connect.Response[ssov1.LdapAuthResponse], error) {
	domain, err := validateDomainFromRequest(req.Msg.GetDomain())
	if err != nil {
		return nil, err
	}
	username := req.Msg.GetUsername()
	password := req.Msg.GetPassword()
	if username == "" || password == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("username and password are required"))
	}

	userInfo, configID, err := s.ssoSvc.AuthenticateLDAP(ctx, domain, username, password)
	if err != nil {
		if errors.Is(err, ssoservice.ErrConfigNotFound) {
			return nil, connect.NewError(connect.CodeNotFound,
				errors.New("SSO config not found"))
		}
		return nil, connect.NewError(connect.CodeUnauthenticated,
			errors.New("LDAP authentication failed"))
	}

	providerName := ssoservice.SSOProviderName(sso.ProtocolLDAP, configID)
	u, tokens, err := s.authSvc.SSOLogin(ctx, &authservice.SSOLoginRequest{
		ProviderName: providerName,
		ExternalID:   userInfo.ExternalID,
		Username:     userInfo.Username,
		Email:        userInfo.Email,
		Name:         userInfo.Name,
		AvatarURL:    userInfo.AvatarURL,
	})
	if err != nil {
		if errors.Is(err, authservice.ErrUserDisabled) {
			return nil, connect.NewError(connect.CodePermissionDenied,
				errors.New("account is disabled"))
		}
		return nil, connect.NewError(connect.CodeInternal,
			errors.New("failed to process authentication"))
	}

	return connect.NewResponse(&ssov1.LdapAuthResponse{
		Token:        tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresAt:    tokens.ExpiresAt.Format("2006-01-02T15:04:05Z"),
		TokenType:    "Bearer",
		User:         toProtoLdapAuthUser(u),
	}), nil
}

// extractEmailDomain mirrors auth_sso.go:189 — split on `@`, lowercase.
// Local copy so the Connect package does not depend on the REST package.
func extractEmailDomain(email string) string {
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 {
		return ""
	}
	return strings.ToLower(parts[1])
}

// MountPublic registers SSOService procedures WITHOUT the auth
// interceptor. The caller (cmd/server/connect_init.go) passes only
// interceptors that do not require authentication (today: none).
func MountPublic(mux *http.ServeMux, srv *Server, opts ...connect.HandlerOption) {
	mux.Handle(DiscoverProcedure, connect.NewUnaryHandler(DiscoverProcedure, srv.Discover, opts...))
	mux.Handle(LdapAuthProcedure, connect.NewUnaryHandler(LdapAuthProcedure, srv.LdapAuth, opts...))
}
