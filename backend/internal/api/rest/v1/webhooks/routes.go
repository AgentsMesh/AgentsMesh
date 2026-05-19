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

type WebhookRouter struct {
	db             *gorm.DB
	cfg            *config.Config
	logger         *slog.Logger
	registry       *HandlerRegistry
	billingSvc     *billing.Service
	paymentFactory *payment.Factory

	repoService    *repository.Service
	webhookService *repository.WebhookService
	mrSyncService  *ticket.MRSyncService
	podService     *agentpod.PodService
	eventBus       *eventbus.EventBus
}

type WebhookRouterOption func(*WebhookRouter)

func WithRepositoryService(svc *repository.Service) WebhookRouterOption {
	return func(r *WebhookRouter) {
		r.repoService = svc
	}
}

func WithWebhookService(svc *repository.WebhookService) WebhookRouterOption {
	return func(r *WebhookRouter) {
		r.webhookService = svc
	}
}

func WithMRSyncService(svc *ticket.MRSyncService) WebhookRouterOption {
	return func(r *WebhookRouter) {
		r.mrSyncService = svc
	}
}

func WithPodService(svc *agentpod.PodService) WebhookRouterOption {
	return func(r *WebhookRouter) {
		r.podService = svc
	}
}

func WithEventBus(eb *eventbus.EventBus) WebhookRouterOption {
	return func(r *WebhookRouter) {
		r.eventBus = eb
	}
}

func NewWebhookRouter(db *gorm.DB, cfg *config.Config, logger *slog.Logger, opts ...WebhookRouterOption) *WebhookRouter {
	registry := NewHandlerRegistry(logger)
	SetupDefaultHandlers(registry, logger)

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

func (r *WebhookRouter) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/:org_slug/gitlab/:repo_id", r.handleGitLabWebhookWithRepo)
	rg.POST("/:org_slug/github/:repo_id", r.handleGitHubWebhookWithRepo)
	rg.POST("/:org_slug/gitee/:repo_id", r.handleGiteeWebhookWithRepo)

	rg.POST("/stripe", r.handleStripeWebhook)
	rg.POST("/lemonsqueezy", r.handleLemonSqueezyWebhook)
	rg.POST("/alipay", r.handleAlipayWebhook)
	rg.POST("/wechat", r.handleWeChatWebhook)

	rg.POST("/mock/complete", r.handleMockCheckoutComplete)
	rg.GET("/mock/session/:session_id", r.getMockSession)
}
