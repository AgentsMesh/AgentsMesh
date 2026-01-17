package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	grpcserver "github.com/anthropics/agentsmesh/backend/internal/api/grpc"
	"github.com/anthropics/agentsmesh/backend/internal/api/rest"
	v1 "github.com/anthropics/agentsmesh/backend/internal/api/rest/v1"
	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/infra/database"
	"github.com/anthropics/agentsmesh/backend/internal/infra/email"
	"github.com/anthropics/agentsmesh/backend/internal/infra/eventbus"
	"github.com/anthropics/agentsmesh/backend/internal/infra/logger"
	"github.com/anthropics/agentsmesh/backend/internal/infra/pki"
	"github.com/anthropics/agentsmesh/backend/internal/infra/storage"
	"github.com/anthropics/agentsmesh/backend/internal/infra/websocket"
	"github.com/anthropics/agentsmesh/backend/internal/job"
	"github.com/anthropics/agentsmesh/backend/internal/service/agent"
	"github.com/anthropics/agentsmesh/backend/internal/service/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/service/auth"
	"github.com/anthropics/agentsmesh/backend/internal/service/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/binding"
	"github.com/anthropics/agentsmesh/backend/internal/service/channel"
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

	// Set as default logger
	appLogger.SetDefault()
	slog.Info("Logger initialized", "level", cfg.Log.Level, "file", cfg.Log.FilePath)

	// Initialize database (supports automatic read-write separation when replicas are configured)
	db, err := database.New(cfg.Database)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Initialize services
	// Initialize infrastructure (Redis needed for auth service)
	hub, eventBus, redisClient := initializeInfrastructure(cfg, appLogger)

	services := initializeServices(cfg, db, redisClient)

	// Setup EventBus → Hub integration (decoupled via subscriptions)
	setupEventBusHub(eventBus, hub)

	// Setup ticket service event publishing (Service layer - Information Expert principle)
	ticketEventPublisher := ticket.NewEventBusPublisher(eventBus, appLogger.Logger)
	services.ticket.SetEventPublisher(ticketEventPublisher)

	// Setup pod service event publishing (Service layer - Information Expert principle)
	podEventPublisher := agentpod.NewEventBusPublisher(eventBus, appLogger.Logger)
	services.pod.SetEventPublisher(podEventPublisher)

	// Setup channel service event publishing for real-time message broadcast
	services.channel.SetEventBus(eventBus)

	// Start Redis subscriber for multi-instance sync (if Redis is available)
	if redisClient != nil {
		eventBus.StartRedisSubscriber(context.Background())
	}

	// Initialize Runner connection manager and Pod coordinator
	runnerConnMgr, podCoordinator, terminalRouter, heartbeatBatcher := initializeRunnerComponents(db, redisClient, appLogger, services.agentType)

	// Setup terminal router event publishing for OSC 777 notifications
	terminalRouter.SetEventBus(eventBus)
	terminalRouter.SetPodInfoGetter(services.pod)

	// Setup event callbacks for runner and pod status changes
	setupRunnerEventCallbacks(db, runnerConnMgr, eventBus)
	setupPodEventCallbacks(db, podCoordinator, eventBus)

	// Initialize PKI service and gRPC Server for mTLS Runner communication
	// Automatically enabled when PKI_CA_CERT_FILE and PKI_CA_KEY_FILE are configured
	var grpcRunnerHandler *v1.GRPCRunnerHandler
	var grpcServer *grpcserver.Server
	if cfg.PKI.CACertFile != "" && cfg.PKI.CAKeyFile != "" {
		grpcServer, grpcRunnerHandler = initializePKIAndGRPC(cfg, services.runner, services.org, services.agentType, runnerConnMgr, appLogger)

		// Connect gRPC Server to PodCoordinator and TerminalRouter for sending commands to runners
		if grpcServer != nil {
			grpcCommandSender := grpcserver.NewGRPCCommandSender(grpcServer.RunnerAdapter())
			podCoordinator.SetCommandSender(grpcCommandSender)
			terminalRouter.SetCommandSender(grpcCommandSender)
			slog.Info("PodCoordinator and TerminalRouter connected to gRPC Server for Runner commands")
		}
	} else {
		slog.Warn("PKI CA files not configured, gRPC/mTLS Runner communication disabled",
			"hint", "Set PKI_CA_CERT_FILE and PKI_CA_KEY_FILE to enable")
	}

	// Create services container for HTTP handlers
	// NOTE: GitProvider and SSHKey services removed - now handled via user.Service
	svc := &v1.Services{
		Auth:              services.auth,
		User:              services.user,
		Org:               services.org,
		AgentType:         services.agentType,
		CredentialProfile: services.credentialProfile,
		UserConfig:        services.userConfig,
		Repository:        services.repository,
		Runner:            services.runner,
		RunnerConnMgr:     runnerConnMgr,
		PodCoordinator:    podCoordinator,
		TerminalRouter:    terminalRouter,
		Pod:               services.pod,
		Channel:           services.channel,
		Binding:           services.binding,
		Ticket:            services.ticket,
		Mesh:              services.mesh,
		Billing:           services.billing,
		Hub:               hub,
		EventBus:          eventBus,
		Invitation:        services.invitation,
		File:              services.file,
		PromoCode:          services.promoCode,
		AgentPodSettings:   services.agentpodSettings,
		AgentPodAIProvider: services.agentpodAIProvider,
		License:            services.license,
		GRPCRunnerHandler:  grpcRunnerHandler,
	}

	// Initialize router
	router := rest.NewRouter(cfg, svc, db, appLogger.Logger)

	// Initialize and start subscription renewal job scheduler
	subscriptionScheduler := startSubscriptionJobs(db, &cfg.Payment, services.email, appLogger.Logger)

	// Create and start HTTP server
	srv := startHTTPServer(cfg, router)

	// Graceful shutdown
	waitForShutdown(srv, grpcServer, eventBus, heartbeatBatcher, subscriptionScheduler, db, redisClient)
}

