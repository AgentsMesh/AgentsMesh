package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/infra/email"
	"github.com/anthropics/agentsmesh/backend/internal/infra/storage"
	"github.com/anthropics/agentsmesh/backend/internal/service/agent"
	"github.com/anthropics/agentsmesh/backend/internal/service/agentpod"
	apikeyservice "github.com/anthropics/agentsmesh/backend/internal/service/apikey"
	"github.com/anthropics/agentsmesh/backend/internal/service/auth"
	"github.com/anthropics/agentsmesh/backend/internal/service/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/binding"
	"github.com/anthropics/agentsmesh/backend/internal/service/channel"
	extensionservice "github.com/anthropics/agentsmesh/backend/internal/service/extension"
	fileservice "github.com/anthropics/agentsmesh/backend/internal/service/file"
	"github.com/anthropics/agentsmesh/backend/internal/service/invitation"
	"github.com/anthropics/agentsmesh/backend/internal/service/license"
	"github.com/anthropics/agentsmesh/backend/internal/service/mesh"
	"github.com/anthropics/agentsmesh/backend/internal/service/organization"
	"github.com/anthropics/agentsmesh/backend/internal/service/promocode"
	"github.com/anthropics/agentsmesh/backend/internal/service/repository"
	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
	"github.com/anthropics/agentsmesh/backend/internal/service/ticket"
	"github.com/anthropics/agentsmesh/backend/internal/service/user"
	"github.com/anthropics/agentsmesh/backend/pkg/crypto"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// serviceContainer holds all initialized services
type serviceContainer struct {
	auth              *auth.Service
	user              *user.Service
	org               *organization.Service
	// Agent services (split by responsibility per SRP)
	agentType         *agent.AgentTypeService
	credentialProfile *agent.CredentialProfileService
	userConfig        *agent.UserConfigService
	repository        *repository.Service
	webhook           *repository.WebhookService
	runner            *runner.Service
	pod               *agentpod.PodService
	autopilot         *agentpod.AutopilotControllerService
	channel           *channel.Service
	ticket            *ticket.Service
	mrSync            *ticket.MRSyncService
	billing           *billing.Service
	binding           *binding.Service
	mesh              *mesh.Service
	invitation        *invitation.Service
	file              *fileservice.Service
	promoCode         *promocode.Service
	agentpodSettings   *agentpod.SettingsService
	agentpodAIProvider *agentpod.AIProviderService
	license           *license.Service
	apikey            *apikeyservice.Service
	apikeyAdapter     *apikeyservice.MiddlewareAdapter
	email             email.Service
	extension         *extensionservice.Service
	extensionRepo     extension.Repository
	skillImporter     *extensionservice.SkillImporter
	marketplaceWorker *extensionservice.MarketplaceWorker
}

