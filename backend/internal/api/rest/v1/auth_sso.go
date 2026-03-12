package v1

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/domain/sso"
	userDomain "github.com/anthropics/agentsmesh/backend/internal/domain/user"
	"github.com/anthropics/agentsmesh/backend/internal/service/auth"
	ssoservice "github.com/anthropics/agentsmesh/backend/internal/service/sso"
	"github.com/anthropics/agentsmesh/backend/internal/service/user"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	ssoprovider "github.com/anthropics/agentsmesh/backend/pkg/auth/sso"
	"github.com/gin-gonic/gin"
)

// domainRegexp validates email domain format in URL path parameters.
var domainRegexp = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?)+$`)

// SSOAuthHandler handles SSO authentication requests
type SSOAuthHandler struct {
	ssoService  *ssoservice.Service
	authService *auth.Service
	userService *user.Service
	config      *config.Config
}

// NewSSOAuthHandler creates a new SSO auth handler
func NewSSOAuthHandler(ssoSvc *ssoservice.Service, authSvc *auth.Service, userSvc *user.Service, cfg *config.Config) *SSOAuthHandler {
	return &SSOAuthHandler{
		ssoService:  ssoSvc,
		authService: authSvc,
		userService: userSvc,
		config:      cfg,
	}
}

// RegisterRoutes registers SSO authentication routes
func (h *SSOAuthHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/discover", h.Discover)
	rg.GET("/:domain/oidc", h.OIDCRedirect)
	rg.GET("/:domain/oidc/callback", h.OIDCCallback)
	rg.GET("/:domain/saml", h.SAMLRedirect)
	rg.POST("/:domain/saml/acs", h.SAMLACS)
	rg.POST("/:domain/ldap", h.LDAPAuth)
	rg.GET("/:domain/saml/metadata", h.SAMLMetadata)
}

// Discover returns available SSO configurations for a given email domain
func (h *SSOAuthHandler) Discover(c *gin.Context) {
	email := c.Query("email")
	if email == "" {
		apierr.InvalidInput(c, "Email is required")
		return
	}

	domain := extractEmailDomain(email)
	if domain == "" {
		apierr.InvalidInput(c, "Invalid email format")
		return
	}

	configs, err := h.ssoService.GetEnabledConfigs(c.Request.Context(), domain)
	if err != nil {
		slog.Error("failed to discover SSO configs", "domain", domain, "error", err)
		c.JSON(http.StatusOK, gin.H{"configs": []interface{}{}})
		return
	}

	// Return sanitized list
	result := make([]*ssoservice.DiscoverResponse, 0, len(configs))
	for _, cfg := range configs {
		result = append(result, h.ssoService.ToDiscoverResponse(cfg))
	}

	c.JSON(http.StatusOK, gin.H{"configs": result})
}

// errAccountDisabled is returned by authenticateSSO when the user account is deactivated.
var errAccountDisabled = fmt.Errorf("account is disabled")

// authenticateSSO creates/gets the user from SSO identity, checks if active, and generates tokens.
// It does NOT write HTTP responses — callers decide how to handle errors (JSON vs redirect).
func (h *SSOAuthHandler) authenticateSSO(c *gin.Context, protocol sso.Protocol, configID int64, userInfo *ssoprovider.UserInfo) (*userDomain.User, *auth.TokenPair, error) {
	providerName := ssoservice.SSOProviderName(protocol, configID)
	u, _, err := h.userService.GetOrCreateByOAuth(c.Request.Context(), providerName, userInfo.ExternalID, userInfo.Username, userInfo.Email, userInfo.Name, userInfo.AvatarURL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create/get user: %w", err)
	}

	// Reject disabled users — admin may have deactivated them after SSO identity was created
	if !u.IsActive {
		return nil, nil, errAccountDisabled
	}

	tokens, err := h.authService.GenerateTokenPair(u, 0, "")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate tokens: %w", err)
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

// extractEmailDomain extracts the domain part from an email address
func extractEmailDomain(email string) string {
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 {
		return ""
	}
	return strings.ToLower(parts[1])
}
