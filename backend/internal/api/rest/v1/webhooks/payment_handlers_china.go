package webhooks

import (
	"encoding/json"
	"io"
	"net/http"

	billingdomain "github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
	"github.com/gin-gonic/gin"
)

// ===========================================
// Payment Webhook Handlers - China (Alipay, WeChat)
// ===========================================

// handleAlipayWebhook handles Alipay webhook events (异步通知)
func (r *WebhookRouter) handleAlipayWebhook(c *gin.Context) {
	// Check if Alipay is configured
	if r.paymentFactory == nil || !r.paymentFactory.IsProviderAvailable(billingdomain.PaymentProviderAlipay) {
		r.logger.Warn("Alipay webhook received but Alipay is not configured")
		c.String(http.StatusServiceUnavailable, "fail")
		return
	}

	// Read the form data (Alipay uses application/x-www-form-urlencoded)
	if err := c.Request.ParseForm(); err != nil {
		r.logger.Error("failed to parse Alipay webhook form", "error", err)
		c.String(http.StatusBadRequest, "fail")
		return
	}

	// Convert form values to JSON for our provider
	formData := make(map[string]string)
	for key, values := range c.Request.Form {
		if len(values) > 0 {
			formData[key] = values[0]
		}
	}
	payload, _ := json.Marshal(formData)

	// Get the Alipay provider
	provider, err := r.paymentFactory.GetProvider(billingdomain.PaymentProviderAlipay)
	if err != nil {
		r.logger.Error("failed to get Alipay provider", "error", err)
		c.String(http.StatusInternalServerError, "fail")
		return
	}

	// Parse and validate the webhook (signature is verified inside)
	event, err := provider.HandleWebhook(c.Request.Context(), payload, "")
	if err != nil {
		r.logger.Error("failed to validate Alipay webhook", "error", err)
		c.String(http.StatusBadRequest, "fail")
		return
	}

	r.logger.Info("received Alipay webhook",
		"event_id", event.EventID,
		"event_type", event.EventType,
		"order_no", event.OrderNo,
		"status", event.Status,
	)

	// Process the event based on status
	var processErr error
	switch event.Status {
	case billingdomain.OrderStatusSucceeded:
		processErr = r.billingSvc.HandlePaymentSucceeded(c, event)

	case billingdomain.OrderStatusFailed:
		processErr = r.billingSvc.HandlePaymentFailed(c, event)

	case billingdomain.OrderStatusCanceled:
		r.logger.Info("Alipay trade closed", "order_no", event.OrderNo)

	default:
		r.logger.Debug("ignoring Alipay event with pending status", "status", event.Status)
	}

	if processErr != nil {
		r.logger.Error("failed to process Alipay webhook",
			"error", processErr,
			"order_no", event.OrderNo,
			"status", event.Status,
		)
		c.String(http.StatusInternalServerError, "fail")
		return
	}

	// Alipay expects "success" string response (not JSON)
	c.String(http.StatusOK, "success")
}

// handleWeChatWebhook handles WeChat Pay webhook events (支付回调)
func (r *WebhookRouter) handleWeChatWebhook(c *gin.Context) {
	// Check if WeChat is configured
	if r.paymentFactory == nil || !r.paymentFactory.IsProviderAvailable(billingdomain.PaymentProviderWeChat) {
		r.logger.Warn("WeChat webhook received but WeChat is not configured")
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"code":    "FAIL",
			"message": "WeChat not configured",
		})
		return
	}

	// Read the request body
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		r.logger.Error("failed to read WeChat webhook body", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "FAIL",
			"message": "failed to read request body",
		})
		return
	}

	// Get signature headers for verification
	signature := c.GetHeader("Wechatpay-Signature")
	timestamp := c.GetHeader("Wechatpay-Timestamp")
	nonce := c.GetHeader("Wechatpay-Nonce")

	// Build verification string for provider
	verifyStr := timestamp + "|" + nonce + "|" + signature

	// Get the WeChat provider
	provider, err := r.paymentFactory.GetProvider(billingdomain.PaymentProviderWeChat)
	if err != nil {
		r.logger.Error("failed to get WeChat provider", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    "FAIL",
			"message": "provider not available",
		})
		return
	}

	// Parse and validate the webhook
	event, err := provider.HandleWebhook(c.Request.Context(), payload, verifyStr)
	if err != nil {
		r.logger.Error("failed to validate WeChat webhook", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "FAIL",
			"message": "invalid webhook",
		})
		return
	}

	r.logger.Info("received WeChat webhook",
		"event_id", event.EventID,
		"event_type", event.EventType,
		"order_no", event.OrderNo,
		"status", event.Status,
	)

	// Process the event based on status
	var processErr error
	switch event.Status {
	case billingdomain.OrderStatusSucceeded:
		processErr = r.billingSvc.HandlePaymentSucceeded(c, event)

	case billingdomain.OrderStatusFailed:
		processErr = r.billingSvc.HandlePaymentFailed(c, event)

	case billingdomain.OrderStatusCanceled:
		r.logger.Info("WeChat order closed", "order_no", event.OrderNo)

	default:
		r.logger.Debug("ignoring WeChat event with pending status", "status", event.Status)
	}

	if processErr != nil {
		r.logger.Error("failed to process WeChat webhook",
			"error", processErr,
			"order_no", event.OrderNo,
			"status", event.Status,
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    "FAIL",
			"message": "failed to process event",
		})
		return
	}

	// WeChat expects specific JSON response format
	c.JSON(http.StatusOK, gin.H{
		"code":    "SUCCESS",
		"message": "",
	})
}
