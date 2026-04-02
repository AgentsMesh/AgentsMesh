// AgentsMesh Backend Server - PKI/gRPC initialization wiring, log upload, and final startup.
package main

import (
	"context"
	"log/slog"

	grpcserver "github.com/anthropics/agentsmesh/backend/internal/api/grpc"
	v1 "github.com/anthropics/agentsmesh/backend/internal/api/rest/v1"
	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/infra/acme"
	"github.com/anthropics/agentsmesh/backend/internal/infra/eventbus"
	"github.com/anthropics/agentsmesh/backend/internal/infra/logger"
	"github.com/anthropics/agentsmesh/backend/internal/infra/websocket"
	"github.com/anthropics/agentsmesh/backend/internal/service/agent"
	"github.com/anthropics/agentsmesh/backend/internal/service/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/service/geo"
	loop "github.com/anthropics/agentsmesh/backend/internal/service/loop"
	"github.com/anthropics/agentsmesh/backend/internal/service/relay"
	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
	runnerlogservice "github.com/anthropics/agentsmesh/backend/internal/service/runnerlog"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"gorm.io/gorm"
)

// grpcWiringResult holds all outputs from initPKIAndGRPCWiring.
type grpcWiringResult struct {
	handler              *v1.GRPCRunnerHandler
	server               *grpcserver.Server
	upgradeCommandSender runner.UpgradeCommandSender
	logUploadSender      runner.LogUploadCommandSender
}

// initPKIAndGRPCWiring initializes PKI/gRPC and wires command senders to coordinator and router.
func initPKIAndGRPCWiring(
	cfg *config.Config,
	services *serviceContainer,
	runnerConnMgr *runner.RunnerConnectionManager,
	podCoordinator *runner.PodCoordinator,
	podRouter *runner.PodRouter,
	sandboxQuerySvc *runner.SandboxQueryService,
	podOrchestrator *agentpod.PodOrchestrator,
	loopOrchestrator *loop.LoopOrchestrator,
	loopRunSvc *loop.LoopRunService,
	appLogger *logger.Logger,
	relayTokenGenerator *relay.TokenGenerator,
	db *gorm.DB,
) *grpcWiringResult {
	if cfg.PKI.CACertFile == "" || cfg.PKI.CAKeyFile == "" {
		slog.Warn("PKI CA files not configured, gRPC/mTLS disabled")
		// Create handler without PKI so token management routes still work
		return &grpcWiringResult{
			handler: v1.NewGRPCRunnerHandler(services.runner, nil, cfg),
		}
	}

	mcpDeps := &grpcserver.MCPDependencies{
		PodService:        services.pod,
		PodOrchestrator:   podOrchestrator,
		ChannelService:    services.channel,
		BindingService:    services.binding,
		TicketService:     services.ticket,
		RepositoryService: services.repository,
		RunnerService:     services.runner,
		AgentSvc:          services.agentSvc,
		UserConfigSvc:     services.userConfig,
		PodRouter:         podRouter,
		LoopService:       services.loop,
		LoopRunService:    services.loopRun,
		LoopOrchestrator:  loopOrchestrator,
	}
	grpcServer, grpcRunnerHandler := initializePKIAndGRPC(cfg, services.runner, services.org, services.agentSvc, runnerConnMgr, appLogger, mcpDeps)

	result := &grpcWiringResult{handler: grpcRunnerHandler, server: grpcServer}
	if grpcServer != nil {
		grpcCommandSender := grpcserver.NewGRPCCommandSender(grpcServer.RunnerAdapter())
		podCoordinator.SetCommandSender(grpcCommandSender)
		podRouter.SetCommandSender(grpcCommandSender)
		sandboxQuerySvc.SetSender(grpcCommandSender)
		result.upgradeCommandSender = grpcCommandSender
		result.logUploadSender = grpcCommandSender
		slog.Info("PodCoordinator and PodRouter connected to gRPC Server")
		setupRelayTokenRefreshCallback(db, runnerConnMgr, relayTokenGenerator, grpcCommandSender)
	}
	return result
}

