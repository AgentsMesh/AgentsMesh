package relay

import (
	"context"
	"fmt"
	"hash/fnv"
	"log/slog"
	"strings"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/infra/dns"
)

// DNSService manages DNS A records for relay servers.
// It is responsible only for DNS record management (domain → IP),
// not for URL generation (scheme, port, path).
type DNSService struct {
	provider   dns.Provider
	baseDomain string
	enabled    bool
	logger     *slog.Logger
}

// NewDNSService creates a new DNS service
func NewDNSService(cfg config.RelayConfig) (*DNSService, error) {
	svc := &DNSService{
		baseDomain: cfg.BaseDomain,
		enabled:    cfg.IsEnabled(),
		logger:     slog.With("component", "relay_dns_service"),
	}

	if !svc.enabled {
		svc.logger.Info("DNS service disabled - no base domain or provider configured")
		return svc, nil
	}

	// Create DNS provider
	provider, err := dns.NewProvider(cfg.DNS)
	if err != nil {
		return nil, fmt.Errorf("failed to create DNS provider: %w", err)
	}
	svc.provider = provider

	svc.logger.Info("DNS service initialized",
		"base_domain", cfg.BaseDomain,
		"provider", cfg.DNS.Provider)

	return svc, nil
}

// IsEnabled returns true if DNS management is enabled
func (s *DNSService) IsEnabled() bool {
	return s.enabled
}

// GenerateRelayDomain generates the full domain name for a relay
// e.g., "us-east-1" -> "us-east-1.relay.agentsmesh.cn"
func (s *DNSService) GenerateRelayDomain(relayName string) string {
	// sanitizeRelayName already handles lowercasing
	name := sanitizeRelayName(relayName)
	if name == "" {
		// Non-ASCII names sanitize to empty; use FNV hash for uniqueness
		// to prevent multiple relays from colliding on the same domain.
		h := fnv.New32a()
		h.Write([]byte(relayName))
		name = fmt.Sprintf("relay-%08x", h.Sum32())
	}

	return fmt.Sprintf("%s.%s", name, s.baseDomain)
}

// CreateRecord creates a DNS A record for a relay
func (s *DNSService) CreateRecord(ctx context.Context, relayName, ip string) error {
	if !s.enabled {
		return fmt.Errorf("dns service is not enabled")
	}

	domain := s.GenerateRelayDomain(relayName)

	s.logger.Info("Creating DNS record",
		"relay_name", relayName,
		"domain", domain,
		"ip", ip)

	if err := s.provider.CreateRecord(ctx, domain, ip); err != nil {
		return fmt.Errorf("failed to create DNS record for %s: %w", domain, err)
	}

	s.logger.Info("DNS record created successfully",
		"domain", domain,
		"ip", ip)

	return nil
}

// DeleteRecord deletes the DNS A record for a relay
func (s *DNSService) DeleteRecord(ctx context.Context, relayName string) error {
	if !s.enabled {
		return fmt.Errorf("dns service is not enabled")
	}

	domain := s.GenerateRelayDomain(relayName)

	s.logger.Info("Deleting DNS record",
		"relay_name", relayName,
		"domain", domain)

	if err := s.provider.DeleteRecord(ctx, domain); err != nil {
		return fmt.Errorf("failed to delete DNS record for %s: %w", domain, err)
	}

	s.logger.Info("DNS record deleted successfully", "domain", domain)

	return nil
}

// UpdateRecord updates the DNS A record for a relay
func (s *DNSService) UpdateRecord(ctx context.Context, relayName, ip string) error {
	if !s.enabled {
		return fmt.Errorf("dns service is not enabled")
	}

	domain := s.GenerateRelayDomain(relayName)

	s.logger.Info("Updating DNS record",
		"relay_name", relayName,
		"domain", domain,
		"ip", ip)

	if err := s.provider.UpdateRecord(ctx, domain, ip); err != nil {
		return fmt.Errorf("failed to update DNS record for %s: %w", domain, err)
	}

	s.logger.Info("DNS record updated successfully",
		"domain", domain,
		"ip", ip)

	return nil
}

// GetRecord returns the current IP for a relay domain
func (s *DNSService) GetRecord(ctx context.Context, relayName string) (string, error) {
	if !s.enabled {
		return "", fmt.Errorf("dns service is not enabled")
	}

	domain := s.GenerateRelayDomain(relayName)
	return s.provider.GetRecord(ctx, domain)
}

// sanitizeRelayName ensures the relay name is valid for DNS
// - lowercase
// - only alphanumeric and hyphens
// - no leading/trailing hyphens
// - max 63 characters (DNS label limit)
func sanitizeRelayName(name string) string {
	var result strings.Builder
	lastWasHyphen := true // prevent leading hyphen

	for _, r := range name {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			result.WriteRune(r)
			lastWasHyphen = false
		} else if r >= 'A' && r <= 'Z' {
			result.WriteRune(r - 'A' + 'a')
			lastWasHyphen = false
		} else if r == '-' || r == '_' || r == '.' {
			if !lastWasHyphen {
				result.WriteRune('-')
				lastWasHyphen = true
			}
		}
	}

	s := result.String()

	// Remove trailing hyphen
	s = strings.TrimRight(s, "-")

	// Limit to 63 characters
	if len(s) > 63 {
		s = s[:63]
		s = strings.TrimRight(s, "-")
	}

	return s
}
