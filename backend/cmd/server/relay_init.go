package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/infra/acme"
	"github.com/anthropics/agentsmesh/backend/internal/service/geo"
	"github.com/anthropics/agentsmesh/backend/internal/service/relay"
)

// initializeGeoResolver creates a GeoIP resolver.
// Tries GEO_MMDB_PATH env, then default Docker path /app/data/geoip.mmdb.
// Falls back to NoOpResolver if no MMDB file is available.
func initializeGeoResolver() geo.Resolver {
	mmdbPath := os.Getenv("GEO_MMDB_PATH")
	if mmdbPath == "" {
		mmdbPath = "/app/data/geoip.mmdb"
	}

	if _, err := os.Stat(mmdbPath); err == nil {
		resolver, err := geo.NewMMDBResolver(mmdbPath)
		if err != nil {
			slog.Warn("Failed to open GeoIP database, geo-aware relay disabled", "path", mmdbPath, "error", err)
			return geo.NewNoOpResolver()
		}
		slog.Info("GeoIP resolver initialized", "path", mmdbPath)
		return resolver
	}

	slog.Info("GeoIP database not found, geo-aware relay disabled", "path", mmdbPath)
	return geo.NewNoOpResolver()
}

// initializeRelayServices initializes Relay DNS and ACME services
func initializeRelayServices(cfg *config.Config) (*relay.DNSService, *acme.Manager) {
	var relayDNSService *relay.DNSService
	var relayACMEManager *acme.Manager

	if !cfg.Relay.IsEnabled() {
		return nil, nil
	}

	var err error
	relayDNSService, err = relay.NewDNSService(cfg.Relay)
	if err != nil {
		slog.Warn("Failed to initialize Relay DNS service", "error", err)
		return nil, nil
	}

	slog.Info("Relay DNS service initialized",
		"base_domain", cfg.Relay.BaseDomain,
		"provider", cfg.Relay.DNS.Provider)

	// Initialize ACME Manager if enabled
	if cfg.Relay.ACME.Enabled {
		dnsProvider := createDNSProvider(cfg.Relay)
		if dnsProvider != nil {
			relayACMEManager, err = acme.NewManager(acme.Config{
				DirectoryURL: cfg.Relay.ACME.DirectoryURL,
				Email:        cfg.Relay.ACME.Email,
				Domain:       cfg.Relay.BaseDomain,
				StorageDir:   cfg.Relay.ACME.StorageDir,
				DNSProvider:  dnsProvider,
				RenewalDays:  30,
			})
			if err != nil {
				slog.Error("Failed to initialize ACME manager", "error", err)
			} else {
				relayACMEManager.StartAutoRenewal(context.Background())
				slog.Info("ACME manager initialized",
					"domain", "*."+cfg.Relay.BaseDomain,
					"email", cfg.Relay.ACME.Email)
			}
		} else {
			slog.Warn("DNS provider not available, ACME disabled")
		}
	}

	return relayDNSService, relayACMEManager
}