// serviceContainer holds all initialized services
type serviceContainer struct {
	auth               *auth.Service
	user               *user.Service
	org                *organization.Service
	// Agent services (split by responsibility per SRP)
	agentType          *agent.AgentTypeService
	credentialProfile  *agent.CredentialProfileService
	userConfig         *agent.UserConfigService
	repository         *repository.Service
	runner             *runner.Service
	pod                *agentpod.PodService
	channel            *channel.Service
	ticket             *ticket.Service
	billing            *billing.Service
	binding            *binding.Service
	mesh               *mesh.Service
	invitation         *invitation.Service
	file               *fileservice.Service
	promoCode          *promocode.Service
	agentpodSettings   *agentpod.SettingsService
	agentpodAIProvider *agentpod.AIProviderService
	license            *license.Service
	email              email.Service
	// NOTE: gitProvider and sshKey removed - now handled via user.Service
}

// initializeServices creates all business services
func initializeServices(cfg *config.Config, db *gorm.DB, redisClient *redis.Client) *serviceContainer {
	// Use JWT secret as encryption key for token encryption (OAuth tokens, etc.)
	userSvc := user.NewServiceWithEncryption(db, cfg.JWT.Secret)
	authCfg := &auth.Config{
		JWTSecret:         cfg.JWT.Secret,
		JWTExpiration:     time.Duration(cfg.JWT.ExpirationHours) * time.Hour,
		RefreshExpiration: time.Duration(cfg.JWT.ExpirationHours*7) * time.Hour, // 7x access token
		Issuer: "agentsmesh",
	}
	authSvc := auth.NewServiceWithRedis(authCfg, userSvc, redisClient)
	orgSvc := organization.NewService(db)
	// Initialize agent sub-services (split by responsibility per SRP)
	agentTypeSvc := agent.NewAgentTypeService(db)
	credentialProfileSvc := agent.NewCredentialProfileService(db, agentTypeSvc)
	userConfigSvc := agent.NewUserConfigService(db, agentTypeSvc)
	repoSvc := repository.NewService(db)
	billingSvc := billing.NewServiceWithConfig(db, &cfg.Payment)
	runnerSvc := runner.NewService(db, billingSvc)
	podSvc := agentpod.NewPodService(db)
	channelSvc := channel.NewService(db)
	ticketSvc := ticket.NewService(db)
	bindingSvc := binding.NewService(db, podSvc)
	meshSvc := mesh.NewService(db, podSvc, channelSvc, bindingSvc)

	// Initialize email service for invitations
	emailSvc := email.NewService(email.Config{
		Provider:    cfg.Email.Provider,
		ResendKey:   cfg.Email.ResendKey,
		FromAddress: cfg.Email.FromAddress,
		BaseURL:     cfg.Email.BaseURL,
	})
	invitationSvc := invitation.NewService(db, emailSvc)

	// Initialize promo code service
	promoCodeSvc := promocode.NewService(db)

	// Initialize AgentPod settings and AI provider services
	agentpodSettingsSvc := agentpod.NewSettingsService(db)
	encryptor := crypto.NewEncryptor(cfg.JWT.Secret)
	agentpodAIProviderSvc := agentpod.NewAIProviderService(db, encryptor)

	// Initialize storage (S3-compatible)
	var fileSvc *fileservice.Service
	if cfg.Storage.AccessKey != "" && cfg.Storage.SecretKey != "" {
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
		} else {
			// Ensure bucket exists
			if err := s3Storage.EnsureBucket(context.Background()); err != nil {
				slog.Warn("Failed to ensure bucket exists", "bucket", cfg.Storage.Bucket, "error", err)
			}
			fileSvc = fileservice.NewService(db, s3Storage, cfg.Storage)
			slog.Info("Storage initialized", "endpoint", cfg.Storage.Endpoint, "bucket", cfg.Storage.Bucket)
		}
	} else {
		slog.Warn("Storage not configured, file upload disabled")
	}

	// Initialize license service (for OnPremise deployments)
	var licenseSvc *license.Service
	if cfg.Payment.IsOnPremise() || cfg.Payment.License.PublicKeyPath != "" {
		var err error
		licenseSvc, err = license.NewService(db, &cfg.Payment.License, slog.Default())
		if err != nil {
			slog.Warn("Failed to initialize license service", "error", err)
		} else {
			slog.Info("License service initialized")
		}
	}

	return &serviceContainer{
		auth:               authSvc,
		user:               userSvc,
		org:                orgSvc,
		agentType:          agentTypeSvc,
		credentialProfile:  credentialProfileSvc,
		userConfig:         userConfigSvc,
		repository:         repoSvc,
		runner:             runnerSvc,
		pod:                podSvc,
		channel:            channelSvc,
		ticket:             ticketSvc,
		billing:            billingSvc,
		binding:            bindingSvc,
		mesh:               meshSvc,
		invitation:         invitationSvc,
		file:               fileSvc,
		promoCode:          promoCodeSvc,
		agentpodSettings:   agentpodSettingsSvc,
		agentpodAIProvider: agentpodAIProviderSvc,
		license:            licenseSvc,
		email:              emailSvc,
	}
}

