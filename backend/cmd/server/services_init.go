package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	agentpodDomain "github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
	runnerDomain "github.com/anthropics/agentsmesh/backend/internal/domain/runner"
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
	ssoservice "github.com/anthropics/agentsmesh/backend/internal/service/sso"
	supportticketservice "github.com/anthropics/agentsmesh/backend/internal/service/supportticket"
	tokenusagesvc "github.com/anthropics/agentsmesh/backend/internal/service/tokenusage"
	"github.com/anthropics/agentsmesh/backend/internal/service/invitation"
	"github.com/anthropics/agentsmesh/backend/internal/service/license"
	loop "github.com/anthropics/agentsmesh/backend/internal/service/loop"
	"github.com/anthropics/agentsmesh/backend/internal/service/mesh"
	notifService "github.com/anthropics/agentsmesh/backend/internal/service/notification"
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
	loop              *loop.LoopService
	loopRun           *loop.LoopRunService
	sso               *ssoservice.Service
	supportTicket     *supportticketservice.Service
	tokenUsage        *tokenusagesvc.Service

	// Notification services
	notifDispatcher *notifService.Dispatcher
	notifPrefStore  *notifService.PreferenceStore

	// Repositories exposed for runner component wiring
	podRepo       agentpodDomain.PodRepository
	runnerRepo    runnerDomain.RunnerRepository
	autopilotRepo agentpodDomain.AutopilotRepository
}

