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

// verifyGitHubSignature verifies GitHub webhook signature using HMAC-SHA256
func (r *WebhookRouter) verifyGitHubSignature(c *gin.Context, secret string) bool {
	// Get the signature from header
	signature := c.GetHeader("X-Hub-Signature-256")
	if signature == "" {
		return false
	}

	// Signature format: sha256=<hex_signature>
	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}
	expectedMAC := signature[7:] // Remove "sha256=" prefix

	// Read the request body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		r.logger.Error("failed to read request body for signature verification", "error", err)
		return false
	}
	// Restore the body for later processing
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

	// Calculate HMAC-SHA256
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	actualMAC := hex.EncodeToString(mac.Sum(nil))

	// Constant-time comparison to prevent timing attacks
	return hmac.Equal([]byte(expectedMAC), []byte(actualMAC))
}

// verifyGiteeSignature verifies Gitee webhook signature using HMAC-SHA256
func (r *WebhookRouter) verifyGiteeSignature(c *gin.Context, secret string) bool {
	// Gitee supports multiple signature methods
	timestamp := c.GetHeader("X-Gitee-Timestamp")
	token := c.GetHeader("X-Gitee-Token")

	if token == "" {
		return false
	}

	// Method 1: X-Gitee-Timestamp + X-Gitee-Token with HMAC (preferred, more secure)
	if timestamp != "" {
		// Read the request body
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			r.logger.Error("failed to read request body for Gitee signature verification", "error", err)
			return false
		}
		// Restore the body for later processing
		c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

		// Calculate HMAC-SHA256: timestamp + "\n" + body
		stringToSign := timestamp + "\n" + string(body)
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte(stringToSign))
		expectedMAC := hex.EncodeToString(mac.Sum(nil))

		return hmac.Equal([]byte(token), []byte(expectedMAC))
	}

	// Method 2: Simple X-Gitee-Token comparison (constant-time)
	return subtle.ConstantTimeCompare([]byte(token), []byte(secret)) == 1
}
