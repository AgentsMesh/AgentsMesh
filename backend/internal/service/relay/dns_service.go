package relay

import (
	"context"
	"fmt"
	"hash/fnv"
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/infra/dns"
	"github.com/anthropics/agentsmesh/backend/pkg/slugkit"
)

type DNSService struct {
	provider   dns.Provider
	baseDomain string
	enabled    bool
	logger     *slog.Logger
}

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

func (s *DNSService) IsEnabled() bool {
	return s.enabled
}

func (s *DNSService) GenerateRelayDomain(relayName string) string {
	name := slugkit.SanitizeDNS(relayName)
	if name == "" {
		h := fnv.New32a()
		h.Write([]byte(relayName))
		name = fmt.Sprintf("relay-%08x", h.Sum32())
	}

	return fmt.Sprintf("%s.%s", name, s.baseDomain)
}

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

func (s *DNSService) GetRecord(ctx context.Context, relayName string) (string, error) {
	if !s.enabled {
		return "", fmt.Errorf("dns service is not enabled")
	}

	domain := s.GenerateRelayDomain(relayName)
	return s.provider.GetRecord(ctx, domain)
}
