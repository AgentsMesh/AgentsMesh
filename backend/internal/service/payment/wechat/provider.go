package wechat

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/core/option"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/native"
	"github.com/wechatpay-apiv3/wechatpay-go/services/refunddomestic"
	"github.com/wechatpay-apiv3/wechatpay-go/utils"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/config"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/payment/types"
)

// Provider implements payment.AgreementProvider for WeChat Pay
type Provider struct {
	client     *core.Client
	appID      string
	mchID      string
	apiV3Key   string
	notifyURL  string
	privateKey *rsa.PrivateKey
}

// NewProvider creates a new WeChat Pay provider
// notifyURL is derived from the application's primary domain
func NewProvider(cfg *config.WeChatConfig, notifyURL string) (*Provider, error) {
	// Load merchant private key
	privateKey, err := utils.LoadPrivateKeyWithPath(cfg.KeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load wechat private key: %w", err)
	}

	// Read certificate serial number from cert file
	cert, err := utils.LoadCertificateWithPath(cfg.CertPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load wechat cert: %w", err)
	}
	certSerialNo := utils.GetCertificateSerialNumber(*cert)

	ctx := context.Background()

	// Create client with auto certificate updating
	opts := []core.ClientOption{
		option.WithWechatPayAutoAuthCipher(cfg.MchID, certSerialNo, privateKey, cfg.APIv3Key),
	}

	client, err := core.NewClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create wechat client: %w", err)
	}

	return &Provider{
		client:     client,
		appID:      cfg.AppID,
		mchID:      cfg.MchID,
		apiV3Key:   cfg.APIv3Key,
		notifyURL:  notifyURL,
		privateKey: privateKey,
	}, nil
}

// GetProviderName returns the provider name
func (p *Provider) GetProviderName() string {
	return billing.PaymentProviderWeChat
}

// CreateCheckoutSession creates a Native payment (扫码支付)
func (p *Provider) CreateCheckoutSession(ctx context.Context, req *types.CheckoutRequest) (*types.CheckoutResponse, error) {
	svc := native.NativeApiService{Client: p.client}

	// Amount in cents (分)
	amountCents := int64(req.ActualAmount * 100)

	prepayReq := native.PrepayRequest{
		Appid:       core.String(p.appID),
		Mchid:       core.String(p.mchID),
		Description: core.String(fmt.Sprintf("AgentsMesh %s Subscription", req.BillingCycle)),
		OutTradeNo:  core.String(req.IdempotencyKey),
		NotifyUrl:   core.String(p.notifyURL),
		Amount: &native.Amount{
			Total:    core.Int64(amountCents),
			Currency: core.String("CNY"),
		},
		TimeExpire: core.Time(time.Now().Add(30 * time.Minute)),
		Attach:     core.String(fmt.Sprintf("org_%d", req.OrganizationID)),
	}

	resp, result, err := svc.Prepay(ctx, prepayReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create wechat prepay: %w", err)
	}

	if result.Response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(result.Response.Body)
		return nil, fmt.Errorf("wechat prepay failed: %s", string(body))
	}

	return &types.CheckoutResponse{
		SessionID:       req.IdempotencyKey,
		OrderNo:         req.IdempotencyKey,
		ExternalOrderNo: req.IdempotencyKey,
		QRCodeURL:       *resp.CodeUrl,
		QRCodeData:      *resp.CodeUrl,
		ExpiresAt:       time.Now().Add(30 * time.Minute),
	}, nil
}

// GetCheckoutStatus checks the status of a payment
func (p *Provider) GetCheckoutStatus(ctx context.Context, sessionID string) (string, error) {
	svc := native.NativeApiService{Client: p.client}

	resp, result, err := svc.QueryOrderByOutTradeNo(ctx, native.QueryOrderByOutTradeNoRequest{
		OutTradeNo: core.String(sessionID),
		Mchid:      core.String(p.mchID),
	})
	if err != nil {
		return "", fmt.Errorf("failed to query wechat order: %w", err)
	}

	if result.Response.StatusCode != http.StatusOK {
		return billing.OrderStatusPending, nil
	}

	switch *resp.TradeState {
	case "SUCCESS":
		return billing.OrderStatusSucceeded, nil
	case "CLOSED":
		return billing.OrderStatusCanceled, nil
	case "NOTPAY", "USERPAYING":
		return billing.OrderStatusPending, nil
	case "PAYERROR":
		return billing.OrderStatusFailed, nil
	default:
		return billing.OrderStatusPending, nil
	}
}

