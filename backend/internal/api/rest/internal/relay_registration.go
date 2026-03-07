package internal

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/service/relay"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// Default values for relay registration fields.
const (
	defaultRelayCapacity = 1000      // max connections when not specified
	defaultRelayRegion   = "default" // region when not specified
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
	// Note: req.IP format is already validated by binding tag ("omitempty,ip").
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
				dnsCreated = true
			} else {
				h.logger.Warn("Failed to replace URL host, using relay-reported URL",
					"url", url,
					"domain", domain,
					"error", err)
			}
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

	// Validate URL scheme (only ws:// and wss:// allowed for relay WebSocket connections)
	parsedURL, err := parseRelayURL(url)
	if err != nil {
		apierr.InvalidInput(c, "url must use ws:// or wss:// scheme with a valid host")
		return
	}

	info := &relay.RelayInfo{
		ID:       req.RelayID,
		URL:      url,
		Region:   req.Region,
		Capacity: req.Capacity,
	}

	if info.Capacity == 0 {
		info.Capacity = defaultRelayCapacity
	}

	if info.Region == "" {
		info.Region = defaultRelayRegion
	}

	// Resolve relay geographic coordinates from its IP
	if h.geoResolver != nil {
		relayIP := req.IP
		if relayIP == "" {
			// Extract IP from parsed URL (only if it's an IP, not a hostname)
			if host := parsedURL.Hostname(); net.ParseIP(host) != nil {
				relayIP = host
			}
		}
		if relayIP != "" {
			if loc := h.geoResolver.Resolve(relayIP); loc != nil {
				info.Latitude = loc.Latitude
				info.Longitude = loc.Longitude
				h.logger.Info("Relay GeoIP resolved",
					"relay_id", req.RelayID,
					"ip", relayIP,
					"latitude", loc.Latitude,
					"longitude", loc.Longitude,
					"country", loc.Country)
			}
		}
	}

	if err := h.relayManager.Register(info); err != nil {
		h.logger.Error("Failed to register relay", "relay_id", req.RelayID, "error", err)
		// Best-effort rollback: clean up DNS record if we created one
		if dnsCreated && h.dnsService != nil {
			if delErr := h.dnsService.DeleteRecord(c.Request.Context(), req.RelayName); delErr != nil {
				h.logger.Warn("Failed to rollback DNS record after registration failure",
					"relay_name", req.RelayName, "error", delErr)
			}
		}
		if errors.Is(err, relay.ErrCapacityLimitReached) {
			apierr.CapacityExceeded(c, "relay capacity limit reached")
		} else {
			apierr.InternalError(c, "failed to register relay")
		}
		return
	}

	h.logger.Info("Relay registered",
		"relay_id", req.RelayID,
		"url", url,
		"region", req.Region,
		"dns_created", dnsCreated)

	response.URL = url
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

// replaceURLHost parses rawURL and replaces only the hostname with newHost,
// preserving scheme, port, and path.
// e.g., replaceURLHost("wss://47.77.190.14:8443", "01.relay.agentsmesh.ai")
//
//	→ "wss://01.relay.agentsmesh.ai:8443"
func replaceURLHost(rawURL, newHost string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	port := u.Port()
	if port != "" {
		u.Host = newHost + ":" + port
	} else {
		u.Host = newHost
	}

	return u.String(), nil
}

// parseRelayURL parses and validates a relay URL.
// Returns the parsed URL or an error if the scheme is not ws/wss or host is empty.
func parseRelayURL(rawURL string) (*url.URL, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid relay URL: %w", err)
	}
	if (u.Scheme != "ws" && u.Scheme != "wss") || u.Host == "" {
		return nil, fmt.Errorf("relay URL must use ws:// or wss:// scheme with a non-empty host")
	}
	return u, nil
}