// initializeServices creates all business services
func initializeServices(cfg *config.Config, db *gorm.DB, redisClient *redis.Client) *serviceContainer {
	// Use JWT secret as encryption key for token encryption (OAuth tokens, etc.)
	userSvc := user.NewServiceWithEncryption(db, cfg.JWT.Secret)
	authCfg := &auth.Config{
		JWTSecret:         cfg.JWT.Secret,
		JWTExpiration:     time.Duration(cfg.JWT.ExpirationHours) * time.Hour,
		RefreshExpiration: time.Duration(cfg.JWT.ExpirationHours*7) * time.Hour, // 7x access token
		Issuer:            "agentsmesh",
	}
	authSvc := auth.NewServiceWithRedis(authCfg, userSvc, redisClient)

	// Initialize agent sub-services (split by responsibility per SRP)
	agentTypeSvc := agent.NewAgentTypeService(db)
	credentialProfileSvc := agent.NewCredentialProfileService(db, agentTypeSvc)
	userConfigSvc := agent.NewUserConfigService(db, agentTypeSvc)

	repoSvc := repository.NewService(db)
	webhookSvc := repository.NewWebhookService(db, cfg, userSvc, slog.Default())
	// Connect webhook service to repository service for automatic registration
	repoSvc.SetWebhookService(webhookSvc)
	billingSvc := billing.NewServiceWithConfig(db, cfg)
	// Organization service must be created after billing service so trial subscriptions
	// are automatically created when new organizations are created
	orgSvc := organization.NewServiceWithBilling(db, billingSvc)
	runnerSvc := runner.NewService(db, billingSvc)
	podSvc := agentpod.NewPodService(db)
	autopilotSvc := agentpod.NewAutopilotControllerService(db)
	channelSvc := channel.NewService(db)
	ticketSvc := ticket.NewService(db)
	// gitProvider is nil for webhook-only usage; batch sync functions won't work
	// but FindOrCreateMR and FindTicketByBranch work fine without it
	mrSyncSvc := ticket.NewMRSyncService(db, nil)
	bindingSvc := binding.NewService(db, podSvc)
	meshSvc := mesh.NewService(db, podSvc, channelSvc, bindingSvc)

	// Initialize email service for invitations
	emailSvc := email.NewService(email.Config{
		Provider:    cfg.Email.Provider,
		ResendKey:   cfg.Email.ResendKey,
		FromAddress: cfg.Email.FromAddress,
		BaseURL:     cfg.FrontendURL(),
	})
	invitationSvc := invitation.NewService(db, emailSvc)

	// Initialize promo code service
	promoCodeSvc := promocode.NewService(db)

	// Initialize AgentPod settings and AI provider services
	agentpodSettingsSvc := agentpod.NewSettingsService(db)
	encryptor := crypto.NewEncryptor(cfg.JWT.Secret)
	agentpodAIProviderSvc := agentpod.NewAIProviderService(db, encryptor)

	// Initialize storage (S3-compatible)
	fileSvc := initializeFileService(cfg, db)

	// Initialize API key service
	apikeySvc := apikeyservice.NewService(db, redisClient)
	apikeyAdapterSvc := apikeyservice.NewMiddlewareAdapter(apikeySvc)

	// Initialize license service (for OnPremise deployments)
	licenseSvc := initializeLicenseService(cfg, db)

	// Initialize extension services (Skills marketplace, MCP servers)
	extSvc, extRepo, skillImp, mktWorker := initializeExtensionServices(cfg, db)

	return &serviceContainer{
		auth:               authSvc,
		user:               userSvc,
		org:                orgSvc,
		agentType:          agentTypeSvc,
		credentialProfile:  credentialProfileSvc,
		userConfig:         userConfigSvc,
		repository:         repoSvc,
		webhook:            webhookSvc,
		runner:             runnerSvc,
		pod:                podSvc,
		autopilot:          autopilotSvc,
		channel:            channelSvc,
		ticket:             ticketSvc,
		mrSync:             mrSyncSvc,
		billing:            billingSvc,
		binding:            bindingSvc,
		mesh:               meshSvc,
		invitation:         invitationSvc,
		file:               fileSvc,
		promoCode:          promoCodeSvc,
		agentpodSettings:   agentpodSettingsSvc,
		agentpodAIProvider: agentpodAIProviderSvc,
		license:            licenseSvc,
		apikey:             apikeySvc,
		apikeyAdapter:      apikeyAdapterSvc,
		email:              emailSvc,
		extension:          extSvc,
		extensionRepo:      extRepo,
		skillImporter:      skillImp,
		marketplaceWorker:  mktWorker,
	}
}

// initializeFileService initializes the file storage service
func initializeFileService(cfg *config.Config, db *gorm.DB) *fileservice.Service {
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

	// Ensure bucket exists
	if err := s3Storage.EnsureBucket(context.Background()); err != nil {
		slog.Warn("Failed to ensure bucket exists", "bucket", cfg.Storage.Bucket, "error", err)
	}

	slog.Info("Storage initialized", "endpoint", cfg.Storage.Endpoint, "bucket", cfg.Storage.Bucket)
	return fileservice.NewService(db, s3Storage, cfg.Storage)
}

// initializeLicenseService initializes the license service for OnPremise deployments
func initializeLicenseService(cfg *config.Config, db *gorm.DB) *license.Service {
	if !cfg.Payment.IsOnPremise() && cfg.Payment.License.PublicKeyPath == "" {
		return nil
	}

	licenseSvc, err := license.NewService(db, &cfg.Payment.License, slog.Default())
	if err != nil {
		slog.Warn("Failed to initialize license service", "error", err)
		return nil
	}

	slog.Info("License service initialized")
	return licenseSvc
}

// initializeExtensionServices initializes extension services (Skills, MCP servers, Marketplace)
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

	// Initialize MCP Registry syncer (optional, enabled by default)
	var mcpRegistrySyncer *extensionservice.McpRegistrySyncer
	if cfg.Marketplace.RegistryEnabled {
		mcpRegistryClient := extensionservice.NewMcpRegistryClient(cfg.Marketplace.RegistryURL)
		mcpRegistrySyncer = extensionservice.NewMcpRegistrySyncer(mcpRegistryClient, extRepo)
		slog.Info("MCP Registry syncer enabled", "url", cfg.Marketplace.RegistryURL)
	}

	// Always create MarketplaceWorker — sources are now managed via Admin API (DB)
	syncInterval := cfg.Marketplace.SyncInterval
	if syncInterval == 0 {
		syncInterval = 1 * time.Hour
	}
	mktWorker := extensionservice.NewMarketplaceWorker(extRepo, skillImp, mcpRegistrySyncer, syncInterval)
	slog.Info("MarketplaceWorker configured", "interval", syncInterval)

	slog.Info("Extension services initialized")
	return extSvc, extRepo, skillImp, mktWorker
}
