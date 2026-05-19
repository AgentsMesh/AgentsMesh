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

const (
	defaultRelayCapacity = 1000      // max connections when not specified
	defaultRelayRegion   = "default" // region when not specified
)

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

	if req.RelayName != "" && req.IP != "" && h.dnsService != nil && h.dnsService.IsEnabled() {
		if err := h.dnsService.CreateRecord(c.Request.Context(), req.RelayName, req.IP); err != nil {
			h.logger.Error("Failed to create DNS record",
				"relay_name", req.RelayName,
				"ip", req.IP,
				"error", err)
		} else {
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

	if url == "" {
		apierr.InvalidInput(c, "url is required when DNS auto-registration is not available")
		return
	}

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

	if h.geoResolver != nil {
		relayIP := req.IP
		if relayIP == "" {
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
