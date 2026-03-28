package alipay

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"github.com/smartwalle/alipay/v3"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/payment/types"
)

// Provider implements payment.AgreementProvider for Alipay
type Provider struct {
	client    *alipay.Client
	appID     string
	notifyURL string
	returnURL string
}

// NewProvider creates a new Alipay provider
// notifyURL and returnURL are derived from the application's primary domain
func NewProvider(cfg *config.AlipayConfig, notifyURL, returnURL string) (*Provider, error) {
	client, err := alipay.New(cfg.AppID, cfg.PrivateKey, !cfg.IsSandbox)
	if err != nil {
		return nil, fmt.Errorf("failed to create alipay client: %w", err)
	}

	// Load Alipay public key for signature verification
	if err := client.LoadAliPayPublicKey(cfg.AlipayPublicKey); err != nil {
		return nil, fmt.Errorf("failed to load alipay public key: %w", err)
	}

	return &Provider{
		client:    client,
		appID:     cfg.AppID,
		notifyURL: notifyURL,
		returnURL: returnURL,
	}, nil
}

// GetProviderName returns the provider name
func (p *Provider) GetProviderName() string {
	return billing.PaymentProviderAlipay
}

// CreateCheckoutSession creates a face-to-face QR code payment
func (p *Provider) CreateCheckoutSession(ctx context.Context, req *types.CheckoutRequest) (*types.CheckoutResponse, error) {
	trade := alipay.TradePreCreate{
		Trade: alipay.Trade{
			Subject:     fmt.Sprintf("AgentsMesh %s Subscription", req.BillingCycle),
			OutTradeNo:  req.IdempotencyKey,
			TotalAmount: fmt.Sprintf("%.2f", req.ActualAmount),
			ProductCode: "FACE_TO_FACE_PAYMENT",
			NotifyURL:   p.notifyURL,
		},
	}

	result, err := p.client.TradePreCreate(ctx, trade)
	if err != nil {
		slog.Error("failed to create alipay precreate", "order_no", req.IdempotencyKey, "amount", req.ActualAmount, "error", err)
		return nil, fmt.Errorf("failed to create alipay precreate: %w", err)
	}

	if !result.IsSuccess() {
		slog.Error("alipay precreate failed", "order_no", req.IdempotencyKey, "sub_code", result.SubCode, "sub_msg", result.SubMsg)
		return nil, fmt.Errorf("alipay precreate failed: %s - %s", result.SubCode, result.SubMsg)
	}

	slog.Info("alipay checkout session created", "order_no", req.IdempotencyKey, "amount", req.ActualAmount)
	return &types.CheckoutResponse{
		SessionID:       req.IdempotencyKey,
		OrderNo:         req.IdempotencyKey,
		ExternalOrderNo: req.IdempotencyKey,
		QRCodeURL:       result.QRCode,
		QRCodeData:      result.QRCode,
		ExpiresAt:       time.Now().Add(30 * time.Minute),
	}, nil
}

// GetCheckoutStatus checks the status of a payment
func (p *Provider) GetCheckoutStatus(ctx context.Context, sessionID string) (string, error) {
	query := alipay.TradeQuery{
		OutTradeNo: sessionID,
	}

	result, err := p.client.TradeQuery(ctx, query)
	if err != nil {
		return billing.OrderStatusPending, nil
	}

	if !result.IsSuccess() {
		return billing.OrderStatusPending, nil
	}

	switch result.TradeStatus {
	case alipay.TradeStatusSuccess:
		return billing.OrderStatusSucceeded, nil
	case alipay.TradeStatusClosed:
		return billing.OrderStatusCanceled, nil
	case alipay.TradeStatusWaitBuyerPay:
		return billing.OrderStatusPending, nil
	default:
		return billing.OrderStatusPending, nil
	}
}