// initLogUploadService initializes the runner log upload service if S3 storage is configured.
func initLogUploadService(
	cfg *config.Config,
	db *gorm.DB,
	runnerConnMgr *runner.RunnerConnectionManager,
) *runnerlogservice.Service {
	if cfg.Storage.AccessKey == "" || cfg.Storage.SecretKey == "" {
		slog.Info("Runner log upload service disabled: storage not configured")
		return nil
	}
	logUploadStorage := initializeLogUploadStorage(cfg)
	if logUploadStorage == nil {
		return nil
	}
	logUploadRepo := infra.NewRunnerLogRepository(db)
	logUploadSvc := runnerlogservice.NewService(logUploadRepo, logUploadStorage)

	// Register callback for log upload status events from Runner
	runnerConnMgr.SetLogUploadStatusCallback(func(runnerID int64, data *runnerv1.LogUploadStatusEvent) {
		logUploadSvc.HandleUploadStatus(runnerID, data.RequestId, data.Phase, data.Progress, data.Message, data.Error, data.SizeBytes)
	})
	slog.Info("Runner log upload service initialized")
	return logUploadSvc
}

// createPodOrchestrator creates the unified PodOrchestrator for REST + MCP paths.
func createPodOrchestrator(services *serviceContainer, podCoordinator *runner.PodCoordinator) *agentpod.PodOrchestrator {
	compositeProvider := agent.NewCompositeProvider(services.agentSvc, services.credentialProfile, services.userConfig)
	configBuilder := agent.NewConfigBuilder(compositeProvider)
	if services.extension != nil {
		configBuilder.SetExtensionProvider(services.extension)
		slog.Info("ExtensionProvider connected to ConfigBuilder")
	}
	orch := agentpod.NewPodOrchestrator(&agentpod.PodOrchestratorDeps{
		PodService:      services.pod,
		ConfigBuilder:   configBuilder,
		PodCoordinator:  podCoordinator,
		BillingService:  services.billing,
		UserService:     services.user,
		RepoService:     services.repository,
		TicketService:   services.ticket,
		RunnerSelector:  services.runner,
		AgentResolver:   services.agentSvc,
		RunnerQuery:     services.runner,
		UserConfigQuery: services.userConfig,
	})
	slog.Info("PodOrchestrator created")
	return orch
}

// buildServicesContainer constructs the v1.Services container from all initialized components.
func buildServicesContainer(
	services *serviceContainer,
	runnerConnMgr *runner.RunnerConnectionManager,
	podCoordinator *runner.PodCoordinator,
	podOrchestrator *agentpod.PodOrchestrator,
	hub *websocket.Hub,
	eventBus *eventbus.EventBus,
	grpcResult *grpcWiringResult,
	sandboxQuerySvc *runner.SandboxQueryService,
	logUploadSvc *runnerlogservice.Service,
	relayManager *relay.Manager,
	relayTokenGenerator *relay.TokenGenerator,
	relayDNSService *relay.DNSService,
	relayACMEManager *acme.Manager,
	geoResolver geo.Resolver,
	versionChecker *runner.VersionChecker,
	loopOrchestrator *loop.LoopOrchestrator,
	loopScheduler *loop.LoopScheduler,
) *v1.Services {
	return &v1.Services{
		Auth:               services.auth,
		User:               services.user,
		Org:                services.org,
		AgentSvc:           services.agentSvc,
		CredentialProfile:  services.credentialProfile,
		UserConfig:         services.userConfig,
		Repository:         services.repository,
		Webhook:            services.webhook,
		Runner:             services.runner,
		RunnerConnMgr:      runnerConnMgr,
		PodCoordinator:     podCoordinator,
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
		GRPCRunnerHandler:    grpcResult.handler,
		SandboxQueryService:  sandboxQuerySvc,
		UpgradeCommandSender: grpcResult.upgradeCommandSender,
		LogUploadSender:      grpcResult.logUploadSender,
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
}

// startMarketplaceWorker starts the marketplace worker and returns a cleanup func.
func startMarketplaceWorker(services *serviceContainer) func() {
	if services.marketplaceWorker != nil {
		services.marketplaceWorker.Start(context.Background())
		slog.Info("MarketplaceWorker started")
	}
	return func() {
		if services.marketplaceWorker != nil {
			services.marketplaceWorker.Stop()
			slog.Info("MarketplaceWorker stopped")
		}
	}
}
