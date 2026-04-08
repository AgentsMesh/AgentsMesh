package promocode_test

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/promocode"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	svc "github.com/anthropics/agentsmesh/backend/internal/service/promocode"
	"github.com/anthropics/agentsmesh/backend/internal/testkit"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := testkit.SetupTestDB(t)

	// Insert test data with proper BLOB for features
	db.Exec(`INSERT INTO users (id, email, username, name) VALUES (1, 'test@example.com', 'testuser', 'Test User')`)
	db.Exec(`INSERT INTO organizations (id, name, slug) VALUES (1, 'Test Org', 'test-org')`)
	db.Exec(`INSERT INTO subscription_plans (id, name, display_name, max_users, max_runners, max_repositories, features) VALUES (1, 'based', 'Based', 1, 1, 3, X'7B7D')`)        // {} as hex
	db.Exec(`INSERT INTO subscription_plans (id, name, display_name, max_users, max_runners, max_repositories, price_per_seat_monthly, features) VALUES (2, 'pro', 'Pro', 5, 10, 10, 20, X'7B7D')`)
	db.Exec(`INSERT INTO subscription_plans (id, name, display_name, max_users, max_runners, max_repositories, price_per_seat_monthly, features) VALUES (3, 'enterprise', 'Enterprise', 50, 100, -1, 40, X'7B7D')`)

	return db
}

// newTestService creates a Service with real infra implementations for integration testing
func newTestService(db *gorm.DB) *svc.Service {
	return svc.NewService(infra.NewPromocodeRepository(db), infra.NewGormBillingProvider(db))
}

// Helper function
func intPtr(i int) *int {
	return &i
}

func TestService_Create(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	tests := []struct {
		name    string
		req     *svc.CreateRequest
		wantErr bool
	}{
		{
			name: "create valid promo code",
			req: &svc.CreateRequest{
				Code:           "TEST2024",
				Name:           "Test Promo",
				Description:    "Test promo code",
				Type:           promocode.PromoTypeMedia,
				PlanName:       "pro",
				DurationMonths: 3,
				CreatedByID:    1,
			},
			wantErr: false,
		},
		{
			name: "create with max uses",
			req: &svc.CreateRequest{
				Code:           "LIMITED100",
				Name:           "Limited Promo",
				Type:           promocode.PromoTypeCampaign,
				PlanName:       "pro",
				DurationMonths: 1,
				MaxUses:        intPtr(100),
				CreatedByID:    1,
			},
			wantErr: false,
		},
		{
			name: "create with invalid plan",
			req: &svc.CreateRequest{
				Code:           "INVALID",
				Name:           "Invalid",
				Type:           promocode.PromoTypeInternal,
				PlanName:       "nonexistent",
				DurationMonths: 1,
				CreatedByID:    1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := service.Create(ctx, tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got == nil {
					t.Error("Create() returned nil")
					return
				}
				if got.Code != tt.req.Code {
					t.Errorf("Create() code = %v, want %v", got.Code, tt.req.Code)
				}
				if got.PlanName != tt.req.PlanName {
					t.Errorf("Create() plan_name = %v, want %v", got.PlanName, tt.req.PlanName)
				}
			}
		})
	}
}