// initializeInfrastructure initializes WebSocket hub, EventBus, and Redis
func initializeInfrastructure(cfg *config.Config, appLogger *logger.Logger) (*websocket.Hub, *eventbus.EventBus, *redis.Client) {
	// Initialize WebSocket hub
	hub := websocket.NewHub()
	go hub.Run()

	// Initialize Redis client (optional, for multi-instance event sync)
	var redisClient *redis.Client
	if cfg.Redis.URL != "" {
		opt, err := redis.ParseURL(cfg.Redis.URL)
		if err != nil {
			slog.Warn("Failed to parse Redis URL, skipping Redis", "error", err)
		} else {
			redisClient = redis.NewClient(opt)
			if err := redisClient.Ping(context.Background()).Err(); err != nil {
				slog.Warn("Failed to connect to Redis, events will be local only", "error", err)
				redisClient = nil
			} else {
				slog.Info("Redis connected", "url", cfg.Redis.URL)
			}
		}
	} else if cfg.Redis.Host != "" {
		redisClient = redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
		})
		if err := redisClient.Ping(context.Background()).Err(); err != nil {
			slog.Warn("Failed to connect to Redis, events will be local only", "error", err)
			redisClient = nil
		} else {
			slog.Info("Redis connected", "host", cfg.Redis.Host, "port", cfg.Redis.Port)
		}
	}

	// Initialize EventBus for real-time events
	eventBus := eventbus.NewEventBus(redisClient, appLogger.Logger)

	return hub, eventBus, redisClient
}

