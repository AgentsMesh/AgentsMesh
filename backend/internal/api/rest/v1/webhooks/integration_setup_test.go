package webhooks

import (
	"log/slog"
	"os"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/testkit"
	billingService "github.com/anthropics/agentsmesh/backend/internal/service/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/payment"
	"gorm.io/gorm"
)

// ===========================================
// Integration Test Setup
// ===========================================

func setupIntegrationDB(t *testing.T) *gorm.DB {
	db := testkit.SetupTestDB(t)

	// Seed test data
	db.Exec(`INSERT INTO subscription_plans (name, display_name, tier, price_per_seat_monthly, price_per_seat_yearly, max_users, max_runners, max_repositories, max_concurrent_pods, included_pod_minutes, is_active)
		VALUES ('based', 'Based', 'based', 0, 0, 5, 1, 3, 2, 100, TRUE)`)
	db.Exec(`INSERT INTO subscription_plans (name, display_name, tier, price_per_seat_monthly, price_per_seat_yearly, max_users, max_runners, max_repositories, max_concurrent_pods, included_pod_minutes, is_active)
		VALUES ('pro', 'Pro', 'pro', 19.99, 199.90, 50, 10, 100, 10, 5000, TRUE)`)
	db.Exec(`INSERT INTO subscription_plans (name, display_name, tier, price_per_seat_monthly, price_per_seat_yearly, max_users, max_runners, max_repositories, max_concurrent_pods, included_pod_minutes, is_active)
		VALUES ('enterprise', 'Enterprise', 'enterprise', 99.99, 999.90, -1, -1, -1, -1, -1, TRUE)`)

	return db
}

func integrationTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func createMockRouter(t *testing.T) (*WebhookRouter, *gorm.DB, *payment.Factory) {
	db := setupIntegrationDB(t)
	logger := integrationTestLogger()
	registry := NewHandlerRegistry(logger)
	SetupDefaultHandlers(registry, logger)

	// Create full config with mock payment enabled
	cfg := &config.Config{
		PrimaryDomain: "localhost:3000",
		UseHTTPS:      false,
		Payment: config.PaymentConfig{
			DeploymentType: config.DeploymentGlobal,
			MockEnabled:    true,
			MockBaseURL:    "http://localhost:3000",
		},
	}

	// Create billing service with mock (uses full config for URL derivation)
	billingSvc := billingService.NewServiceWithConfig(infra.NewBillingRepository(db), cfg)
	factory := billingSvc.GetPaymentFactory()

	return &WebhookRouter{
		db:             db,
		cfg:            cfg,
		logger:         logger,
		registry:       registry,
		billingSvc:     billingSvc,
		paymentFactory: factory,
	}, db, factory
}
