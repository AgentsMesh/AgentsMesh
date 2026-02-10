// AgentsMesh Backend Server
// Build version marker: 2026-02-06-fix-webhook-api-errors
package main

import (
	"context"
	"log"
	"log/slog"

	grpcserver "github.com/anthropics/agentsmesh/backend/internal/api/grpc"
	"github.com/anthropics/agentsmesh/backend/internal/api/rest"
	v1 "github.com/anthropics/agentsmesh/backend/internal/api/rest/v1"
	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/infra/acme"
	"github.com/anthropics/agentsmesh/backend/internal/infra/database"
	"github.com/anthropics/agentsmesh/backend/internal/infra/logger"
	"github.com/anthropics/agentsmesh/backend/internal/service/agent"
	"github.com/anthropics/agentsmesh/backend/internal/service/agentpod"
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

	// Start Redis subscriber for multi-instance sync
	if redisClient != nil {
		eventBus.StartRedisSubscriber(context.Background())
	}

	// Initialize Runner components
	runnerConnMgr, podCoordinator, terminalRouter, heartbeatBatcher, sandboxQuerySvc := initializeRunnerComponents(db, redisClient, appLogger, services.agentType)

	// Initialize Relay services
	relayManager := relay.NewManager()
	relayTokenGenerator := relay.NewTokenGenerator(cfg.JWT.Secret, "agentsmesh-relay")
	relayDNSService, relayACMEManager := initializeRelayServices(cfg)
	slog.Info("Relay services initialized")

	// Setup terminal router event publishing
	terminalRouter.SetEventBus(eventBus)
	terminalRouter.SetPodInfoGetter(services.pod)

	// Setup event callbacks
	setupRunnerEventCallbacks(db, runnerConnMgr, eventBus)
	setupPodEventCallbacks(db, podCoordinator, eventBus)

	// Create PodOrchestrator (unified Pod creation logic for REST + MCP paths)
	compositeProvider := agent.NewCompositeProvider(services.agentType, services.credentialProfile, services.userConfig)
	configBuilder := agent.NewConfigBuilder(compositeProvider)
	podOrchestrator := agentpod.NewPodOrchestrator(&agentpod.PodOrchestratorDeps{
		PodService:     services.pod,
		ConfigBuilder:  configBuilder,
		PodCoordinator: podCoordinator,
		BillingService: services.billing,
		UserService:    services.user,
		RepoService:    services.repository,
		TicketService:  services.ticket,
	})
	slog.Info("PodOrchestrator created")

	// Initialize PKI and gRPC
	var grpcRunnerHandler *v1.GRPCRunnerHandler
	var grpcServer *grpcserver.Server
	var sandboxQuerySender runner.SandboxQuerySender
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
			slog.Info("PodCoordinator and TerminalRouter connected to gRPC Server")
			setupRelayTokenRefreshCallback(db, runnerConnMgr, relayTokenGenerator, grpcCommandSender)
		}
	} else {
		slog.Warn("PKI CA files not configured, gRPC/mTLS disabled")
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
		GRPCRunnerHandler:  grpcRunnerHandler,
		SandboxQueryService: sandboxQuerySvc,
		SandboxQuerySender:  sandboxQuerySender,
		RelayManager:        relayManager,
		RelayTokenGenerator: relayTokenGenerator,
		RelayDNSService:     relayDNSService,
		RelayACMEManager:    relayACMEManager,
	}

	// Initialize router
	router := rest.NewRouter(cfg, svc, db, appLogger.Logger)

	// Start scheduled jobs
	subscriptionScheduler := startSubscriptionJobs(db, cfg, services.email, appLogger.Logger)

	// Start HTTP server
	srv := startHTTPServer(cfg, router)

	// Graceful shutdown
	waitForShutdown(srv, grpcServer, eventBus, heartbeatBatcher, subscriptionScheduler, db, redisClient)
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
