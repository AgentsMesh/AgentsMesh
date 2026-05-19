package alipay

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/smartwalle/alipay/v3"

	"github.com/anthropics/agentsmesh/backend/internal/service/payment/types"
)

func (p *Provider) CreateAgreementSign(ctx context.Context, req *types.AgreementSignRequest) (*types.AgreementSignResponse, error) {
	externalAgreementNo := fmt.Sprintf("org_%d_%d", req.OrganizationID, time.Now().Unix())

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
		slog.ErrorContext(ctx, "failed to execute alipay agreement pay", "order_no", req.OrderNo, "amount", req.Amount, "error", err)
		return nil, fmt.Errorf("failed to execute alipay agreement pay: %w", err)
	}

	if !result.IsSuccess() {
		slog.ErrorContext(ctx, "alipay agreement pay failed", "order_no", req.OrderNo, "sub_code", result.SubCode, "sub_msg", result.SubMsg)
		return nil, fmt.Errorf("alipay agreement pay failed: %s - %s", result.SubCode, result.SubMsg)
	}

	paidAt := time.Now()
	slog.InfoContext(ctx, "alipay agreement pay succeeded", "order_no", req.OrderNo, "transaction_id", result.TradeNo, "amount", req.Amount)
	return &types.AgreementPayResponse{
		TransactionID: result.TradeNo,
		Status:        "success",
		Amount:        req.Amount,
		PaidAt:        &paidAt,
	}, nil
}

func (p *Provider) CancelAgreement(ctx context.Context, agreementNo string) error {
	return fmt.Errorf("alipay agreement cancellation requires additional merchant configuration")
}

func (p *Provider) GetAgreementStatus(ctx context.Context, agreementNo string) (string, error) {
	return "", fmt.Errorf("alipay agreement query requires additional merchant configuration")
}
