package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
)

// Register registers this relay with the backend
func (c *Client) Register(ctx context.Context) error {
	// Auto-detect public IP if enabled and relay name is set
	if c.autoIP && c.relayName != "" && c.relayIP == "" {
		ip, err := c.detectPublicIP(ctx)
		if err != nil {
			c.logger.Warn("Failed to auto-detect public IP", "error", err)
		} else {
			c.relayIP = ip
			c.logger.Info("Auto-detected public IP", "ip", ip)
		}
	}

	req := RegisterRequest{
		RelayID:  c.relayID,
		RelayName: c.relayName,
		IP:       c.relayIP,
		URL:      c.relayURL,
		Region:   c.relayRegion,
		Capacity: c.relayCapacity,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal register request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/internal/relays/register", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create register request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Internal-Secret", c.internalAPISecret)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send register request: %w", err)
	}
	defer drainBody(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("registration failed with status: %d", resp.StatusCode)
	}

	// Parse response
	var regResp RegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&regResp); err != nil {
		c.logger.Warn("Failed to decode registration response", "error", err)
	} else {
		c.mu.Lock()
		// Update relay URL if returned from backend (DNS auto-registration)
		if regResp.URL != "" && regResp.DNSCreated {
			c.relayURL = regResp.URL
			c.logger.Info("DNS record created, using generated URL", "url", regResp.URL)
		}
		// Store TLS certificate if returned from backend (ACME)
		if regResp.TLSCert != "" && regResp.TLSKey != "" {
			c.tlsCert = regResp.TLSCert
			c.tlsKey = regResp.TLSKey
			c.tlsExpiry = regResp.TLSExpiry
			c.mu.Unlock()

			// Save certificate to files for persistence across restarts
			if err := c.saveCertificateFiles(regResp.TLSCert, regResp.TLSKey); err != nil {
				c.logger.Warn("Failed to save certificate files", "error", err)
			}

			c.logger.Info("TLS certificate received from backend", "expiry", regResp.TLSExpiry)
		} else {
			c.mu.Unlock()
		}
	}

	c.mu.Lock()
	c.registered = true
	c.mu.Unlock()

	c.logger.Info("Registered with backend", "relay_id", c.relayID, "url", c.relayURL)
	return nil
}

// NotifySessionClosed notifies backend that a session is closed
func (c *Client) NotifySessionClosed(ctx context.Context, podKey, sessionID string) error {
	req := SessionClosedRequest{
		PodKey:    podKey,
		SessionID: sessionID,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal session closed request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/internal/relays/session-closed", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create session closed request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Internal-Secret", c.internalAPISecret)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send session closed notification: %w", err)
	}
	defer drainBody(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("session closed notification failed with status: %d", resp.StatusCode)
	}

	c.logger.Info("Notified backend of session closed", "pod_key", podKey, "session_id", sessionID)
	return nil
}

// Unregister notifies backend that this relay is shutting down
func (c *Client) Unregister(ctx context.Context, reason string) error {
	req := UnregisterRequest{
		RelayID: c.relayID,
		Reason:  reason,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal unregister request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/internal/relays/unregister", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create unregister request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Internal-Secret", c.internalAPISecret)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send unregister request: %w", err)
	}
	defer drainBody(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unregistration failed with status: %d", resp.StatusCode)
	}

	c.mu.Lock()
	c.registered = false
	c.mu.Unlock()

	c.logger.Info("Unregistered from backend", "relay_id", c.relayID, "reason", reason)
	return nil
}

// detectPublicIP detects the public IP address using external services
func (c *Client) detectPublicIP(ctx context.Context) (string, error) {
	// List of public IP detection services
	services := []string{
		"https://api.ipify.org",
		"https://ifconfig.me/ip",
		"https://icanhazip.com",
	}

	for _, url := range services {
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			continue
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			continue
		}

		body, err := func() ([]byte, error) {
			defer drainBody(resp.Body)
			return io.ReadAll(io.LimitReader(resp.Body, 256)) // IP address is at most 45 bytes
		}()
		if err != nil {
			continue
		}

		ip := strings.TrimSpace(string(body))
		// Validate IP format with strict parsing
		if ip != "" && net.ParseIP(ip) != nil {
			return ip, nil
		}
	}

	return "", fmt.Errorf("failed to detect public IP from all services")
}
