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

	grpcserver "github.com/anthropics/agentsmesh/backend/internal/api/grpc"
	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/infra/database"
	"github.com/anthropics/agentsmesh/backend/internal/infra/email"
	"github.com/anthropics/agentsmesh/backend/internal/infra/eventbus"
	"github.com/anthropics/agentsmesh/backend/internal/job"
	"github.com/anthropics/agentsmesh/backend/internal/service/instance"
	"github.com/anthropics/agentsmesh/backend/internal/service/relay"
	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

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
func startSubscriptionJobs(db *gorm.DB, appConfig *config.Config, emailSvc email.Service, logger *slog.Logger) *job.SubscriptionScheduler {
	scheduler := job.NewSubscriptionScheduler(db, appConfig, emailSvc, logger)
	scheduler.Start()
	slog.Info("subscription scheduler started")
	return scheduler
}

// LoopSchedulerStopper is an interface for stopping the loop scheduler
type LoopSchedulerStopper interface {
	Stop()
}

// waitForShutdown handles graceful shutdown
func waitForShutdown(
	srv *http.Server,
	grpcServer *grpcserver.Server,
	eventBus *eventbus.EventBus,
	heartbeatBatcher *runner.HeartbeatBatcher,
	subscriptionScheduler *job.SubscriptionScheduler,
	loopScheduler LoopSchedulerStopper,
	orgAwareness *instance.OrgAwarenessService,
	relayManager *relay.Manager,
	db *gorm.DB,
	redisClient *redis.Client,
) {
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

	// Stop loop scheduler
	if loopScheduler != nil {
		loopScheduler.Stop()
	}

	// Stop org awareness service
	if orgAwareness != nil {
		orgAwareness.Stop()
	}

	// Stop heartbeat batcher (flush pending writes)
	if heartbeatBatcher != nil {
		heartbeatBatcher.Stop()
	}

	// Stop relay manager (terminates health check goroutine)
	if relayManager != nil {
		relayManager.Stop()
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