// initializeServices creates all business services
func initializeServices(cfg *config.Config, db *gorm.DB, redisClient *redis.Client) *serviceContainer {
	// Use JWT secret as encryption key for token encryption (OAuth tokens, etc.)
	userRepo := infra.NewUserRepository(db)
	userSvc := user.NewServiceWithEncryption(userRepo, cfg.JWT.Secret)
	authCfg := &auth.Config{
		JWTSecret:         cfg.JWT.Secret,
		JWTExpiration:     time.Duration(cfg.JWT.ExpirationHours) * time.Hour,
		RefreshExpiration: time.Duration(cfg.JWT.ExpirationHours*7) * time.Hour, // 7x access token
		Issuer:            "agentsmesh",
	}
	authSvc := auth.NewServiceWithRedis(authCfg, userSvc, redisClient)

	// Initialize SSO service (with Redis for SAML InResponseTo validation)
	ssoRepo := infra.NewSSOConfigRepository(db)
	ssoSvc := ssoservice.NewServiceWithRedis(ssoRepo, cfg.JWT.Secret, cfg, redisClient)
	// Wire SSO enforcement checker into auth service
	authSvc.SetSSOChecker(ssoSvc)

	// Initialize encryptor for credential encryption (shared across services)
	encryptor := crypto.NewEncryptor(cfg.JWT.Secret)

	// Initialize agent sub-services (split by responsibility per SRP)
	agentTypeRepo := infra.NewAgentTypeRepository(db)
	agentTypeSvc := agent.NewAgentTypeService(agentTypeRepo)
	credentialProfileRepo := infra.NewCredentialProfileRepository(db)
	credentialProfileSvc := agent.NewCredentialProfileService(credentialProfileRepo, agentTypeSvc, encryptor)
	userConfigRepo := infra.NewUserConfigRepository(db)
	userConfigSvc := agent.NewUserConfigService(userConfigRepo, agentTypeSvc)

	gitRepoRepo := infra.NewGitProviderRepository(db)
	repoSvc := repository.NewService(gitRepoRepo)
	webhookSvc := repository.NewWebhookService(gitRepoRepo, cfg, userSvc, slog.Default())
	// Connect webhook service to repository service for automatic registration
	repoSvc.SetWebhookService(webhookSvc)
	billingRepo := infra.NewBillingRepository(db)
	billingSvc := billing.NewServiceWithConfig(billingRepo, cfg)
	// Organization service must be created after billing service so trial subscriptions
	// are automatically created when new organizations are created
	orgRepo := infra.NewOrganizationRepository(db)
	orgSvc := organization.NewServiceWithBilling(orgRepo, billingSvc)
	runnerRepo := infra.NewRunnerRepository(db)
	runnerSvc := runner.NewService(runnerRepo, billingSvc)
	podRepo := infra.NewPodRepository(db)
	podSvc := agentpod.NewPodService(podRepo)
	autopilotRepo := infra.NewAutopilotRepository(db)
	autopilotSvc := agentpod.NewAutopilotControllerService(autopilotRepo)
	channelRepo := infra.NewChannelRepository(db)
	channelSvc := channel.NewService(channelRepo)
	ticketRepo := infra.NewTicketRepository(db)
	ticketSvc := ticket.NewService(ticketRepo)
	// gitProvider is nil for webhook-only usage; batch sync functions won't work
	// but FindOrCreateMR and FindTicketByBranch work fine without it
	mrSyncRepo := infra.NewMRSyncRepository(db)
	mrSyncSvc := ticket.NewMRSyncService(mrSyncRepo, nil)
	bindingRepo := infra.NewBindingRepository(db)
	bindingSvc := binding.NewService(bindingRepo, podSvc)
	meshRepo := infra.NewMeshRepository(db)
	meshSvc := mesh.NewService(meshRepo, podSvc, channelSvc, bindingSvc)

	// Initialize email service for invitations
	emailSvc := email.NewService(email.Config{
		Provider:    cfg.Email.Provider,
		ResendKey:   cfg.Email.ResendKey,
		FromAddress: cfg.Email.FromAddress,
		BaseURL:     cfg.FrontendURL(),
	})
	invitationRepo := infra.NewInvitationRepository(db)
	invitationSvc := invitation.NewService(invitationRepo, emailSvc)

	// Initialize promo code service
	promocodeRepo := infra.NewPromocodeRepository(db)
	promoCodeSvc := promocode.NewService(promocodeRepo, infra.NewGormBillingProvider(db))

	// Initialize AgentPod settings and AI provider services
	agentpodSettingsRepo := infra.NewSettingsRepository(db)
	agentpodSettingsSvc := agentpod.NewSettingsService(agentpodSettingsRepo)
	aiProviderRepo := infra.NewAIProviderRepository(db)
	agentpodAIProviderSvc := agentpod.NewAIProviderService(aiProviderRepo, encryptor)

	// Initialize storage (S3-compatible)
	fileSvc := initializeFileService(cfg, db)

	// Initialize support ticket service (reuses file service's storage config)
	supportTicketSvc := initializeSupportTicketService(cfg, db)

	// Initialize API key service
	apikeyRepo := infra.NewAPIKeyRepository(db)
	apikeySvc := apikeyservice.NewService(apikeyRepo, redisClient)
	apikeyAdapterSvc := apikeyservice.NewMiddlewareAdapter(apikeySvc)

	// Initialize loop services
	loopRepo := infra.NewLoopRepository(db)
	loopRunRepo := infra.NewLoopRunRepository(db)
	loopSvc := loop.NewLoopService(loopRepo)
	loopRunSvc := loop.NewLoopRunService(loopRunRepo)

	// Initialize license service (for OnPremise deployments)
	licenseSvc := initializeLicenseService(cfg, db)

	// Initialize extension services (Skills marketplace, MCP servers)
	extSvc, extRepo, skillImp, mktWorker := initializeExtensionServices(cfg, db)

	// Initialize notification preference store
	notifPrefRepo := infra.NewNotificationPreferenceRepository(db)
	notifPrefStore := notifService.NewPreferenceStore(notifPrefRepo)

	// Initialize token usage service
	tokenUsageRepo := infra.NewTokenUsageRepository(db)
	tokenUsageSvc := tokenusagesvc.NewService(tokenUsageRepo, slog.Default())

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
		loop:               loopSvc,
		loopRun:            loopRunSvc,
		sso:                ssoSvc,
		supportTicket:      supportTicketSvc,
		notifPrefStore:     notifPrefStore,
		tokenUsage:         tokenUsageSvc,
		podRepo:            podRepo,
		runnerRepo:         runnerRepo,
		autopilotRepo:      autopilotRepo,
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
	fileRepo := infra.NewFileRepository(db)
	return fileservice.NewService(fileRepo, s3Storage, cfg.Storage)
}

// initializeLicenseService initializes the license service for OnPremise deployments
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

// initializeSupportTicketService initializes the support ticket service
func initializeSupportTicketService(cfg *config.Config, db *gorm.DB) *supportticketservice.Service {
	supportTicketRepo := infra.NewSupportTicketRepository(db)

	if cfg.Storage.AccessKey == "" || cfg.Storage.SecretKey == "" {
		slog.Warn("Storage not configured, support ticket attachments disabled")
		// Still create service with nil storage (text-only tickets work)
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
// Reuses the same S3 configuration as other storage services.
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
