// AgentsMesh Backend Server
// Build version marker: 2026-02-06-fix-webhook-api-errors
package main

import (
	"context"
	"log"
	"log/slog"
	"os"

	grpcserver "github.com/anthropics/agentsmesh/backend/internal/api/grpc"
	"github.com/anthropics/agentsmesh/backend/internal/api/rest"
	v1 "github.com/anthropics/agentsmesh/backend/internal/api/rest/v1"
	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/infra/acme"
	"github.com/anthropics/agentsmesh/backend/internal/infra/database"
	"github.com/anthropics/agentsmesh/backend/internal/infra/logger"
	"github.com/anthropics/agentsmesh/backend/internal/service/agent"
	"github.com/anthropics/agentsmesh/backend/internal/service/agentpod"
	channelService "github.com/anthropics/agentsmesh/backend/internal/service/channel"
	notifDomain "github.com/anthropics/agentsmesh/backend/internal/domain/notification"
	"github.com/anthropics/agentsmesh/backend/internal/service/geo"
	"github.com/anthropics/agentsmesh/backend/internal/service/instance"
	loop "github.com/anthropics/agentsmesh/backend/internal/service/loop"
	notifService "github.com/anthropics/agentsmesh/backend/internal/service/notification"
	"github.com/anthropics/agentsmesh/backend/internal/service/relay"
	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
	runnerlogservice "github.com/anthropics/agentsmesh/backend/internal/service/runnerlog"
	"github.com/anthropics/agentsmesh/backend/internal/service/ticket"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	cfg.WarnInsecureDefaults()

	// Initialize logger
	appLogger, err := logger.New(logger.Config{
		Level:      cfg.Log.Level,
		Format:     cfg.Log.Format,
		FilePath:   cfg.Log.FilePath,
		MaxSizeMB:  cfg.Log.MaxSizeMB,
		MaxBackups: cfg.Log.MaxBackups,
	})
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer appLogger.Close()
	appLogger.SetDefault()
	slog.Info("Logger initialized", "level", cfg.Log.Level, "file", cfg.Log.FilePath)

	// Initialize database
	db, err := database.New(cfg.Database)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Initialize infrastructure and services
	hub, eventBus, redisClient := initializeInfrastructure(cfg, appLogger)
	services := initializeServices(cfg, db, redisClient)

	// Setup EventBus → Hub integration
	setupEventBusHub(eventBus, hub)

	// Setup event publishers
	ticketEventPublisher := ticket.NewEventBusPublisher(eventBus, appLogger.Logger)
	services.ticket.SetEventPublisher(ticketEventPublisher)
	podEventPublisher := agentpod.NewEventBusPublisher(eventBus, appLogger.Logger)
	services.pod.SetEventPublisher(podEventPublisher)
	services.channel.SetEventBus(eventBus)

	// Create notification dispatcher and register resolvers
	notifDispatcher := notifService.NewDispatcher(eventBus, services.notifPrefStore)
	notifDispatcher.RegisterResolver("pod_creator", notifService.NewPodCreatorResolver(services.pod))
	notifDispatcher.RegisterResolver("channel_members", notifService.NewChannelMemberResolver(services.channel))
	services.notifDispatcher = notifDispatcher

	// Register channel PostSendHooks (order matters: mention validation → event → notification → pod prompt)
	channelRepo := infra.NewChannelRepository(db)
	userLookup := infra.NewChannelUserLookup(db)
	podLookup := infra.NewChannelPodLookup(db)
	channelUserNames := infra.NewChannelUserNameResolver(db)
	services.channel.AddPostSendHook(channelService.NewMentionValidatorHook(userLookup, podLookup, channelRepo))
	services.channel.AddPostSendHook(channelService.NewEventPublishHook(eventBus, channelUserNames))
	services.channel.AddPostSendHook(channelService.NewNotificationHook(notifDispatcher, channelUserNames))
	slog.Info("Channel PostSendHooks registered")

	// Register user pre-delete hook for channel data cleanup (replaces FK CASCADE)
	services.user.AddPreDeleteHook(func(ctx context.Context, userID int64) error {
		return services.channel.CleanupUserReferences(ctx, userID)
	})

	// Start Redis subscriber for multi-instance sync
	if redisClient != nil {
		eventBus.StartRedisSubscriber(context.Background())
	}

	// Initialize Runner components
	runnerConnMgr, podCoordinator, terminalRouter, heartbeatBatcher, sandboxQuerySvc := initializeRunnerComponents(services.podRepo, services.runnerRepo, redisClient, appLogger, services.agentType)

	// Wire AutopilotRepository into PodCoordinator for autopilot event handling
	podCoordinator.SetAutopilotRepo(services.autopilotRepo)

	// Wire TokenUsageService into PodCoordinator for token usage recording
	podCoordinator.SetTokenUsageService(services.tokenUsage)

	// Initialize Relay services
	relayManager := relay.NewManagerWithOptions()
	relayTokenGenerator := relay.NewTokenGenerator(cfg.JWT.Secret, "agentsmesh-relay")
	relayDNSService, relayACMEManager := initializeRelayServices(cfg)
	slog.Info("Relay services initialized")

	// Initialize GeoIP resolver for geo-aware relay selection
	geoResolver := initializeGeoResolver()
	defer geoResolver.Close()

	// Setup terminal router event publishing
	terminalRouter.SetEventBus(eventBus)
	terminalRouter.SetPodInfoGetter(services.pod)

	// Route OSC terminal notifications through NotificationDispatcher (preference-aware)
	terminalRouter.SetNotifyFunc(func(ctx context.Context, orgID int64, source, entityID, title, body, link, resolver string) {
		if err := notifDispatcher.Dispatch(ctx, &notifDomain.NotificationRequest{
			OrganizationID:    orgID,
			Source:            source,
			SourceEntityID:    entityID,
			Title:             title,
			Body:              body,
			Link:              link,
			RecipientResolver: resolver,
		}); err != nil {
			slog.Error("failed to dispatch notification", "source", source, "error", err)
		}
	})

	// Wire PodPromptHook (must be after terminalRouter is initialized)
	services.channel.AddPostSendHook(channelService.NewPodPromptHook(terminalRouter, channelRepo))
	slog.Info("PodPromptHook registered with TerminalRouter")

	// Setup event callbacks
	setupRunnerEventCallbacks(db, runnerConnMgr, eventBus)
	setupPodEventCallbacks(db, podCoordinator, eventBus, notifDispatcher)

	// Create PodOrchestrator (unified Pod creation logic for REST + MCP paths)
	compositeProvider := agent.NewCompositeProvider(services.agentType, services.credentialProfile, services.userConfig)
	configBuilder := agent.NewConfigBuilder(compositeProvider)
	if services.extension != nil {
		configBuilder.SetExtensionProvider(services.extension)
		slog.Info("ExtensionProvider connected to ConfigBuilder")
	}
	podOrchestrator := agentpod.NewPodOrchestrator(&agentpod.PodOrchestratorDeps{
		PodService:        services.pod,
		ConfigBuilder:     configBuilder,
		PodCoordinator:    podCoordinator,
		BillingService:    services.billing,
		UserService:       services.user,
		RepoService:       services.repository,
		TicketService:     services.ticket,
		RunnerSelector:    services.runner,
		AgentTypeResolver: services.agentType,
		RunnerQuery:       services.runner,
	})
	slog.Info("PodOrchestrator created")

	// Initialize OrgAwarenessService (tracks which orgs this instance serves)
	runnerOrgQuerier := infra.NewRunnerOrgQuerier(db)
	orgAwareness := instance.NewOrgAwarenessService(runnerOrgQuerier, runnerConnMgr, redisClient, cfg.Server.Address, appLogger.Logger)
	orgAwareness.Start()
	setupOrgAwarenessRefresh(eventBus, orgAwareness)
	slog.Info("OrgAwarenessService started")

	// Wire AutopilotControllerService with PodCoordinator for gRPC command sending.
	// PodCoordinator implements AutopilotCommandSender (SendCreateAutopilot).
	services.autopilot.SetCommandSender(podCoordinator)

	// Initialize Loop orchestrator and scheduler
	loopOrchestrator := loop.NewLoopOrchestrator(services.loop, services.loopRun, eventBus, appLogger.Logger)
	loopOrchestrator.SetPodDependencies(podOrchestrator, services.autopilot, podCoordinator, services.ticket)
	loopScheduler := loop.NewLoopScheduler(services.loop, loopOrchestrator, orgAwareness, appLogger.Logger)
	loopScheduler.Start()
	setupLoopEventSubscriptions(eventBus, loopOrchestrator)
	slog.Info("Loop orchestrator and scheduler created")

	// Initialize PKI and gRPC
	var grpcRunnerHandler *v1.GRPCRunnerHandler
	var grpcServer *grpcserver.Server
	var sandboxQuerySender runner.SandboxQuerySender
	var upgradeCommandSender runner.UpgradeCommandSender
	var logUploadSender runner.LogUploadCommandSender
	if cfg.PKI.CACertFile != "" && cfg.PKI.CAKeyFile != "" {
		mcpDeps := &grpcserver.MCPDependencies{
			PodService:        services.pod,
			PodOrchestrator:   podOrchestrator,
			ChannelService:    services.channel,
			BindingService:    services.binding,
			TicketService:     services.ticket,
			RepositoryService: services.repository,
			RunnerService:     services.runner,
			AgentTypeSvc:      services.agentType,
			UserConfigSvc:     services.userConfig,
			TerminalRouter:    terminalRouter,
		}
		grpcServer, grpcRunnerHandler = initializePKIAndGRPC(cfg, services.runner, services.org, services.agentType, runnerConnMgr, appLogger, mcpDeps)
		if grpcServer != nil {
			grpcCommandSender := grpcserver.NewGRPCCommandSender(grpcServer.RunnerAdapter())
			podCoordinator.SetCommandSender(grpcCommandSender)
			terminalRouter.SetCommandSender(grpcCommandSender)
			sandboxQuerySender = grpcCommandSender
			upgradeCommandSender = grpcCommandSender
			logUploadSender = grpcCommandSender
			slog.Info("PodCoordinator and TerminalRouter connected to gRPC Server")
			setupRelayTokenRefreshCallback(db, runnerConnMgr, relayTokenGenerator, grpcCommandSender)
		}
	} else {
		slog.Warn("PKI CA files not configured, gRPC/mTLS disabled")
	}

	// Initialize Runner version checker (checks GitHub Releases for latest version)
	versionChecker := runner.NewVersionChecker(redisClient)
	if versionChecker != nil {
		versionChecker.Start(context.Background())
	}

	// Initialize runner log upload service (requires S3 storage)
	var logUploadSvc *runnerlogservice.Service
	if cfg.Storage.AccessKey != "" && cfg.Storage.SecretKey != "" {
		logUploadStorage := initializeLogUploadStorage(cfg)
		if logUploadStorage != nil {
			logUploadRepo := infra.NewRunnerLogRepository(db)
			logUploadSvc = runnerlogservice.NewService(logUploadRepo, logUploadStorage)

			// Register callback for log upload status events from Runner
			runnerConnMgr.SetLogUploadStatusCallback(func(runnerID int64, data *runnerv1.LogUploadStatusEvent) {
				logUploadSvc.HandleUploadStatus(runnerID, data.RequestId, data.Phase, data.Progress, data.Message, data.Error, data.SizeBytes)
			})
			slog.Info("Runner log upload service initialized")
		}
	} else {
		slog.Info("Runner log upload service disabled: storage not configured")
	}

	// Create services container
	svc := &v1.Services{
		Auth:               services.auth,
		User:               services.user,
		Org:                services.org,
		AgentType:          services.agentType,
		CredentialProfile:  services.credentialProfile,
		UserConfig:         services.userConfig,
		Repository:         services.repository,
		Webhook:            services.webhook,
		Runner:             services.runner,
		RunnerConnMgr:      runnerConnMgr,
		PodCoordinator:     podCoordinator,
		TerminalRouter:     terminalRouter,
		Pod:                services.pod,
		PodOrchestrator:    podOrchestrator,
		Autopilot:          services.autopilot,
		Channel:            services.channel,
		Binding:            services.binding,
		Ticket:             services.ticket,
		MRSync:             services.mrSync,
		Mesh:               services.mesh,
		Billing:            services.billing,
		Hub:                hub,
		EventBus:           eventBus,
		Invitation:         services.invitation,
		File:               services.file,
		PromoCode:          services.promoCode,
		AgentPodSettings:   services.agentpodSettings,
		AgentPodAIProvider: services.agentpodAIProvider,
		License:            services.license,
		APIKey:             services.apikey,
		APIKeyAdapter:      services.apikeyAdapter,
		GRPCRunnerHandler:  grpcRunnerHandler,
		SandboxQueryService: sandboxQuerySvc,
		SandboxQuerySender:   sandboxQuerySender,
		UpgradeCommandSender: upgradeCommandSender,
		LogUploadSender:      logUploadSender,
		LogUploadService:     logUploadSvc,
		RelayManager:        relayManager,
		RelayTokenGenerator: relayTokenGenerator,
		RelayDNSService:     relayDNSService,
		RelayACMEManager:    relayACMEManager,
		GeoResolver:         geoResolver,
		VersionChecker:      versionChecker,
		Extension:           services.extension,
		ExtensionRepo:       services.extensionRepo,
		MarketplaceWorker:   services.marketplaceWorker,
		Loop:                services.loop,
		LoopRun:             services.loopRun,
		LoopOrchestrator:    loopOrchestrator,
		LoopScheduler:       loopScheduler,
		SSO:                  services.sso,
		SupportTicket:       services.supportTicket,
		NotificationPrefStore: services.notifPrefStore,
		TokenUsage:            services.tokenUsage,
	}

	// Initialize router
	router := rest.NewRouter(cfg, svc, db, appLogger.Logger, redisClient)

	// Start MarketplaceWorker if configured
	if services.marketplaceWorker != nil {
		services.marketplaceWorker.Start(context.Background())
		slog.Info("MarketplaceWorker started")
	}
	defer func() {
		if services.marketplaceWorker != nil {
			services.marketplaceWorker.Stop()
			slog.Info("MarketplaceWorker stopped")
		}
	}()

	// Start scheduled jobs
	subscriptionScheduler := startSubscriptionJobs(db, cfg, services.email, appLogger.Logger)

	// Start HTTP server
	srv := startHTTPServer(cfg, router)

	// Graceful shutdown
	waitForShutdown(srv, grpcServer, eventBus, heartbeatBatcher, subscriptionScheduler, loopScheduler, orgAwareness, relayManager, db, redisClient)
}

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

// Build trigger: 20260119003527
// rebuild trigger
