package main

import (
	"context"
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
	"github.com/anthropics/agentmesh/backend/internal/infra/email"
	"github.com/anthropics/agentmesh/backend/internal/infra/logger"
	"github.com/anthropics/agentmesh/backend/internal/infra/storage"
	"github.com/anthropics/agentmesh/backend/internal/infra/websocket"
	"github.com/anthropics/agentmesh/backend/internal/service/agent"
	"github.com/anthropics/agentmesh/backend/internal/service/auth"
	"github.com/anthropics/agentmesh/backend/internal/service/billing"
	"github.com/anthropics/agentmesh/backend/internal/service/binding"
	"github.com/anthropics/agentmesh/backend/internal/service/channel"
	"github.com/anthropics/agentmesh/backend/internal/service/devmesh"
	"github.com/anthropics/agentmesh/backend/internal/service/gitprovider"
	"github.com/anthropics/agentmesh/backend/internal/service/invitation"
	"github.com/anthropics/agentmesh/backend/internal/service/organization"
	"github.com/anthropics/agentmesh/backend/internal/service/repository"
	"github.com/anthropics/agentmesh/backend/internal/service/agentpod"
	fileservice "github.com/anthropics/agentmesh/backend/internal/service/file"
	"github.com/anthropics/agentmesh/backend/internal/service/runner"
	"github.com/anthropics/agentmesh/backend/internal/service/sshkey"
	"github.com/anthropics/agentmesh/backend/internal/service/ticket"
	"github.com/anthropics/agentmesh/backend/internal/service/user"
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

	// Initialize database
	db, err := database.New(cfg.Database)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Initialize services
	// Use JWT secret as encryption key for token encryption (OAuth tokens, etc.)
	userSvc := user.NewServiceWithEncryption(db, cfg.JWT.Secret)
	authCfg := &auth.Config{
		JWTSecret:         cfg.JWT.Secret,
		JWTExpiration:     time.Duration(cfg.JWT.ExpirationHours) * time.Hour,
		RefreshExpiration: time.Duration(cfg.JWT.ExpirationHours*7) * time.Hour, // 7x access token
		Issuer:            "agentmesh",
	}
	authSvc := auth.NewService(authCfg, userSvc)
	orgSvc := organization.NewService(db)
	agentSvc := agent.NewService(db)
	gitProviderSvc := gitprovider.NewService(db)
	repoSvc := repository.NewService(db)
	runnerSvc := runner.NewService(db)
	podSvc := agentpod.NewPodService(db)
	channelSvc := channel.NewService(db)
	ticketSvc := ticket.NewService(db)
	sshKeySvc := sshkey.NewService(db)
	billingSvc := billing.NewService(db, "") // Empty stripe key for now
	bindingSvc := binding.NewService(db, podSvc) // podSvc implements PodQuerier for auto-approve
	devmeshSvc := devmesh.NewService(db, podSvc, channelSvc, bindingSvc)

	// Initialize email service for invitations
	emailSvc := email.NewService(email.Config{
		Provider:    cfg.Email.Provider,
		ResendKey:   cfg.Email.ResendKey,
		FromAddress: cfg.Email.FromAddress,
		BaseURL:     cfg.Email.BaseURL,
	})
	invitationSvc := invitation.NewService(db, emailSvc)

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

	// Initialize WebSocket hub
	hub := websocket.NewHub()
	go hub.Run()

	// Initialize Runner connection manager
	runnerConnMgr := runner.NewConnectionManager(appLogger.Logger)

	// Initialize Terminal router (routes terminal data between frontend and runner)
	terminalRouter := runner.NewTerminalRouter(runnerConnMgr, appLogger.Logger)

	// Initialize Pod coordinator (manages pod lifecycle between backend and runner)
	podCoordinator := runner.NewPodCoordinator(db, runnerConnMgr, terminalRouter, appLogger.Logger)

	// Create services container
	svc := &v1.Services{
		Auth:           authSvc,
		User:           userSvc,
		Org:            orgSvc,
		Agent:          agentSvc,
		GitProvider:    gitProviderSvc,
		Repository:     repoSvc,
		Runner:         runnerSvc,
		RunnerConnMgr:  runnerConnMgr,
		PodCoordinator: podCoordinator,
		TerminalRouter: terminalRouter,
		Pod:            podSvc,
		Channel:        channelSvc,
		Binding:        bindingSvc,
		Ticket:         ticketSvc,
		SSHKey:         sshKeySvc,
		DevMesh:        devmeshSvc,
		Billing:        billingSvc,
		Hub:            hub,
		Invitation:     invitationSvc,
		File:           fileSvc,
	}

	// Initialize router
	router := rest.NewRouter(cfg, svc, db, appLogger.Logger)

	// Create HTTP server
	srv := &http.Server{
		Addr:         cfg.Server.Address,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		slog.Info("Starting server", "address", cfg.Server.Address)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Failed to start server", "error", err)
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Graceful shutdown
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

	slog.Info("Server exited")
}
