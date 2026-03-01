package internal

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/relay"
	"github.com/AgentsMesh/AgentsMesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// Register handles relay registration
// POST /api/internal/relays/register
func (h *RelayHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	response := RegisterResponse{
		Status: "registered",
	}

	url := req.URL
	dnsCreated := false

	// Handle DNS auto-registration if relay_name and IP provided and DNS service enabled.
	// DNS service only manages A records (domain → IP).
	// The relay's URL (scheme + port) is authoritative from the relay itself.
	if req.RelayName != "" && req.IP != "" && h.dnsService != nil && h.dnsService.IsEnabled() {
		// Create/update DNS A record
		if err := h.dnsService.CreateRecord(c.Request.Context(), req.RelayName, req.IP); err != nil {
			h.logger.Error("Failed to create DNS record",
				"relay_name", req.RelayName,
				"ip", req.IP,
				"error", err)
			// Don't fail registration, just log the error
			// Relay can still work if URL is provided manually
		} else {
			// Replace host in relay's URL with DNS-generated domain, preserve scheme and port
			domain := h.dnsService.GenerateRelayDomain(req.RelayName)
			if newURL, err := replaceURLHost(url, domain); err == nil {
				url = newURL
			} else {
				h.logger.Warn("Failed to replace URL host, using relay-reported URL",
					"url", url,
					"domain", domain,
					"error", err)
			}
			dnsCreated = true
			h.logger.Info("DNS record created for relay",
				"relay_name", req.RelayName,
				"ip", req.IP,
				"url", url)
		}
	}

	// Validate that we have a URL (either provided or generated)
	if url == "" {
		apierr.InvalidInput(c, "url is required when DNS auto-registration is not available")
		return
	}

	info := &relay.RelayInfo{
		ID:          req.RelayID,
		URL:         url,
		InternalURL: req.InternalURL,
		Region:      req.Region,
		Capacity:    req.Capacity,
	}

	if info.Capacity == 0 {
		info.Capacity = 1000 // Default capacity
	}

	if info.Region == "" {
		info.Region = "default"
	}

	if err := h.relayManager.Register(info); err != nil {
		h.logger.Error("Failed to register relay", "relay_id", req.RelayID, "error", err)
		apierr.InternalError(c, "failed to register relay")
		return
	}

	h.logger.Info("Relay registered",
		"relay_id", req.RelayID,
		"url", url,
		"region", req.Region,
		"dns_created", dnsCreated)

	response.URL = url
	response.InternalURL = req.InternalURL
	response.DNSCreated = dnsCreated

	// Include TLS certificate if ACME is enabled and certificate is available
	if h.acmeManager != nil {
		cert, key, expiry, err := h.acmeManager.GetCertificatePEM()
		if err == nil && cert != "" {
			response.TLSCert = cert
			response.TLSKey = key
			response.TLSExpiry = expiry.Format(time.RFC3339)
			h.logger.Info("TLS certificate included in registration response",
				"relay_id", req.RelayID,
				"cert_expiry", expiry)
		} else if err != nil {
			h.logger.Warn("ACME certificate not available",
				"relay_id", req.RelayID,
				"error", err)
		}
	}

	c.JSON(http.StatusOK, response)
}

// Unregister handles graceful relay unregistration
// POST /api/internal/relays/unregister
func (h *RelayHandler) Unregister(c *gin.Context) {
	var req UnregisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	// Get relay info before unregistering
	relayInfo := h.relayManager.GetRelayByID(req.RelayID)
	if relayInfo == nil {
		// Relay not found, but that's OK for unregister (idempotent)
		h.logger.Info("Unregister request for unknown relay",
			"relay_id", req.RelayID,
			"reason", req.Reason)
		c.JSON(http.StatusOK, gin.H{"status": "not_found"})
		return
	}

	// Gracefully unregister
	h.relayManager.GracefulUnregister(req.RelayID, req.Reason)

	h.logger.Info("Relay gracefully unregistered",
		"relay_id", req.RelayID,
		"reason", req.Reason)

	c.JSON(http.StatusOK, gin.H{
		"status": "unregistered",
		"reason": req.Reason,
	})
}

// ForceUnregister removes a relay
// DELETE /api/internal/relays/:relay_id
func (h *RelayHandler) ForceUnregister(c *gin.Context) {
	relayID := c.Param("relay_id")
	if relayID == "" {
		apierr.InvalidInput(c, "relay_id is required")
		return
	}

	// Check if relay exists
	relayInfo := h.relayManager.GetRelayByID(relayID)
	if relayInfo == nil {
		apierr.ResourceNotFound(c, "relay not found")
		return
	}

	// Force unregister
	h.relayManager.ForceUnregister(relayID)

	h.logger.Info("Relay force unregistered", "relay_id", relayID)

	c.JSON(http.StatusOK, gin.H{
		"status":   "unregistered",
		"relay_id": relayID,
	})
}

// replaceURLHost parses rawURL and replaces only the hostname with newHost,
// preserving scheme, port, and path.
// e.g., replaceURLHost("wss://47.77.190.14:8443", "01.relay.agentsmesh.ai")
//
//	→ "wss://01.relay.agentsmesh.ai:8443"
func replaceURLHost(rawURL, newHost string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL %q: %w", rawURL, err)
	}

	port := u.Port()
	if port != "" {
		u.Host = newHost + ":" + port
	} else {
		u.Host = newHost
	}

	return u.String(), nil
}
