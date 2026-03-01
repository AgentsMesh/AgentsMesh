package acme

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-acme/lego/v4/challenge/dns01"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/infra/dns"
)

// dnsProviderAdapter adapts our dns.Provider to lego's challenge.Provider interface
type dnsProviderAdapter struct {
	provider dns.Provider
	logger   *slog.Logger
}

// Present creates a TXT record for DNS-01 challenge
// domain: the domain being validated (e.g., "*.relay.example.com")
// token: unused for DNS-01
// keyAuth: the key authorization that needs to be placed in the TXT record
func (d *dnsProviderAdapter) Present(domain, token, keyAuth string) error {
	// Get the challenge info (FQDN and value)
	fqdn, value := dns01.GetRecord(domain, keyAuth)

	d.logger.Info("Presenting DNS-01 challenge",
		"domain", domain,
		"fqdn", fqdn,
		"value_preview", value[:min(10, len(value))]+"...")

	// Create TXT record
	// fqdn is like "_acme-challenge.relay.example.com."
	// We need to strip the trailing dot and create a TXT record
	recordName := fqdn
	if len(recordName) > 0 && recordName[len(recordName)-1] == '.' {
		recordName = recordName[:len(recordName)-1]
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Use CreateRecord to create/update the TXT record
	// Note: Our current DNS provider interface only supports A records
	// We need to extend it for TXT records
	if err := d.createTXTRecord(ctx, recordName, value); err != nil {
		return fmt.Errorf("failed to create TXT record: %w", err)
	}

	d.logger.Info("DNS-01 challenge TXT record created",
		"fqdn", fqdn)

	return nil
}

// CleanUp removes the TXT record after challenge is complete
func (d *dnsProviderAdapter) CleanUp(domain, token, keyAuth string) error {
	fqdn, _ := dns01.GetRecord(domain, keyAuth)

	d.logger.Info("Cleaning up DNS-01 challenge",
		"domain", domain,
		"fqdn", fqdn)

	recordName := fqdn
	if len(recordName) > 0 && recordName[len(recordName)-1] == '.' {
		recordName = recordName[:len(recordName)-1]
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := d.deleteTXTRecord(ctx, recordName); err != nil {
		d.logger.Warn("Failed to clean up TXT record", "fqdn", fqdn, "error", err)
		// Don't return error on cleanup failure
	}

	return nil
}

// createTXTRecord creates a TXT record using our DNS provider
// This requires extending the DNS provider interface
func (d *dnsProviderAdapter) createTXTRecord(ctx context.Context, fqdn, value string) error {
	// Check if provider supports TXT records
	txtProvider, ok := d.provider.(TXTRecordProvider)
	if !ok {
		return fmt.Errorf("DNS provider does not support TXT records")
	}

	return txtProvider.CreateTXTRecord(ctx, fqdn, value)
}

// deleteTXTRecord deletes a TXT record using our DNS provider
func (d *dnsProviderAdapter) deleteTXTRecord(ctx context.Context, fqdn string) error {
	txtProvider, ok := d.provider.(TXTRecordProvider)
	if !ok {
		return fmt.Errorf("DNS provider does not support TXT records")
	}

	return txtProvider.DeleteTXTRecord(ctx, fqdn)
}

// TXTRecordProvider interface for DNS providers that support TXT records
type TXTRecordProvider interface {
	CreateTXTRecord(ctx context.Context, fqdn, value string) error
	DeleteTXTRecord(ctx context.Context, fqdn string) error
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
