package alipay

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/smartwalle/alipay/v3"

	"github.com/anthropics/agentsmesh/backend/internal/service/payment/types"
)

// CreateAgreementSign creates a signing request for auto-debit agreement
// Note: This requires special Alipay merchant permissions
func (p *Provider) CreateAgreementSign(ctx context.Context, req *types.AgreementSignRequest) (*types.AgreementSignResponse, error) {
	// Generate external agreement number
	externalAgreementNo := fmt.Sprintf("org_%d_%d", req.OrganizationID, time.Now().Unix())

	// Build sign URL params (this would typically be done via SDK)
	signParams := fmt.Sprintf(
		"app_id=%s&method=alipay.user.agreement.page.sign&charset=utf-8&sign_type=RSA2"+
			"&personal_product_code=GENERAL_WITHHOLDING_P&sign_scene=INDUSTRY|DIGITAL_MEDIA"+
			"&external_agreement_no=%s&notify_url=%s&return_url=%s",
		p.appID,
		externalAgreementNo,
		p.notifyURL,
		req.ReturnURL,
	)

	return &types.AgreementSignResponse{
		SignURL:   fmt.Sprintf("https://openapi.alipay.com/gateway.do?%s", signParams),
		RequestNo: externalAgreementNo,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}, nil
}

// ExecuteAgreementPay executes a payment using the agreement
func (p *Provider) ExecuteAgreementPay(ctx context.Context, req *types.AgreementPayRequest) (*types.AgreementPayResponse, error) {
	pay := alipay.TradePay{
		Trade: alipay.Trade{
			Subject:     req.Description,
			OutTradeNo:  req.OrderNo,
			TotalAmount: fmt.Sprintf("%.2f", req.Amount),
			ProductCode: "GENERAL_WITHHOLDING",
		},
		AgreementParams: &alipay.AgreementParams{
			AgreementNo: req.AgreementNo,
		},
	}

	result, err := p.client.TradePay(ctx, pay)
	if err != nil {
		slog.Error("failed to execute alipay agreement pay", "order_no", req.OrderNo, "amount", req.Amount, "error", err)
		return nil, fmt.Errorf("failed to execute alipay agreement pay: %w", err)
	}

	if !result.IsSuccess() {
		slog.Error("alipay agreement pay failed", "order_no", req.OrderNo, "sub_code", result.SubCode, "sub_msg", result.SubMsg)
		return nil, fmt.Errorf("alipay agreement pay failed: %s - %s", result.SubCode, result.SubMsg)
	}

	paidAt := time.Now()
	slog.Info("alipay agreement pay succeeded", "order_no", req.OrderNo, "transaction_id", result.TradeNo, "amount", req.Amount)
	return &types.AgreementPayResponse{
		TransactionID: result.TradeNo,
		Status:        "success",
		Amount:        req.Amount,
		PaidAt:        &paidAt,
	}, nil
}

// CancelAgreement cancels an auto-debit agreement
// Note: This requires the agreement to be in an active state
func (p *Provider) CancelAgreement(ctx context.Context, agreementNo string) error {
	return fmt.Errorf("alipay agreement cancellation requires additional merchant configuration")
}

// GetAgreementStatus checks the status of an agreement
func (p *Provider) GetAgreementStatus(ctx context.Context, agreementNo string) (string, error) {
	return "", fmt.Errorf("alipay agreement query requires additional merchant configuration")
}
