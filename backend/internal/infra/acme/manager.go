package acme

import (
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/challenge/dns01"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"

	"github.com/anthropics/agentsmesh/backend/internal/infra/dns"
)

type Config struct {
	// ACME directory URL
	// Production: https://acme-v02.api.letsencrypt.org/directory
	// Staging: https://acme-staging-v02.api.letsencrypt.org/directory
	DirectoryURL string

	Email string

	Domain string

	StorageDir string

	DNSProvider dns.Provider

	// Certificate renewal threshold (default: 30 days before expiry)
	RenewalDays int
}

type Manager struct {
	cfg    Config
	client *lego.Client
	user   *acmeUser

	cert      *Certificate
	certMu    sync.RWMutex

	logger *slog.Logger
}

type Certificate struct {
	Domain      string    `json:"domain"`
	Certificate []byte    `json:"certificate"` // PEM encoded certificate chain
	PrivateKey  []byte    `json:"private_key"` // PEM encoded private key
	NotBefore   time.Time `json:"not_before"`
	NotAfter    time.Time `json:"not_after"`
	IssuedAt    time.Time `json:"issued_at"`
}

func NewManager(cfg Config) (*Manager, error) {
	if cfg.DirectoryURL == "" {
		cfg.DirectoryURL = lego.LEDirectoryProduction
	}
	if cfg.RenewalDays == 0 {
		cfg.RenewalDays = 30
	}
	if cfg.StorageDir == "" {
		cfg.StorageDir = "/var/lib/agentsmesh/acme"
	}

	if err := os.MkdirAll(cfg.StorageDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	m := &Manager{
		cfg:    cfg,
		logger: slog.With("component", "acme_manager"),
	}

	if err := m.loadOrCreateUser(); err != nil {
		return nil, fmt.Errorf("failed to load/create ACME user: %w", err)
	}

	legoConfig := lego.NewConfig(m.user)
	legoConfig.CADirURL = cfg.DirectoryURL
	legoConfig.Certificate.KeyType = certcrypto.EC256

	client, err := lego.NewClient(legoConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create ACME client: %w", err)
	}

	dnsProvider := &dnsProviderAdapter{provider: cfg.DNSProvider, logger: m.logger}
	if err := client.Challenge.SetDNS01Provider(dnsProvider, dns01.AddRecursiveNameservers([]string{"8.8.8.8:53", "1.1.1.1:53"})); err != nil {
		return nil, fmt.Errorf("failed to set DNS provider: %w", err)
	}

	m.client = client

	if m.user.Registration == nil {
		reg, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
		if err != nil {
			return nil, fmt.Errorf("failed to register ACME user: %w", err)
		}
		m.user.Registration = reg
		if err := m.saveUser(); err != nil {
			return nil, fmt.Errorf("failed to save user registration: %w", err)
		}
		m.logger.Info("ACME user registered", "email", cfg.Email)
	}

	if err := m.loadCertificate(); err != nil {
		m.logger.Warn("No existing certificate found", "error", err)
	}

	m.logger.Info("ACME manager initialized",
		"directory", cfg.DirectoryURL,
		"domain", cfg.Domain,
		"email", cfg.Email)

	return m, nil
}