// initializeRunnerComponents initializes runner-related components
func initializeRunnerComponents(db *gorm.DB, redisClient *redis.Client, appLogger *logger.Logger, agentTypeSvc *agent.AgentTypeService) (*runner.RunnerConnectionManager, *runner.PodCoordinator, *runner.TerminalRouter, *runner.HeartbeatBatcher) {
	// Initialize Runner connection manager
	runnerConnMgr := runner.NewRunnerConnectionManager(appLogger.Logger)

	// Setup AgentTypesProvider for initialization handshake
	agentTypesAdapter := runner.NewAgentTypeServiceAdapter(agentTypeSvc)
	runnerConnMgr.SetAgentTypesProvider(agentTypesAdapter)
	runnerConnMgr.SetServerVersion("1.0.0") // TODO: Get from build info

	// Start initialization timeout checker (removes connections that don't complete handshake)
	runnerConnMgr.StartInitTimeoutChecker()

	// Initialize Terminal router (routes terminal data between frontend and runner)
	terminalRouter := runner.NewTerminalRouter(runnerConnMgr, appLogger.Logger)

	// Initialize Heartbeat batcher (batches heartbeat DB writes for high-scale performance)
	heartbeatBatcher := runner.NewHeartbeatBatcher(redisClient, db, appLogger.Logger)
	heartbeatBatcher.Start()

	// Initialize Pod coordinator (manages pod lifecycle between backend and runner)
	podCoordinator := runner.NewPodCoordinator(db, runnerConnMgr, terminalRouter, heartbeatBatcher, appLogger.Logger)

	return runnerConnMgr, podCoordinator, terminalRouter, heartbeatBatcher
}

// startHTTPServer creates and starts the HTTP server
func startHTTPServer(cfg *config.Config, handler http.Handler) *http.Server {
	srv := &http.Server{
		Addr:         cfg.Server.Address,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("Starting server", "address", cfg.Server.Address)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Failed to start server", "error", err)
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	return srv
}

// startSubscriptionJobs initializes and starts subscription-related scheduled jobs
func startSubscriptionJobs(db *gorm.DB, paymentCfg *config.PaymentConfig, emailSvc email.Service, logger *slog.Logger) *job.SubscriptionScheduler {
	scheduler := job.NewSubscriptionScheduler(db, paymentCfg, emailSvc, logger)
	scheduler.Start()
	slog.Info("subscription scheduler started")
	return scheduler
}

// waitForShutdown handles graceful shutdown
func waitForShutdown(srv *http.Server, grpcServer *grpcserver.Server, eventBus *eventbus.EventBus, heartbeatBatcher *runner.HeartbeatBatcher, subscriptionScheduler *job.SubscriptionScheduler, db *gorm.DB, redisClient *redis.Client) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	// Stop gRPC server
	if grpcServer != nil {
		grpcServer.Stop()
	}

	// Stop subscription scheduler
	if subscriptionScheduler != nil {
		subscriptionScheduler.Stop()
	}

	// Stop heartbeat batcher (flush pending writes)
	if heartbeatBatcher != nil {
		heartbeatBatcher.Stop()
	}

	// Close EventBus
	eventBus.Close()

	// Close database connection
	if db != nil {
		if err := database.Close(db); err != nil {
			slog.Error("Failed to close database connection", "error", err)
		}
	}

	// Close Redis connection
	if redisClient != nil {
		redisClient.Close()
	}

	slog.Info("Server exited")
}

