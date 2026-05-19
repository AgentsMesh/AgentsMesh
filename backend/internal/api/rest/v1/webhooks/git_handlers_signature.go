package webhooks

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"io"
	"strings"

	"github.com/gin-gonic/gin"
)

func (r *WebhookRouter) verifyGitHubSignature(c *gin.Context, secret string) bool {
	signature := c.GetHeader("X-Hub-Signature-256")
	if signature == "" {
		return false
	}

	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}
	expectedMAC := signature[7:] // Remove "sha256=" prefix

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		r.logger.Error("failed to read request body for signature verification", "error", err)
		return false
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	actualMAC := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expectedMAC), []byte(actualMAC))
}

func (r *WebhookRouter) verifyGiteeSignature(c *gin.Context, secret string) bool {
	timestamp := c.GetHeader("X-Gitee-Timestamp")
	token := c.GetHeader("X-Gitee-Token")

	if token == "" {
		return false
	}

	if timestamp != "" {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			r.logger.Error("failed to read request body for Gitee signature verification", "error", err)
			return false
		}
		c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

		stringToSign := timestamp + "\n" + string(body)
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte(stringToSign))
		expectedMAC := hex.EncodeToString(mac.Sum(nil))

		return hmac.Equal([]byte(token), []byte(expectedMAC))
	}

	return subtle.ConstantTimeCompare([]byte(token), []byte(secret)) == 1
}
