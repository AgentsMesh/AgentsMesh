// AgentsMesh Backend Server
// Build version marker: 2026-02-06-fix-webhook-api-errors
package main

import (
	"context"
	"log"
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	notifDomain "github.com/anthropics/agentsmesh/backend/internal/domain/notification"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/infra/database"
	"github.com/anthropics/agentsmesh/backend/internal/infra/logger"
	"github.com/anthropics/agentsmesh/backend/internal/infra/websocket"
	"github.com/anthropics/agentsmesh/backend/internal/api/rest"
	"github.com/anthropics/agentsmesh/backend/internal/service/agentpod"
	channelService "github.com/anthropics/agentsmesh/backend/internal/service/channel"
	"github.com/anthropics/agentsmesh/backend/internal/service/instance"
	loop "github.com/anthropics/agentsmesh/backend/internal/service/loop"
	notifService "github.com/anthropics/agentsmesh/backend/internal/service/notification"
	"github.com/anthropics/agentsmesh/backend/internal/service/relay"
	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
	"github.com/anthropics/agentsmesh/backend/internal/service/ticket"
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

	// Create notification relay (Hub + Redis cross-instance push)
	notifRelay := websocket.NewNotificationRelay(hub, redisClient, appLogger.Logger)
	notifRelay.StartSubscriber(context.Background())

	// Create notification dispatcher and register resolvers
	notifDispatcher := notifService.NewDispatcher(notifRelay, services.notifPrefStore)
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
	runnerConnMgr, podCoordinator, podRouter, heartbeatBatcher, sandboxQuerySvc := initializeRunnerComponents(services.podRepo, services.runnerRepo, redisClient, appLogger, services.agentSvc)

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

	// Setup pod router event publishing
	podRouter.SetEventBus(eventBus)
	podRouter.SetPodInfoGetter(services.pod)

	// Route OSC terminal notifications through NotificationDispatcher (preference-aware)
	podRouter.SetNotifyFunc(func(ctx context.Context, orgID int64, source, entityID, title, body, link, resolver string) {
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
		// Record OSC notification to deduplicate against task:completed
		RecordOSCNotification(entityID)
	})

	// Create PodOrchestrator (unified Pod creation logic for REST + MCP paths)
	podOrchestrator := createPodOrchestrator(services, podCoordinator)

	// Wire PodPromptHook (must be after podRouter is initialized)
	services.channel.AddPostSendHook(channelService.NewPodPromptHook(podRouter, channelRepo))
	slog.Info("PodPromptHook registered with PodRouter")

	// Setup event callbacks
	setupRunnerEventCallbacks(db, runnerConnMgr, eventBus)
	setupPodEventCallbacks(db, podCoordinator, eventBus, notifDispatcher)
	startOSCDedupCleanup()

	// Wire AutopilotControllerService with PodCoordinator for gRPC command sending.
	// PodCoordinator implements AutopilotCommandSender (SendCreateAutopilot).
	services.autopilot.SetCommandSender(podCoordinator)

	// Initialize OrgAwarenessService (tracks which orgs this instance serves)
	runnerOrgQuerier := infra.NewRunnerOrgQuerier(db)
	orgAwareness := instance.NewOrgAwarenessService(runnerOrgQuerier, runnerConnMgr, redisClient, cfg.Server.Address, appLogger.Logger)
	orgAwareness.Start()
	setupOrgAwarenessRefresh(eventBus, orgAwareness)
	slog.Info("OrgAwarenessService started")

	// Initialize Loop orchestrator and scheduler
	loopOrchestrator := loop.NewLoopOrchestrator(services.loop, services.loopRun, eventBus, appLogger.Logger)
	loopOrchestrator.SetPodDependencies(podOrchestrator, services.autopilot, podCoordinator, services.ticket, services.repository)
	loopScheduler := loop.NewLoopScheduler(services.loop, loopOrchestrator, orgAwareness, appLogger.Logger)
	loopScheduler.Start()
	setupLoopEventSubscriptions(eventBus, loopOrchestrator)
	slog.Info("Loop orchestrator and scheduler created")

	// Initialize PKI, gRPC, and wire command senders
	grpcResult := initPKIAndGRPCWiring(cfg, services, runnerConnMgr, podCoordinator, podRouter, sandboxQuerySvc, podOrchestrator, loopOrchestrator, services.loopRun, appLogger, relayTokenGenerator, db)

	// Initialize Runner version checker (checks GitHub Releases for latest version)
	versionChecker := runner.NewVersionChecker(redisClient)
	if versionChecker != nil {
		versionChecker.Start(context.Background())
	}

	// Initialize runner log upload service (requires S3 storage)
	logUploadSvc := initLogUploadService(cfg, db, runnerConnMgr)

	// Build services container for REST handlers
	svc := buildServicesContainer(services, runnerConnMgr, podCoordinator, podOrchestrator, hub, eventBus,
		grpcResult, sandboxQuerySvc, logUploadSvc, relayManager, relayTokenGenerator, relayDNSService,
		relayACMEManager, geoResolver, versionChecker, loopOrchestrator, loopScheduler)

	// Initialize router
	router := rest.NewRouter(cfg, svc, db, appLogger.Logger, redisClient)

	// Start MarketplaceWorker if configured
	cleanupMkt := startMarketplaceWorker(services)
	defer cleanupMkt()

	// Start scheduled jobs
	subscriptionScheduler := startSubscriptionJobs(db, cfg, services.email, appLogger.Logger)

	// Start HTTP server
	srv := startHTTPServer(cfg, router)

	// Graceful shutdown
	waitForShutdown(srv, grpcResult.server, eventBus, heartbeatBatcher, subscriptionScheduler, loopScheduler, orgAwareness, relayManager, db, redisClient)
}

// Build trigger: 20260119003527
// rebuild trigger