// HandleWebhook parses and validates a WeChat Pay notification
func (p *Provider) HandleWebhook(ctx context.Context, payload []byte, signature string) (*types.WebhookEvent, error) {
	// Parse notification content
	var notification struct {
		ID           string `json:"id"`
		CreateTime   string `json:"create_time"`
		ResourceType string `json:"resource_type"`
		EventType    string `json:"event_type"`
		Resource     struct {
			Ciphertext     string `json:"ciphertext"`
			AssociatedData string `json:"associated_data"`
			Nonce          string `json:"nonce"`
		} `json:"resource"`
	}

	if err := json.Unmarshal(payload, &notification); err != nil {
		return nil, fmt.Errorf("failed to parse wechat notification: %w", err)
	}

	result := &types.WebhookEvent{
		EventID:   notification.ID,
		EventType: notification.EventType,
		Provider:  billing.PaymentProviderWeChat,
		Currency:  "CNY",
	}

	// Decrypt the resource content
	plaintext, err := utils.DecryptAES256GCM(
		p.apiV3Key,
		notification.Resource.AssociatedData,
		notification.Resource.Nonce,
		notification.Resource.Ciphertext,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt wechat notification: %w", err)
	}

	var paymentResult struct {
		TransactionID string `json:"transaction_id"`
		OutTradeNo    string `json:"out_trade_no"`
		TradeState    string `json:"trade_state"`
		Amount        struct {
			Total int64 `json:"total"`
		} `json:"amount"`
	}

	if err := json.Unmarshal([]byte(plaintext), &paymentResult); err != nil {
		return nil, fmt.Errorf("failed to parse wechat payment result: %w", err)
	}

	result.OrderNo = paymentResult.OutTradeNo
	result.ExternalOrderNo = paymentResult.TransactionID
	result.Amount = float64(paymentResult.Amount.Total) / 100

	switch paymentResult.TradeState {
	case "SUCCESS":
		result.Status = billing.OrderStatusSucceeded
	case "CLOSED":
		result.Status = billing.OrderStatusCanceled
	case "PAYERROR":
		result.Status = billing.OrderStatusFailed
	default:
		result.Status = billing.OrderStatusPending
	}

	result.RawPayload = make(map[string]interface{})
	_ = json.Unmarshal([]byte(plaintext), &result.RawPayload)

	return result, nil
}

// RefundPayment initiates a refund
func (p *Provider) RefundPayment(ctx context.Context, req *types.RefundRequest) (*types.RefundResponse, error) {
	svc := refunddomestic.RefundsApiService{Client: p.client}

	amountCents := int64(req.Amount * 100)

	refundReq := refunddomestic.CreateRequest{
		OutTradeNo:  core.String(req.OrderNo),
		OutRefundNo: core.String(req.IdempotencyKey),
		Reason:      core.String(req.Reason),
		Amount: &refunddomestic.AmountReq{
			Refund:   core.Int64(amountCents),
			Total:    core.Int64(amountCents), // Assuming full refund
			Currency: core.String("CNY"),
		},
	}

	resp, result, err := svc.Create(ctx, refundReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create wechat refund: %w", err)
	}

	if result.Response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(result.Response.Body)
		return nil, fmt.Errorf("wechat refund failed: %s", string(body))
	}

	status := "processing"
	if resp.Status != nil {
		switch *resp.Status {
		case "SUCCESS":
			status = "success"
		case "CLOSED":
			status = "failed"
		}
	}

	return &types.RefundResponse{
		RefundID: *resp.RefundId,
		Status:   status,
		Amount:   float64(*resp.Amount.Refund) / 100,
		Currency: "CNY",
	}, nil
}

// CancelSubscription closes a pending order
func (p *Provider) CancelSubscription(ctx context.Context, subscriptionID string, immediate bool) error {
	svc := native.NativeApiService{Client: p.client}

	result, err := svc.CloseOrder(ctx, native.CloseOrderRequest{
		OutTradeNo: core.String(subscriptionID),
		Mchid:      core.String(p.mchID),
	})
	if err != nil {
		return fmt.Errorf("failed to close wechat order: %w", err)
	}

	if result.Response.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(result.Response.Body)
		return fmt.Errorf("wechat close order failed: %s", string(body))
	}

	return nil
}

// CreateAgreementSign creates a contract signing request (委托代扣签约)
func (p *Provider) CreateAgreementSign(ctx context.Context, req *types.AgreementSignRequest) (*types.AgreementSignResponse, error) {
	// WeChat Pay contract signing (微信支付委托代扣)
	// This requires additional API permissions from WeChat
	// For now, return a placeholder that indicates the feature needs configuration

	contractID := fmt.Sprintf("contract_org_%d_%d", req.OrganizationID, time.Now().Unix())

	// Build the contract signing URL
	// Note: This is a simplified version. Production implementation would use
	// the wechatpay-go SDK's withhold API when available
	signURL := fmt.Sprintf(
		"weixin://wxpay/papay?appid=%s&mch_id=%s&contract_id=%s&plan_id=%s&contract_display_account=%s&timestamp=%d&notify_url=%s&version=1.0&sign_type=HMAC-SHA256&sign=",
		p.appID,
		p.mchID,
		contractID,
		"agentsmesh_subscription", // Plan ID configured in WeChat merchant portal
		req.UserEmail,
		time.Now().Unix(),
		p.notifyURL,
	)

	return &types.AgreementSignResponse{
		SignURL:   signURL,
		RequestNo: contractID,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}, nil
}

// ExecuteAgreementPay executes a payment using the contract (代扣)
func (p *Provider) ExecuteAgreementPay(ctx context.Context, req *types.AgreementPayRequest) (*types.AgreementPayResponse, error) {
	// WeChat Pay withhold payment (微信支付代扣)
	// This requires the contract to be signed first

	// Note: Production implementation would use the wechatpay-go SDK's
	// withhold/pappay API. This is a placeholder structure.

	return nil, fmt.Errorf("wechat agreement pay requires additional merchant configuration")
}

// CancelAgreement cancels a contract (解约)
func (p *Provider) CancelAgreement(ctx context.Context, agreementNo string) error {
	// WeChat Pay contract termination
	// This would call the contract termination API

	return fmt.Errorf("wechat agreement cancellation requires additional merchant configuration")
}

// GetAgreementStatus checks the status of a contract
func (p *Provider) GetAgreementStatus(ctx context.Context, agreementNo string) (string, error) {
	// Query contract status
	// This would call the contract query API

	return "", fmt.Errorf("wechat agreement query requires additional merchant configuration")
}
