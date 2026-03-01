package lemonsqueezy

import (
	"context"
	"fmt"
	"strconv"
	"time"

	lemonsqueezy "github.com/NdoleStudio/lemonsqueezy-go"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/payment/types"
)

// CreateCheckoutSession creates a LemonSqueezy Checkout session
func (p *Provider) CreateCheckoutSession(ctx context.Context, req *types.CheckoutRequest) (*types.CheckoutResponse, error) {
	// LemonSqueezy requires a variant ID - this should be passed via metadata
	variantID := ""
	if req.Metadata != nil {
		variantID = req.Metadata["variant_id"]
	}
	if variantID == "" {
		return nil, fmt.Errorf("variant_id is required in metadata for LemonSqueezy checkout")
	}

	// Build custom data for checkout
	customData := map[string]any{
		"organization_id": strconv.FormatInt(req.OrganizationID, 10),
		"user_id":         strconv.FormatInt(req.UserID, 10),
		"order_type":      req.OrderType,
		"billing_cycle":   req.BillingCycle,
		"seats":           strconv.Itoa(req.Seats),
	}
	if req.Metadata != nil {
		if orderNo, ok := req.Metadata["order_no"]; ok {
			customData["order_no"] = orderNo
		}
	}

	// Helper function for bool pointers
	boolPtr := func(b bool) *bool { return &b }

	// Prepare checkout attributes
	checkoutAttrs := &lemonsqueezy.CheckoutCreateAttributes{
		CheckoutData: lemonsqueezy.CheckoutCreateData{
			Email:  req.UserEmail,
			Custom: customData,
			VariantQuantities: []lemonsqueezy.CheckoutCreateDataQuantity{
				{
					VariantID: stringToInt(variantID),
					Quantity:  req.Seats,
				},
			},
		},
		CheckoutOptions: lemonsqueezy.CheckoutCreateOptions{
			Embed:               boolPtr(false),
			Media:               boolPtr(true),
			Logo:                boolPtr(true),
			Desc:                boolPtr(true),
			Discount:            boolPtr(true),
			Dark:                boolPtr(false),
			SubscriptionPreview: boolPtr(true),
		},
		ProductOptions: lemonsqueezy.CheckoutCreateProductOptions{
			EnabledVariants: []int{stringToInt(variantID)},
			RedirectURL:     req.SuccessURL,
		},
	}

	// Set expiration time (30 minutes from now)
	expiresAt := time.Now().Add(30 * time.Minute).Format(time.RFC3339)
	checkoutAttrs.ExpiresAt = &expiresAt

	// Create checkout
	storeID := stringToInt(p.storeID)
	variantIDInt := stringToInt(variantID)

	checkout, _, err := p.client.Checkouts.Create(ctx, storeID, variantIDInt, checkoutAttrs)
	if err != nil {
		return nil, fmt.Errorf("failed to create LemonSqueezy checkout: %w", err)
	}

	// Get expiration time
	expiresAtTime := time.Now().Add(30 * time.Minute)
	if checkout.Data.Attributes.ExpiresAt != nil {
		expiresAtTime = *checkout.Data.Attributes.ExpiresAt
	}

	return &types.CheckoutResponse{
		SessionID:       checkout.Data.ID,
		SessionURL:      checkout.Data.Attributes.URL,
		OrderNo:         req.IdempotencyKey,
		ExternalOrderNo: checkout.Data.ID,
		ExpiresAt:       expiresAtTime,
	}, nil
}
