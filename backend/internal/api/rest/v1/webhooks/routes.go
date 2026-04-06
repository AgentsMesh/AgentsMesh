package webhooks

import (
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/infra/eventbus"
	"github.com/anthropics/agentsmesh/backend/internal/service/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/service/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/payment"
	"github.com/anthropics/agentsmesh/backend/internal/service/repository"
	"github.com/anthropics/agentsmesh/backend/internal/service/ticket"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// WebhookRouter handles webhook endpoint routing
type WebhookRouter struct {
	db             *gorm.DB
	cfg            *config.Config
	logger         *slog.Logger
	registry       *HandlerRegistry
	billingSvc     *billing.Service
	paymentFactory *payment.Factory

	// Services for git webhook handling
	repoService    *repository.Service
	webhookService *repository.WebhookService
	mrSyncService  *ticket.MRSyncService
	podService     *agentpod.PodService
	eventBus       *eventbus.EventBus
}

// WebhookRouterOption configures the WebhookRouter
type WebhookRouterOption func(*WebhookRouter)

// WithRepositoryService sets the repository service
func WithRepositoryService(svc *repository.Service) WebhookRouterOption {
	return func(r *WebhookRouter) {
		r.repoService = svc
	}
}

// WithWebhookService sets the webhook service
func WithWebhookService(svc *repository.WebhookService) WebhookRouterOption {
	return func(r *WebhookRouter) {
		r.webhookService = svc
	}
}

// WithMRSyncService sets the MR sync service
func WithMRSyncService(svc *ticket.MRSyncService) WebhookRouterOption {
	return func(r *WebhookRouter) {
		r.mrSyncService = svc
	}
}

// WithPodService sets the pod service
func WithPodService(svc *agentpod.PodService) WebhookRouterOption {
	return func(r *WebhookRouter) {
		r.podService = svc
	}
}

// WithEventBus sets the event bus
func WithEventBus(eb *eventbus.EventBus) WebhookRouterOption {
	return func(r *WebhookRouter) {
		r.eventBus = eb
	}
}

// NewWebhookRouter creates a new webhook router
func NewWebhookRouter(db *gorm.DB, cfg *config.Config, logger *slog.Logger, opts ...WebhookRouterOption) *WebhookRouter {
	registry := NewHandlerRegistry(logger)
	SetupDefaultHandlers(registry, logger)

	// Initialize billing service and payment factory for payment webhooks
	// Full config is passed for URL derivation (AlipayNotifyURL, etc.)
	billingRepo := infra.NewBillingRepository(db)
	billingSvc := billing.NewServiceWithConfig(billingRepo, cfg)
	paymentFactory := billingSvc.GetPaymentFactory()

	r := &WebhookRouter{
		db:             db,
		cfg:            cfg,
		logger:         logger,
		registry:       registry,
		billingSvc:     billingSvc,
		paymentFactory: paymentFactory,
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// NewWebhookRouterWithBillingSvc creates a new webhook router with an external billing service
// This allows sharing the same payment factory instance across the application
func NewWebhookRouterWithBillingSvc(db *gorm.DB, cfg *config.Config, logger *slog.Logger, billingSvc *billing.Service, opts ...WebhookRouterOption) *WebhookRouter {
	registry := NewHandlerRegistry(logger)
	SetupDefaultHandlers(registry, logger)

	r := &WebhookRouter{
		db:             db,
		cfg:            cfg,
		logger:         logger,
		registry:       registry,
		billingSvc:     billingSvc,
		paymentFactory: billingSvc.GetPaymentFactory(),
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// RegisterRoutes registers webhook routes to the router group
func (r *WebhookRouter) RegisterRoutes(rg *gin.RouterGroup) {
	// Git webhook endpoints with org_slug and repo_id
	// New format: POST /webhooks/:org_slug/:provider/:repo_id
	rg.POST("/:org_slug/gitlab/:repo_id", r.handleGitLabWebhookWithRepo)
	rg.POST("/:org_slug/github/:repo_id", r.handleGitHubWebhookWithRepo)
	rg.POST("/:org_slug/gitee/:repo_id", r.handleGiteeWebhookWithRepo)

	// Payment webhook endpoints
	rg.POST("/stripe", r.handleStripeWebhook)
	rg.POST("/lemonsqueezy", r.handleLemonSqueezyWebhook)
	rg.POST("/alipay", r.handleAlipayWebhook)
	rg.POST("/wechat", r.handleWeChatWebhook)

	// Mock payment endpoints (for testing)
	rg.POST("/mock/complete", r.handleMockCheckoutComplete)
	rg.GET("/mock/session/:session_id", r.getMockSession)
}
