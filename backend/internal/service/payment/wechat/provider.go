package wechat

import (
	"context"
	"crypto/rsa"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/core/option"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/native"
	"github.com/wechatpay-apiv3/wechatpay-go/utils"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/payment/types"
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
