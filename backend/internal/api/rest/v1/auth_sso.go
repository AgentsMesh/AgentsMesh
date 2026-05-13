package v1

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/domain/sso"
	userDomain "github.com/anthropics/agentsmesh/backend/internal/domain/user"
	"github.com/anthropics/agentsmesh/backend/internal/service/auth"
	ssoservice "github.com/anthropics/agentsmesh/backend/internal/service/sso"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	ssoprovider "github.com/anthropics/agentsmesh/backend/pkg/auth/sso"
	"github.com/gin-gonic/gin"
)

// domainRegexp validates email domain format in URL path parameters.
var domainRegexp = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?)+$`)

// SSOAuthHandler hosts the OIDC/SAML browser-redirect surface that
// Connect-RPC cannot replace — IdP-initiated GETs issue a 307 to the
// IdP, the IdP callback path consumes their redirect, and the SAML
// metadata route serves XML. Connect's unary contract can't return
// `Location:` headers or `application/xml`, so these endpoints stay
// on REST permanently.
//
// Discover and LDAPAuth migrated to proto.sso.v1.SSOService —
// see backend/internal/api/connect/sso.
type SSOAuthHandler struct {
	ssoService  *ssoservice.Service
	authService *auth.Service
	config      *config.Config
}

func NewSSOAuthHandler(ssoSvc *ssoservice.Service, authSvc *auth.Service, cfg *config.Config) *SSOAuthHandler {
	return &SSOAuthHandler{
		ssoService:  ssoSvc,
		authService: authSvc,
		config:      cfg,
	}
}

// RegisterRoutes mounts the OIDC/SAML browser-redirect endpoints.
// Discover and LDAPAuth are owned by Connect-RPC.
func (h *SSOAuthHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/:domain/oidc", h.OIDCRedirect)
	rg.GET("/:domain/oidc/callback", h.OIDCCallback)
	rg.GET("/:domain/saml", h.SAMLRedirect)
	rg.POST("/:domain/saml/acs", h.SAMLACS)
	rg.GET("/:domain/saml/metadata", h.SAMLMetadata)
}

// authenticateSSO creates/gets the user from SSO identity, checks if active, and generates tokens.
// It does NOT write HTTP responses — callers decide how to handle errors (JSON vs redirect).
func (h *SSOAuthHandler) authenticateSSO(c *gin.Context, protocol sso.Protocol, configID int64, userInfo *ssoprovider.UserInfo) (*userDomain.User, *auth.TokenPair, error) {
	providerName := ssoservice.SSOProviderName(protocol, configID)
	u, tokens, err := h.authService.SSOLogin(c.Request.Context(), &auth.SSOLoginRequest{
		ProviderName: providerName,
		ExternalID:   userInfo.ExternalID,
		Username:     userInfo.Username,
		Email:        userInfo.Email,
		Name:         userInfo.Name,
		AvatarURL:    userInfo.AvatarURL,
	})
	if err != nil {
		return nil, nil, err
	}

	return u, tokens, nil
}

// redirectWithError redirects to the frontend with an error code as a query parameter.
// Used for browser-based flows (OIDC/SAML) where returning JSON would not be useful.
func (h *SSOAuthHandler) redirectWithError(c *gin.Context, redirectTo, errorCode string) {
	if !h.isAllowedRedirect(redirectTo) {
		redirectTo = h.config.FrontendURL() + "/auth/sso/callback"
	}

	redirectURL, err := url.Parse(redirectTo)
	if err != nil {
		redirectURL, _ = url.Parse(h.config.FrontendURL() + "/auth/sso/callback")
	}

	q := redirectURL.Query()
	q.Set("error", errorCode)
	redirectURL.RawQuery = q.Encode()

	c.Redirect(http.StatusTemporaryRedirect, redirectURL.String())
}

// redirectWithTokens redirects to the frontend with tokens as query parameters.
// Validates the redirect URL as a defense-in-depth measure.
func (h *SSOAuthHandler) redirectWithTokens(c *gin.Context, redirectTo string, tokens *auth.TokenPair) {
	// Defense-in-depth: re-validate redirect URL before sending tokens
	if !h.isAllowedRedirect(redirectTo) {
		redirectTo = h.config.FrontendURL() + "/auth/sso/callback"
	}

	redirectURL, err := url.Parse(redirectTo)
	if err != nil {
		redirectURL, _ = url.Parse(h.config.FrontendURL() + "/auth/sso/callback")
	}

	q := redirectURL.Query()
	q.Set("token", tokens.AccessToken)
	q.Set("refresh_token", tokens.RefreshToken)
	redirectURL.RawQuery = q.Encode()

	c.Redirect(http.StatusTemporaryRedirect, redirectURL.String())
}

// isAllowedRedirect validates that a redirect URL is safe.
// It checks both hostname and port against FrontendURL to prevent
// open-redirect attacks to attacker-controlled ports on the same host.
func (h *SSOAuthHandler) isAllowedRedirect(redirectTo string) bool {
	if strings.HasPrefix(redirectTo, "/") && !strings.HasPrefix(redirectTo, "//") {
		return true
	}
	parsed, err := url.Parse(redirectTo)
	if err != nil {
		return false
	}
	// Only allow http/https schemes to prevent javascript: and other dangerous protocols
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	allowed, err := url.Parse(h.config.FrontendURL())
	if err != nil {
		return false
	}
	return parsed.Hostname() == allowed.Hostname() &&
		normalizePort(parsed) == normalizePort(allowed)
}

// normalizePort returns the explicit port or the default port for the scheme,
// ensuring that "https://example.com" and "https://example.com:443" are treated equally.
func normalizePort(u *url.URL) string {
	if p := u.Port(); p != "" {
		return p
	}
	if u.Scheme == "https" {
		return "443"
	}
	return "80"
}

// validateDomain checks the domain path parameter format and returns it lowercased, or writes an error response.
func validateDomain(c *gin.Context) (string, bool) {
	domain := strings.ToLower(strings.TrimSpace(c.Param("domain")))
	if domain == "" {
		apierr.InvalidInput(c, "Domain is required")
		return "", false
	}
	if !domainRegexp.MatchString(domain) {
		apierr.InvalidInput(c, "Invalid domain format")
		return "", false
	}
	return domain, true
}
