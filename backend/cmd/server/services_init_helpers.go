package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/infra/storage"
	extensionservice "github.com/anthropics/agentsmesh/backend/internal/service/extension"
	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
	fileservice "github.com/anthropics/agentsmesh/backend/internal/service/file"
	"github.com/anthropics/agentsmesh/backend/internal/service/license"
	supportticketservice "github.com/anthropics/agentsmesh/backend/internal/service/supportticket"
	"github.com/anthropics/agentsmesh/backend/pkg/crypto"
	"gorm.io/gorm"
)

func initializeFileService(cfg *config.Config) *fileservice.Service {
	if cfg.Storage.AccessKey == "" || cfg.Storage.SecretKey == "" {
		slog.Warn("Storage not configured, file upload disabled")
		return nil
	}

	s3Storage, err := storage.NewS3Storage(storage.S3Config{
		Endpoint:       cfg.Storage.Endpoint,
		PublicEndpoint: cfg.Storage.PublicEndpoint,
		Region:         cfg.Storage.Region,
		Bucket:         cfg.Storage.Bucket,
		AccessKey:      cfg.Storage.AccessKey,
		SecretKey:      cfg.Storage.SecretKey,
		UseSSL:         cfg.Storage.UseSSL,
		UsePathStyle:   cfg.Storage.UsePathStyle,
	})
	if err != nil {
		slog.Error("Failed to initialize storage", "error", err)
		return nil
	}

	if err := s3Storage.EnsureBucket(context.Background()); err != nil {
		slog.Warn("Failed to ensure bucket exists", "bucket", cfg.Storage.Bucket, "error", err)
	}

	slog.Info("Storage initialized", "endpoint", cfg.Storage.Endpoint, "bucket", cfg.Storage.Bucket)
	return fileservice.NewService(s3Storage, cfg.Storage)
}

func initializeLicenseService(cfg *config.Config, db *gorm.DB) *license.Service {
	if !cfg.Payment.IsOnPremise() && cfg.Payment.License.PublicKeyPath == "" {
		return nil
	}

	licenseRepo := infra.NewLicenseRepository(db)
	licenseSvc, err := license.NewService(licenseRepo, &cfg.Payment.License, slog.Default())
	if err != nil {
		slog.Warn("Failed to initialize license service", "error", err)
		return nil
	}

	slog.Info("License service initialized")
	return licenseSvc
}

func initializeExtensionServices(cfg *config.Config, db *gorm.DB) (*extensionservice.Service, extension.Repository, *extensionservice.SkillImporter, *extensionservice.MarketplaceWorker) {
	if cfg.Storage.AccessKey == "" || cfg.Storage.SecretKey == "" {
		slog.Warn("Storage not configured, extension services disabled")
		return nil, nil, nil, nil
	}

	s3Storage, err := storage.NewS3Storage(storage.S3Config{
		Endpoint:       cfg.Storage.Endpoint,
		PublicEndpoint: cfg.Storage.PublicEndpoint,
		Region:         cfg.Storage.Region,
		Bucket:         cfg.Storage.Bucket,
		AccessKey:      cfg.Storage.AccessKey,
		SecretKey:      cfg.Storage.SecretKey,
		UseSSL:         cfg.Storage.UseSSL,
		UsePathStyle:   cfg.Storage.UsePathStyle,
	})
	if err != nil {
		slog.Error("Failed to initialize storage for extensions", "error", err)
		return nil, nil, nil, nil
	}

	extRepo := infra.NewExtensionRepository(db)
	encryptor := crypto.NewEncryptor(cfg.JWT.Secret)
	extSvc := extensionservice.NewService(extRepo, s3Storage, encryptor)
	skillPkg := extensionservice.NewSkillPackager(extRepo, s3Storage)
	extSvc.SetSkillPackager(skillPkg)
	skillImp := extensionservice.NewSkillImporter(extRepo, s3Storage)
	extSvc.SetSkillImporter(skillImp)
	skillImp.SetCredentialDecryptor(extSvc.DecryptCredential)

	var mcpRegistrySyncer *extensionservice.McpRegistrySyncer
	if cfg.Marketplace.RegistryEnabled {
		mcpRegistryClient := extensionservice.NewMcpRegistryClient(cfg.Marketplace.RegistryURL)
		mcpRegistrySyncer = extensionservice.NewMcpRegistrySyncer(mcpRegistryClient, extRepo)
		slog.Info("MCP Registry syncer enabled", "url", cfg.Marketplace.RegistryURL)
	}

	syncInterval := cfg.Marketplace.SyncInterval
	if syncInterval == 0 {
		syncInterval = 1 * time.Hour
	}
	mktWorker := extensionservice.NewMarketplaceWorker(extRepo, skillImp, mcpRegistrySyncer, syncInterval)
	slog.Info("MarketplaceWorker configured", "interval", syncInterval)

	slog.Info("Extension services initialized")
	return extSvc, extRepo, skillImp, mktWorker
}

func initializeSupportTicketService(cfg *config.Config, db *gorm.DB) *supportticketservice.Service {
	supportTicketRepo := infra.NewSupportTicketRepository(db)

	if cfg.Storage.AccessKey == "" || cfg.Storage.SecretKey == "" {
		slog.Warn("Storage not configured, support ticket attachments disabled")
		return supportticketservice.NewService(supportTicketRepo, nil, cfg.Storage)
	}

	s3Storage, err := storage.NewS3Storage(storage.S3Config{
		Endpoint:       cfg.Storage.Endpoint,
		PublicEndpoint: cfg.Storage.PublicEndpoint,
		Region:         cfg.Storage.Region,
		Bucket:         cfg.Storage.Bucket,
		AccessKey:      cfg.Storage.AccessKey,
		SecretKey:      cfg.Storage.SecretKey,
		UseSSL:         cfg.Storage.UseSSL,
		UsePathStyle:   cfg.Storage.UsePathStyle,
	})
	if err != nil {
		slog.Error("Failed to initialize storage for support tickets", "error", err)
		return supportticketservice.NewService(supportTicketRepo, nil, cfg.Storage)
	}

	slog.Info("Support ticket service initialized")
	return supportticketservice.NewService(supportTicketRepo, s3Storage, cfg.Storage)
}

// initializeLogUploadStorage creates an S3 storage client for runner log uploads.
func initializeLogUploadStorage(cfg *config.Config) storage.Storage {
	s3Storage, err := storage.NewS3Storage(storage.S3Config{
		Endpoint:       cfg.Storage.Endpoint,
		PublicEndpoint: cfg.Storage.PublicEndpoint,
		Region:         cfg.Storage.Region,
		Bucket:         cfg.Storage.Bucket,
		AccessKey:      cfg.Storage.AccessKey,
		SecretKey:      cfg.Storage.SecretKey,
		UseSSL:         cfg.Storage.UseSSL,
		UsePathStyle:   cfg.Storage.UsePathStyle,
	})
	if err != nil {
		slog.Error("Failed to initialize storage for runner logs", "error", err)
		return nil
	}

	if err := s3Storage.EnsureBucket(context.Background()); err != nil {
		slog.Warn("Failed to ensure bucket for runner logs", "error", err)
	}

	return s3Storage
}
