package main

import (
	"context"
	"log"
	"log/slog"
	"os"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	notifDomain "github.com/anthropics/agentsmesh/backend/internal/domain/notification"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	blockstoreinfra "github.com/anthropics/agentsmesh/backend/internal/infra/blockstore"
	"github.com/anthropics/agentsmesh/backend/internal/infra/database"
	"github.com/anthropics/agentsmesh/backend/internal/infra/logger"
	otelinit "github.com/anthropics/agentsmesh/backend/internal/infra/otel"
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
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		runMigrate(os.Args[2:])
		return
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	cfg.WarnInsecureDefaults()

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

	otelProvider, err := otelinit.InitProvider(context.Background(), "agentsmesh-backend", "1.0.0")
	if err != nil {
		slog.Warn("OpenTelemetry initialization failed, continuing without tracing", "error", err)
		otelProvider = &otelinit.Provider{}
	}
	defer otelProvider.Shutdown(context.Background())

	slog.SetDefault(slog.New(otelinit.NewTraceContextHandler(slog.Default().Handler())))

	db, err := database.New(cfg.Database)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		log.Fatalf("Failed to connect to database: %v", err)
	}

	hub, eventBus, redisClient := initializeInfrastructure(cfg, appLogger)
	services := initializeServices(cfg, db, redisClient)

	setupEventBusHub(eventBus, hub)

	ticketEventPublisher := ticket.NewEventBusPublisher(eventBus, appLogger.Logger)
	services.ticket.SetEventPublisher(ticketEventPublisher)
	podEventPublisher := agentpod.NewEventBusPublisher(eventBus, appLogger.Logger)
	services.pod.SetEventPublisher(podEventPublisher)
	services.channel.SetEventBus(eventBus)
	services.channel.SetPodCreatorResolver(services.pod)
	services.blockstore.SetPublisher(blockstoreinfra.NewOpPublisher(eventBus))

	notifRelay := websocket.NewNotificationRelay(hub, redisClient, appLogger.Logger)
	notifRelay.StartSubscriber(context.Background())

	notifDispatcher := notifService.NewDispatcher(notifRelay, services.notifPrefStore)
	notifDispatcher.RegisterResolver("pod_creator", notifService.NewPodCreatorResolver(services.pod))
	notifDispatcher.RegisterResolver("channel_members", notifService.NewChannelMemberResolver(services.channel))
	services.notifDispatcher = notifDispatcher

	channelRepo := infra.NewChannelRepository(db)
	userLookup := infra.NewChannelUserLookup(db)
	podLookup := infra.NewChannelPodLookup(db)
	channelUserNames := infra.NewChannelUserNameResolver(db)
	services.channel.SetUserLookup(userLookup)
	services.channel.AddPostSendHook(channelService.NewMentionValidatorHook(userLookup, podLookup, channelRepo))
	services.channel.AddPostSendHook(channelService.NewEventPublishHook(eventBus, channelUserNames, services.channel))
	services.channel.AddPostSendHook(channelService.NewNotificationHook(notifDispatcher, channelUserNames))
	slog.Info("Channel PostSendHooks registered")

	services.user.AddPreDeleteHook(func(ctx context.Context, userID int64) error {
		return services.channel.CleanupUserReferences(ctx, userID)
	})

	if redisClient != nil {
		eventBus.StartRedisSubscriber(context.Background())
	}

	runnerConnMgr, podCoordinator, podRouter, heartbeatBatcher, sandboxQuerySvc := initializeRunnerComponents(services.pod, services.runnerRepo, redisClient, appLogger, services.agentSvc)

	podCoordinator.SetAutopilotRepo(services.autopilotRepo)

	podCoordinator.SetTokenUsageService(services.tokenUsage)

	relayManager := relay.NewManagerWithOptions()
	relayTokenGenerator := relay.NewTokenGenerator(cfg.JWT.Secret, "agentsmesh-relay")
	relayDNSService, relayACMEManager := initializeRelayServices(cfg)
	slog.Info("Relay services initialized")

	geoResolver := initializeGeoResolver()
	defer geoResolver.Close()

	podRouter.SetEventBus(eventBus)
	podRouter.SetPodInfoGetter(services.pod)

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
		RecordOSCNotification(entityID)
	})

	podOrchestrator := createPodOrchestrator(services, podCoordinator)

	services.channel.AddPostSendHook(channelService.NewPodPromptHook(podRouter, channelRepo))
	slog.Info("PodPromptHook registered with PodRouter")

	setupRunnerEventCallbacks(db, runnerConnMgr, eventBus)
	setupPodEventCallbacks(db, podCoordinator, eventBus, notifDispatcher)
	setupPerpetualPodCallbacks(db, podCoordinator, eventBus)
	startOSCDedupCleanup()

	services.autopilot.SetCommandSender(podCoordinator)

	runnerOrgQuerier := infra.NewRunnerOrgQuerier(db)
	orgAwareness := instance.NewOrgAwarenessService(runnerOrgQuerier, runnerConnMgr, redisClient, cfg.Server.Address, appLogger.Logger)
	orgAwareness.Start()
	setupOrgAwarenessRefresh(eventBus, orgAwareness)
	slog.Info("OrgAwarenessService started")

	loopOrchestrator := loop.NewLoopOrchestrator(services.loop, services.loopRun, eventBus, appLogger.Logger)
	loopOrchestrator.SetPodDependencies(podOrchestrator, services.autopilot, podCoordinator, services.ticket, services.repository)
	loopScheduler := loop.NewLoopScheduler(services.loop, loopOrchestrator, orgAwareness, appLogger.Logger)
	loopScheduler.Start()
	setupLoopEventSubscriptions(eventBus, loopOrchestrator)
	slog.Info("Loop orchestrator and scheduler created")

	grpcResult := initPKIAndGRPCWiring(cfg, services, runnerConnMgr, podCoordinator, podRouter, sandboxQuerySvc, podOrchestrator, loopOrchestrator, services.loopRun, appLogger, relayTokenGenerator, db)

	versionChecker := runner.NewVersionChecker(redisClient)
	if versionChecker != nil {
		versionChecker.Start(context.Background())
	}

	logUploadSvc := initLogUploadService(cfg, db, runnerConnMgr)

	svc := buildServicesContainer(services, runnerConnMgr, podCoordinator, podOrchestrator, hub, eventBus,
		grpcResult, sandboxQuerySvc, logUploadSvc, relayManager, relayTokenGenerator, relayDNSService,
		relayACMEManager, geoResolver, versionChecker, loopOrchestrator, loopScheduler, redisClient)

	router := rest.NewRouter(cfg, svc, db, appLogger.Logger, redisClient)

	cleanupMkt := startMarketplaceWorker(services)
	defer cleanupMkt()

	subscriptionScheduler := startSubscriptionJobs(db, cfg, services.email, appLogger.Logger)

	srv := startHTTPServer(cfg, router)

	waitForShutdown(srv, grpcResult.server, eventBus, heartbeatBatcher, subscriptionScheduler, loopScheduler, orgAwareness, relayManager, services, db, redisClient)
}
