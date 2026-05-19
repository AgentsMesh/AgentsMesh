package v1

import (
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
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	ssoprovider "github.com/anthropics/agentsmesh/backend/pkg/auth/sso"
	"github.com/gin-gonic/gin"
)

var domainRegexp = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?)+$`)

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

func (h *SSOAuthHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/discover", h.Discover)
	rg.GET("/:domain/oidc", h.OIDCRedirect)
	rg.GET("/:domain/oidc/callback", h.OIDCCallback)
	rg.GET("/:domain/saml", h.SAMLRedirect)
	rg.POST("/:domain/saml/acs", h.SAMLACS)
	rg.POST("/:domain/ldap", h.LDAPAuth)
	rg.GET("/:domain/saml/metadata", h.SAMLMetadata)
}

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
		slog.ErrorContext(c.Request.Context(), "failed to discover SSO configs", "domain", domain, "error", err)
		c.JSON(http.StatusOK, gin.H{"configs": []interface{}{}})
		return
	}

	result := make([]*ssoservice.DiscoverResponse, 0, len(configs))
	for _, cfg := range configs {
		result = append(result, h.ssoService.ToDiscoverResponse(cfg))
	}

	c.JSON(http.StatusOK, gin.H{"configs": result})
}

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

func (h *SSOAuthHandler) redirectWithTokens(c *gin.Context, redirectTo string, tokens *auth.TokenPair) {
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

func (h *SSOAuthHandler) isAllowedRedirect(redirectTo string) bool {
	if strings.HasPrefix(redirectTo, "/") && !strings.HasPrefix(redirectTo, "//") {
		return true
	}
	parsed, err := url.Parse(redirectTo)
	if err != nil {
		return false
	}
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

func normalizePort(u *url.URL) string {
	if p := u.Port(); p != "" {
		return p
	}
	if u.Scheme == "https" {
		return "443"
	}
	return "80"
}

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

func extractEmailDomain(email string) string {
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 {
		return ""
	}
	return strings.ToLower(parts[1])
}
