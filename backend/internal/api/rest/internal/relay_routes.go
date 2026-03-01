package internal

import (
	"log/slog"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/infra/acme"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/relay"
	"github.com/AgentsMesh/AgentsMesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// RelayHandler handles internal relay API endpoints
type RelayHandler struct {
	relayManager *relay.Manager
	dnsService   *relay.DNSService
	acmeManager  *acme.Manager
	logger       *slog.Logger
}

// NewRelayHandler creates a new relay handler
func NewRelayHandler(relayManager *relay.Manager, dnsService *relay.DNSService, acmeManager *acme.Manager) *RelayHandler {
	return &RelayHandler{
		relayManager: relayManager,
		dnsService:   dnsService,
		acmeManager:  acmeManager,
		logger:       slog.With("component", "relay_handler"),
	}
}

// RelayRouterDeps holds dependencies for relay routes
type RelayRouterDeps struct {
	RelayManager   *relay.Manager
	DNSService     *relay.DNSService
	ACMEManager    *acme.Manager
	InternalSecret string
}

// RegisterRelayRoutes registers relay API routes
func RegisterRelayRoutes(router *gin.RouterGroup, deps *RelayRouterDeps) {
	handler := NewRelayHandler(deps.RelayManager, deps.DNSService, deps.ACMEManager)

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

// InternalAPIAuth is middleware for internal API authentication
func InternalAPIAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("X-Internal-Secret")
		if auth != secret {
			apierr.AbortUnauthorized(c, apierr.AUTH_REQUIRED, "unauthorized")
			return
		}
		c.Next()
	}
}
