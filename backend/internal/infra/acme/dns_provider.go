package acme

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-acme/lego/v4/challenge/dns01"

	"github.com/anthropics/agentsmesh/backend/internal/infra/dns"
)

type dnsProviderAdapter struct {
	provider dns.Provider
	logger   *slog.Logger
}

func (d *dnsProviderAdapter) Present(domain, token, keyAuth string) error {
	fqdn, value := dns01.GetRecord(domain, keyAuth)

	d.logger.Info("Presenting DNS-01 challenge",
		"domain", domain,
		"fqdn", fqdn,
		"value_preview", value[:min(10, len(value))]+"...")

	recordName := fqdn
	if len(recordName) > 0 && recordName[len(recordName)-1] == '.' {
		recordName = recordName[:len(recordName)-1]
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := d.createTXTRecord(ctx, recordName, value); err != nil {
		return fmt.Errorf("failed to create TXT record: %w", err)
	}

	d.logger.Info("DNS-01 challenge TXT record created",
		"fqdn", fqdn)

	return nil
}

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
	}

	return nil
}

func (d *dnsProviderAdapter) createTXTRecord(ctx context.Context, fqdn, value string) error {
	txtProvider, ok := d.provider.(TXTRecordProvider)
	if !ok {
		return fmt.Errorf("DNS provider does not support TXT records")
	}

	return txtProvider.CreateTXTRecord(ctx, fqdn, value)
}

func (d *dnsProviderAdapter) deleteTXTRecord(ctx context.Context, fqdn string) error {
	txtProvider, ok := d.provider.(TXTRecordProvider)
	if !ok {
		return fmt.Errorf("DNS provider does not support TXT records")
	}

	return txtProvider.DeleteTXTRecord(ctx, fqdn)
}

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