// initializePKIAndGRPC initializes PKI service, gRPC Server, and gRPC runner handler.
// Returns nil values if initialization fails.
// Requires: PKI_CA_CERT_FILE, PKI_CA_KEY_FILE (CA certificate and key files)
// Optional: GRPC_ADDRESS (default :9090), GRPC_ENDPOINT (public endpoint for runners)
func initializePKIAndGRPC(cfg *config.Config, runnerSvc *runner.Service, orgSvc *organization.Service, agentTypeSvc *agent.AgentTypeService, runnerConnMgr *runner.RunnerConnectionManager, appLogger *logger.Logger) (*grpcserver.Server, *v1.GRPCRunnerHandler) {

	// Initialize PKI service
	pkiService, err := pki.NewService(&pki.Config{
		CACertFile:     cfg.PKI.CACertFile,
		CAKeyFile:      cfg.PKI.CAKeyFile,
		ServerCertFile: cfg.PKI.ServerCertFile,
		ServerKeyFile:  cfg.PKI.ServerKeyFile,
		ValidityDays:   cfg.PKI.ValidityDays,
	})
	if err != nil {
		slog.Error("Failed to initialize PKI service", "error", err)
		slog.Warn("Continuing without gRPC/mTLS support")
		return nil, nil
	}

	slog.Info("PKI service initialized",
		"ca_cert", cfg.PKI.CACertFile,
		"validity_days", cfg.PKI.ValidityDays,
	)

	// Create gRPC runner handler for REST API (certificate issuance/renewal)
	grpcRunnerHandler := v1.NewGRPCRunnerHandler(runnerSvc, pkiService, cfg)

	// Create service adapters for gRPC server
	runnerServiceAdapter := &grpcRunnerServiceAdapter{svc: runnerSvc}
	orgServiceAdapter := &grpcOrgServiceAdapter{svc: orgSvc}
	agentTypesAdapter := &grpcAgentTypesAdapter{svc: agentTypeSvc}

	// Create and start gRPC server
	grpcServerInst, err := grpcserver.NewServer(&grpcserver.ServerDependencies{
		Logger:             appLogger.Logger,
		Config:             &cfg.GRPC,
		PKIService:         pkiService,
		RunnerService:      runnerServiceAdapter,
		OrgService:         orgServiceAdapter,
		AgentTypesProvider: agentTypesAdapter,
		ConnManager:        runnerConnMgr,
	})
	if err != nil {
		slog.Error("Failed to create gRPC server", "error", err)
		slog.Warn("Continuing without gRPC server")
		return nil, grpcRunnerHandler
	}

	// Start gRPC server
	if err := grpcServerInst.Start(); err != nil {
		slog.Error("Failed to start gRPC server", "error", err)
		slog.Warn("Continuing without gRPC server")
		return nil, grpcRunnerHandler
	}

	slog.Info("gRPC/mTLS Runner communication enabled",
		"grpc_address", cfg.GRPC.Address,
	)

	return grpcServerInst, grpcRunnerHandler
}

// grpcRunnerServiceAdapter adapts runner.Service to grpcserver.RunnerServiceInterface
type grpcRunnerServiceAdapter struct {
	svc *runner.Service
}

func (a *grpcRunnerServiceAdapter) GetByNodeID(ctx context.Context, nodeID string) (grpcserver.RunnerInfo, error) {
	r, err := a.svc.GetByNodeID(ctx, nodeID)
	if err != nil {
		return grpcserver.RunnerInfo{}, err
	}
	certSerial := ""
	if r.CertSerialNumber != nil {
		certSerial = *r.CertSerialNumber
	}
	return grpcserver.RunnerInfo{
		ID:               r.ID,
		NodeID:           r.NodeID,
		OrganizationID:   r.OrganizationID,
		IsEnabled:        r.IsEnabled,
		CertSerialNumber: certSerial,
	}, nil
}

func (a *grpcRunnerServiceAdapter) UpdateLastSeen(ctx context.Context, runnerID int64) error {
	return a.svc.UpdateLastSeen(ctx, runnerID)
}

func (a *grpcRunnerServiceAdapter) UpdateAvailableAgents(ctx context.Context, runnerID int64, agents []string) error {
	return a.svc.UpdateAvailableAgents(ctx, runnerID, agents)
}

func (a *grpcRunnerServiceAdapter) IsCertificateRevoked(ctx context.Context, serialNumber string) (bool, error) {
	return a.svc.IsCertificateRevoked(ctx, serialNumber)
}

// grpcOrgServiceAdapter adapts organization.Service to grpcserver.OrganizationServiceInterface
type grpcOrgServiceAdapter struct {
	svc *organization.Service
}

func (a *grpcOrgServiceAdapter) GetBySlug(ctx context.Context, slug string) (grpcserver.OrganizationInfo, error) {
	org, err := a.svc.GetOrgBySlug(ctx, slug)
	if err != nil {
		return grpcserver.OrganizationInfo{}, err
	}
	return grpcserver.OrganizationInfo{
		ID:   org.ID,
		Slug: org.Slug,
	}, nil
}

// grpcAgentTypesAdapter adapts agent.AgentTypeService to grpcserver.AgentTypesProvider
type grpcAgentTypesAdapter struct {
	svc *agent.AgentTypeService
}

func (a *grpcAgentTypesAdapter) GetAgentTypesForRunner() []grpcserver.AgentTypeInfo {
	types := a.svc.GetAgentTypesForRunner()
	result := make([]grpcserver.AgentTypeInfo, len(types))
	for i, t := range types {
		result[i] = grpcserver.AgentTypeInfo{
			Slug:          t.Slug,
			Name:          t.Name,
			Executable:    t.Executable,
			LaunchCommand: t.LaunchCommand,
		}
	}
	return result
}