// HandleWebhook parses and validates an Alipay async notification
func (p *Provider) HandleWebhook(ctx context.Context, payload []byte, signature string) (*types.WebhookEvent, error) {
	// Parse the form data from JSON
	var formData map[string]string
	if err := json.Unmarshal(payload, &formData); err != nil {
		slog.Error("failed to parse alipay notification", "error", err)
		return nil, fmt.Errorf("failed to parse alipay notification: %w", err)
	}

	// Convert to url.Values for verification
	values := make(url.Values)
	for k, v := range formData {
		values.Set(k, v)
	}

	// Verify signature
	if err := p.client.VerifySign(values); err != nil {
		slog.Error("alipay signature verification failed", "notify_id", formData["notify_id"], "error", err)
		return nil, fmt.Errorf("alipay signature verification failed: %w", err)
	}

	// Build webhook event
	result := &types.WebhookEvent{
		EventID:         formData["notify_id"],
		EventType:       formData["notify_type"],
		Provider:        billing.PaymentProviderAlipay,
		OrderNo:         formData["out_trade_no"],
		ExternalOrderNo: formData["trade_no"],
		Currency:        "CNY",
	}

	// Parse amount
	if totalAmount, ok := formData["total_amount"]; ok && totalAmount != "" {
		var amount float64
		_, _ = fmt.Sscanf(totalAmount, "%f", &amount)
		result.Amount = amount
	}

	// Map trade status
	switch formData["trade_status"] {
	case "TRADE_SUCCESS", "TRADE_FINISHED":
		result.Status = billing.OrderStatusSucceeded
	case "TRADE_CLOSED":
		result.Status = billing.OrderStatusCanceled
	case "WAIT_BUYER_PAY":
		result.Status = billing.OrderStatusPending
	default:
		result.Status = billing.OrderStatusPending
	}

	// Store raw payload
	result.RawPayload = make(map[string]interface{})
	for k, v := range formData {
		result.RawPayload[k] = v
	}

	slog.Info("alipay webhook processed", "notify_id", result.EventID, "order_no", result.OrderNo, "status", result.Status)
	return result, nil
}

// RefundPayment initiates a refund
func (p *Provider) RefundPayment(ctx context.Context, req *types.RefundRequest) (*types.RefundResponse, error) {
	refund := alipay.TradeRefund{
		OutTradeNo:   req.OrderNo,
		RefundAmount: fmt.Sprintf("%.2f", req.Amount),
		RefundReason: req.Reason,
		OutRequestNo: req.IdempotencyKey,
	}

	result, err := p.client.TradeRefund(ctx, refund)
	if err != nil {
		slog.Error("failed to create alipay refund", "order_no", req.OrderNo, "amount", req.Amount, "error", err)
		return nil, fmt.Errorf("failed to create alipay refund: %w", err)
	}

	if !result.IsSuccess() {
		slog.Error("alipay refund failed", "order_no", req.OrderNo, "sub_code", result.SubCode, "sub_msg", result.SubMsg)
		return nil, fmt.Errorf("alipay refund failed: %s - %s", result.SubCode, result.SubMsg)
	}

	slog.Info("alipay refund created", "order_no", req.OrderNo, "amount", req.Amount)
	return &types.RefundResponse{
		RefundID: req.IdempotencyKey,
		Status:   "success",
		Amount:   req.Amount,
		Currency: "CNY",
	}, nil
}

// CancelSubscription cancels a subscription (closes the trade if pending)
func (p *Provider) CancelSubscription(ctx context.Context, subscriptionID string, immediate bool) error {
	// For Alipay, we close the trade
	close := alipay.TradeClose{
		OutTradeNo: subscriptionID,
	}

	result, err := p.client.TradeClose(ctx, close)
	if err != nil {
		slog.Error("failed to close alipay trade", "trade_no", subscriptionID, "error", err)
		return fmt.Errorf("failed to close alipay trade: %w", err)
	}

	if !result.IsSuccess() && result.SubCode != "ACQ.TRADE_NOT_EXIST" {
		slog.Error("alipay trade close failed", "trade_no", subscriptionID, "sub_code", result.SubCode, "sub_msg", result.SubMsg)
		return fmt.Errorf("alipay trade close failed: %s - %s", result.SubCode, result.SubMsg)
	}

	slog.Info("alipay trade closed", "trade_no", subscriptionID)
	return nil
}
