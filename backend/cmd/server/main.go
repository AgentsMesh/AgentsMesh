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

	"github.com/anthropics/agentmesh/backend/internal/api/rest"
	v1 "github.com/anthropics/agentmesh/backend/internal/api/rest/v1"
	"github.com/anthropics/agentmesh/backend/internal/config"
	"github.com/anthropics/agentmesh/backend/internal/infra/database"
	"gorm.io/gorm"
	"github.com/anthropics/agentmesh/backend/internal/infra/email"
	"github.com/anthropics/agentmesh/backend/internal/infra/eventbus"
	"github.com/anthropics/agentmesh/backend/internal/infra/logger"
	"github.com/anthropics/agentmesh/backend/internal/infra/storage"
	"github.com/anthropics/agentmesh/backend/internal/infra/websocket"
	"github.com/anthropics/agentmesh/backend/internal/service/agent"
	"github.com/anthropics/agentmesh/backend/internal/service/agentpod"
	"github.com/anthropics/agentmesh/backend/internal/service/auth"
	"github.com/anthropics/agentmesh/backend/internal/service/billing"
	"github.com/anthropics/agentmesh/backend/internal/service/binding"
	"github.com/anthropics/agentmesh/backend/internal/service/channel"
	"github.com/anthropics/agentmesh/backend/internal/service/devmesh"
	fileservice "github.com/anthropics/agentmesh/backend/internal/service/file"
	"github.com/anthropics/agentmesh/backend/internal/service/invitation"
	"github.com/anthropics/agentmesh/backend/internal/service/organization"
	"github.com/anthropics/agentmesh/backend/internal/service/promocode"
	"github.com/anthropics/agentmesh/backend/internal/service/repository"
	"github.com/anthropics/agentmesh/backend/internal/service/runner"
	"github.com/anthropics/agentmesh/backend/internal/service/ticket"
	"github.com/anthropics/agentmesh/backend/internal/service/user"
	"github.com/anthropics/agentmesh/backend/pkg/crypto"
	"github.com/redis/go-redis/v9"
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
		DevMesh:           services.devmesh,
		Billing:           services.billing,
		Hub:               hub,
		EventBus:          eventBus,
		Invitation:        services.invitation,
		File:              services.file,
		PromoCode:         services.promoCode,
		AgentPodSettings:  services.agentpodSettings,
		AgentPodAIProvider: services.agentpodAIProvider,
	}

	// Initialize router
	router := rest.NewRouter(cfg, svc, db, appLogger.Logger)

	// Create and start HTTP server
	srv := startHTTPServer(cfg, router)

	// Graceful shutdown
	waitForShutdown(srv, eventBus, heartbeatBatcher, db, redisClient)
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
	devmesh            *devmesh.Service
	invitation         *invitation.Service
	file               *fileservice.Service
	promoCode          *promocode.Service
	agentpodSettings   *agentpod.SettingsService
	agentpodAIProvider *agentpod.AIProviderService
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
		Issuer:            "agentmesh",
	}
	authSvc := auth.NewServiceWithRedis(authCfg, userSvc, redisClient)
	orgSvc := organization.NewService(db)
	// Initialize agent sub-services (split by responsibility per SRP)
	agentTypeSvc := agent.NewAgentTypeService(db)
	credentialProfileSvc := agent.NewCredentialProfileService(db, agentTypeSvc)
	userConfigSvc := agent.NewUserConfigService(db, agentTypeSvc)
	repoSvc := repository.NewService(db)
	billingSvc := billing.NewService(db, "") // Empty stripe key for now
	runnerSvc := runner.NewService(db, billingSvc)
	podSvc := agentpod.NewPodService(db)
	channelSvc := channel.NewService(db)
	ticketSvc := ticket.NewService(db)
	bindingSvc := binding.NewService(db, podSvc)
	devmeshSvc := devmesh.NewService(db, podSvc, channelSvc, bindingSvc)

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
		devmesh:            devmeshSvc,
		invitation:         invitationSvc,
		file:               fileSvc,
		promoCode:          promoCodeSvc,
		agentpodSettings:   agentpodSettingsSvc,
		agentpodAIProvider: agentpodAIProviderSvc,
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
func initializeRunnerComponents(db *gorm.DB, redisClient *redis.Client, appLogger *logger.Logger, agentTypeSvc *agent.AgentTypeService) (*runner.ConnectionManager, *runner.PodCoordinator, *runner.TerminalRouter, *runner.HeartbeatBatcher) {
	// Initialize Runner connection manager
	runnerConnMgr := runner.NewConnectionManager(appLogger.Logger)

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

// waitForShutdown handles graceful shutdown
func waitForShutdown(srv *http.Server, eventBus *eventbus.EventBus, heartbeatBatcher *runner.HeartbeatBatcher, db *gorm.DB, redisClient *redis.Client) {
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
