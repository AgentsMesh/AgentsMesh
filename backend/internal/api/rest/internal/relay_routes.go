package internal

import (
	"crypto/subtle"
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/infra/acme"
	"github.com/anthropics/agentsmesh/backend/internal/service/geo"
	"github.com/anthropics/agentsmesh/backend/internal/service/relay"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// RelayHandler handles internal relay API endpoints
type RelayHandler struct {
	relayManager *relay.Manager
	dnsService   *relay.DNSService
	acmeManager  *acme.Manager
	geoResolver  geo.Resolver
	logger       *slog.Logger
}

// NewRelayHandler creates a new relay handler
func NewRelayHandler(relayManager *relay.Manager, dnsService *relay.DNSService, acmeManager *acme.Manager, geoResolver geo.Resolver) *RelayHandler {
	return &RelayHandler{
		relayManager: relayManager,
		dnsService:   dnsService,
		acmeManager:  acmeManager,
		geoResolver:  geoResolver,
		logger:       slog.With("component", "relay_handler"),
	}
}

// RelayRouterDeps holds dependencies for relay routes
type RelayRouterDeps struct {
	RelayManager   *relay.Manager
	DNSService     *relay.DNSService
	ACMEManager    *acme.Manager
	GeoResolver    geo.Resolver
	InternalSecret string
}

// RegisterRelayRoutes registers relay API routes
func RegisterRelayRoutes(router *gin.RouterGroup, deps *RelayRouterDeps) {
	handler := NewRelayHandler(deps.RelayManager, deps.DNSService, deps.ACMEManager, deps.GeoResolver)

	// Internal API authentication middleware
	router.Use(InternalAPIAuth(deps.InternalSecret))

	router.POST("/register", handler.Register)
	router.POST("/heartbeat", handler.Heartbeat)
	router.POST("/unregister", handler.Unregister)
	router.GET("/stats", handler.Stats)
	router.GET("", handler.List)
	router.GET("/:relay_id", handler.Get)
	router.DELETE("/:relay_id", handler.ForceUnregister)
}

// InternalAPIAuth is middleware for internal API authentication.
// Uses constant-time comparison to prevent timing attacks on the secret.
// Panics at setup time if secret is empty to prevent accidental auth bypass.
func InternalAPIAuth(secret string) gin.HandlerFunc {
	if secret == "" {
		panic("internal API secret must not be empty")
	}
	secretBytes := []byte(secret)
	return func(c *gin.Context) {
		auth := []byte(c.GetHeader("X-Internal-Secret"))
		if subtle.ConstantTimeCompare(auth, secretBytes) != 1 {
			apierr.AbortUnauthorized(c, apierr.AUTH_REQUIRED, "unauthorized")
			return
		}
		c.Next()
	}
}
