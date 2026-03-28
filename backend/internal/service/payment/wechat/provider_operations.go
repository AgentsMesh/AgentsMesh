package wechat

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/native"
	"github.com/wechatpay-apiv3/wechatpay-go/services/refunddomestic"
	"github.com/wechatpay-apiv3/wechatpay-go/utils"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/payment/types"
)

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
		slog.Error("failed to parse wechat notification", "error", err)
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
		slog.Error("failed to decrypt wechat notification", "notification_id", notification.ID, "error", err)
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

	slog.Info("wechat webhook processed", "notification_id", result.EventID, "order_no", result.OrderNo, "status", result.Status)
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
		slog.Error("failed to create wechat refund", "order_no", req.OrderNo, "amount", req.Amount, "error", err)
		return nil, fmt.Errorf("failed to create wechat refund: %w", err)
	}

	if result.Response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(result.Response.Body)
		slog.Error("wechat refund failed", "order_no", req.OrderNo, "status_code", result.Response.StatusCode)
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
		slog.Error("failed to close wechat order", "trade_no", subscriptionID, "error", err)
		return fmt.Errorf("failed to close wechat order: %w", err)
	}

	if result.Response.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(result.Response.Body)
		slog.Error("wechat close order failed", "trade_no", subscriptionID, "status_code", result.Response.StatusCode)
		return fmt.Errorf("wechat close order failed: %s", string(body))
	}

	slog.Info("wechat order closed", "trade_no", subscriptionID)
	return nil
}

// CreateAgreementSign creates a contract signing request (委托代扣签约)
func (p *Provider) CreateAgreementSign(ctx context.Context, req *types.AgreementSignRequest) (*types.AgreementSignResponse, error) {
	contractID := fmt.Sprintf("contract_org_%d_%d", req.OrganizationID, time.Now().Unix())

	signURL := fmt.Sprintf(
		"weixin://wxpay/papay?appid=%s&mch_id=%s&contract_id=%s&plan_id=%s&contract_display_account=%s&timestamp=%d&notify_url=%s&version=1.0&sign_type=HMAC-SHA256&sign=",
		p.appID,
		p.mchID,
		contractID,
		"agentsmesh_subscription",
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
	return nil, fmt.Errorf("wechat agreement pay requires additional merchant configuration")
}

// CancelAgreement cancels a contract (解约)
func (p *Provider) CancelAgreement(ctx context.Context, agreementNo string) error {
	return fmt.Errorf("wechat agreement cancellation requires additional merchant configuration")
}

// GetAgreementStatus checks the status of a contract
func (p *Provider) GetAgreementStatus(ctx context.Context, agreementNo string) (string, error) {
	return "", fmt.Errorf("wechat agreement query requires additional merchant configuration")
}
