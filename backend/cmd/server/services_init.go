package main

import (
	"log/slog"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	agentpodDomain "github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
	runnerDomain "github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	blockstoreinfra "github.com/anthropics/agentsmesh/backend/internal/infra/blockstore"
	"github.com/anthropics/agentsmesh/backend/internal/infra/database"
	"github.com/anthropics/agentsmesh/backend/internal/infra/email"
	"github.com/anthropics/agentsmesh/backend/internal/service/agent"
	"github.com/anthropics/agentsmesh/backend/internal/service/agentpod"
	adminservice "github.com/anthropics/agentsmesh/backend/internal/service/admin"
	apikeyservice "github.com/anthropics/agentsmesh/backend/internal/service/apikey"
	"github.com/anthropics/agentsmesh/backend/internal/service/auth"
	"github.com/anthropics/agentsmesh/backend/internal/service/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/binding"
	blockstoreservice "github.com/anthropics/agentsmesh/backend/internal/service/blockstore"
	"github.com/anthropics/agentsmesh/backend/internal/service/channel"
	envbundleservice "github.com/anthropics/agentsmesh/backend/internal/service/envbundle"
	extensionservice "github.com/anthropics/agentsmesh/backend/internal/service/extension"
	fileservice "github.com/anthropics/agentsmesh/backend/internal/service/file"
	grantservice "github.com/anthropics/agentsmesh/backend/internal/service/grant"
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

var _ runner.PodStore = (*agentpod.PodService)(nil)

type serviceContainer struct {
	auth              *auth.Service
	user              *user.Service
	org               *organization.Service
	admin             *adminservice.Service
	adminDB           database.DB
	agentSvc          *agent.AgentService
	envBundle         *envbundleservice.Service
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
	message           *agent.MessageService
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
	blockstore        *blockstoreservice.Service
	grant             *grantservice.Service

	notifDispatcher *notifService.Dispatcher
	notifPrefStore  *notifService.PreferenceStore

	podRepo       agentpodDomain.PodRepository
	runnerRepo    runnerDomain.RunnerRepository
	autopilotRepo agentpodDomain.AutopilotRepository
}

func (s *serviceContainer) Close() {
	if s == nil {
		return
	}
	if s.blockstore != nil {
		s.blockstore.Close()
	}
}

func initializeServices(cfg *config.Config, db *gorm.DB, redisClient *redis.Client) *serviceContainer {
	userRepo := infra.NewUserRepository(db)
	userSvc := user.NewServiceWithEncryption(userRepo, cfg.JWT.Secret)
	userSvc.SetDisableOAuthSignup(cfg.OAuth.DisableSignup)
	authCfg := &auth.Config{
		JWTSecret:         cfg.JWT.Secret,
		JWTExpiration:     time.Duration(cfg.JWT.ExpirationHours) * time.Hour,
		RefreshExpiration: time.Duration(cfg.JWT.ExpirationHours*7) * time.Hour,
		Issuer:            "agentsmesh",
	}
	authSvc := auth.NewServiceWithRedis(authCfg, userSvc, redisClient)

	ssoRepo := infra.NewSSOConfigRepository(db)
	ssoSvc := ssoservice.NewServiceWithRedis(ssoRepo, cfg.JWT.Secret, cfg, redisClient)
	authSvc.SetSSOChecker(ssoSvc)

	encryptor := crypto.NewEncryptor(cfg.JWT.Secret)

	agentRepo := infra.NewAgentRepository(db)
	agentSvc := agent.NewAgentService(agentRepo)
	envBundleRepo := infra.NewEnvBundleRepository(db)
	envBundleSvc := envbundleservice.NewService(envBundleRepo, encryptor)
	userConfigRepo := infra.NewUserConfigRepository(db)
	userConfigSvc := agent.NewUserConfigService(userConfigRepo, agentSvc)

	gitRepoRepo := infra.NewGitProviderRepository(db)
	repoSvc := repository.NewService(gitRepoRepo)
	webhookSvc := repository.NewWebhookService(gitRepoRepo, cfg, userSvc, slog.Default())
	repoSvc.SetWebhookService(webhookSvc)
	billingRepo := infra.NewBillingRepository(db)
	billingSvc := billing.NewServiceWithConfig(billingRepo, cfg)
	orgRepo := infra.NewOrganizationRepository(db)
	orgSvc := organization.NewServiceWithBilling(orgRepo, billingSvc)
	runnerRepo := infra.NewRunnerRepository(db)
	runnerSvc := runner.NewService(runnerRepo, billingSvc)
	grantRepo := infra.NewGrantRepository(db)
	grantSvc := grantservice.NewService(grantRepo)
	runnerSvc.SetGrantQuerier(grantRepo)
	podRepo := infra.NewPodRepository(db)
	podSvc := agentpod.NewPodService(podRepo)
	autopilotRepo := infra.NewAutopilotRepository(db)
	autopilotSvc := agentpod.NewAutopilotControllerService(autopilotRepo)
	channelRepo := infra.NewChannelRepository(db)
	channelSvc := channel.NewService(channelRepo)
	ticketRepo := infra.NewTicketRepository(db)
	ticketSvc := ticket.NewService(ticketRepo)
	mrSyncRepo := infra.NewMRSyncRepository(db)
	mrSyncSvc := ticket.NewMRSyncService(mrSyncRepo, nil)
	bindingRepo := infra.NewBindingRepository(db)
	bindingSvc := binding.NewService(bindingRepo, podSvc)
	meshRepo := infra.NewMeshRepository(db)
	meshSvc := mesh.NewService(meshRepo, podSvc, channelSvc, bindingSvc)
	agentMessageRepo := infra.NewAgentMessageRepository(db)
	messageSvc := agent.NewMessageService(agentMessageRepo)

	emailSvc := email.NewService(email.Config{
		Provider:    cfg.Email.Provider,
		ResendKey:   cfg.Email.ResendKey,
		FromAddress: cfg.Email.FromAddress,
		BaseURL:     cfg.FrontendURL(),
	})
	invitationRepo := infra.NewInvitationRepository(db)
	invitationSvc := invitation.NewService(invitationRepo, emailSvc)

	promocodeRepo := infra.NewPromocodeRepository(db)
	promoCodeSvc := promocode.NewService(promocodeRepo, infra.NewGormBillingProvider(db))

	agentpodSettingsRepo := infra.NewSettingsRepository(db)
	agentpodSettingsSvc := agentpod.NewSettingsService(agentpodSettingsRepo)
	aiProviderRepo := infra.NewAIProviderRepository(db)
	agentpodAIProviderSvc := agentpod.NewAIProviderService(aiProviderRepo, encryptor)

	fileSvc := initializeFileService(cfg)
	supportTicketSvc := initializeSupportTicketService(cfg, db)

	apikeyRepo := infra.NewAPIKeyRepository(db)
	apikeySvc := apikeyservice.NewService(apikeyRepo, redisClient)
	apikeyAdapterSvc := apikeyservice.NewMiddlewareAdapter(apikeySvc)

	loopRepo := infra.NewLoopRepository(db)
	loopRunRepo := infra.NewLoopRunRepository(db)
	loopSvc := loop.NewLoopService(loopRepo)
	loopRunSvc := loop.NewLoopRunService(loopRunRepo)

	licenseSvc := initializeLicenseService(cfg, db)
	extSvc, extRepo, skillImp, mktWorker := initializeExtensionServices(cfg, db)

	notifPrefRepo := infra.NewNotificationPreferenceRepository(db)
	notifPrefStore := notifService.NewPreferenceStore(notifPrefRepo)

	tokenUsageRepo := infra.NewTokenUsageRepository(db)
	tokenUsageSvc := tokenusagesvc.NewService(tokenUsageRepo, slog.Default())

	blockstoreRepo := blockstoreinfra.NewRepository(db)
	blockstoreSvc := blockstoreservice.NewService(blockstoreRepo, slog.Default())
	if embedder := selectEmbedder(); embedder != nil {
		blockstoreSvc.SetEmbedder(embedder)
	}
	ticketSvc.SetBlockstore(blockstoreSvc)

	// admin.Service backs both the REST admin handlers (mounted by
	// admin.RegisterRoutes) and the Connect admin handlers (mounted by
	// connect_init's mountAdminServices). Keeping a single instance here
	// avoids two parallel audit-log pipelines.
	dbWrapper := database.NewGormWrapper(db)
	adminSvc := adminservice.NewService(dbWrapper)

	return &serviceContainer{
		auth:               authSvc,
		user:               userSvc,
		org:                orgSvc,
		admin:              adminSvc,
		adminDB:            dbWrapper,
		agentSvc:           agentSvc,
		envBundle:          envBundleSvc,
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
		message:            messageSvc,
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
		blockstore:         blockstoreSvc,
		grant:              grantSvc,
		podRepo:            podRepo,
		runnerRepo:         runnerRepo,
		autopilotRepo:      autopilotRepo,
	}
}
