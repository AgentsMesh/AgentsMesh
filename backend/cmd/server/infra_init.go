package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	runnerDomain "github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	"github.com/anthropics/agentsmesh/backend/internal/infra/eventbus"
	"github.com/anthropics/agentsmesh/backend/internal/infra/logger"
	"github.com/anthropics/agentsmesh/backend/internal/infra/websocket"
	"github.com/anthropics/agentsmesh/backend/internal/service/agent"
	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
	"github.com/redis/go-redis/v9"
)

// initializeInfrastructure initializes WebSocket hub, EventBus, and Redis
func initializeInfrastructure(cfg *config.Config, appLogger *logger.Logger) (*websocket.Hub, *eventbus.EventBus, *redis.Client) {
	// Initialize WebSocket hub (sharded hub auto-starts goroutines in NewHub)
	hub := websocket.NewHub()

	// Initialize Redis client (optional, for multi-instance event sync)
	redisClient := initializeRedis(cfg)

	// Initialize EventBus for real-time events
	eventBus := eventbus.NewEventBus(redisClient, appLogger.Logger)

	return hub, eventBus, redisClient
}

// initializeRedis initializes the Redis client
func initializeRedis(cfg *config.Config) *redis.Client {
	if cfg.Redis.URL != "" {
		return initializeRedisFromURL(cfg.Redis.URL)
	}
	if cfg.Redis.Host != "" {
		return initializeRedisFromHost(cfg)
	}
	return nil
}

// initializeRedisFromURL creates a Redis client from a URL
func initializeRedisFromURL(url string) *redis.Client {
	opt, err := redis.ParseURL(url)
	if err != nil {
		slog.Warn("Failed to parse Redis URL, skipping Redis", "error", err)
		return nil
	}

	client := redis.NewClient(opt)
	if err := client.Ping(context.Background()).Err(); err != nil {
		slog.Warn("Failed to connect to Redis, events will be local only", "error", err)
		return nil
	}

	slog.Info("Redis connected", "url", url)
	return client
}

// initializeRedisFromHost creates a Redis client from host configuration
func initializeRedisFromHost(cfg *config.Config) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	if err := client.Ping(context.Background()).Err(); err != nil {
		slog.Warn("Failed to connect to Redis, events will be local only", "error", err)
		return nil
	}

	slog.Info("Redis connected", "host", cfg.Redis.Host, "port", cfg.Redis.Port)
	return client
}

// initializeRunnerComponents initializes runner-related components
func initializeRunnerComponents(
	podStore runner.PodStore,
	runnerRepo runnerDomain.RunnerRepository,
	redisClient *redis.Client,
	appLogger *logger.Logger,
	agentSvc *agent.AgentService,
) (*runner.RunnerConnectionManager, *runner.PodCoordinator, *runner.PodRouter, *runner.HeartbeatBatcher, *runner.SandboxQueryService) {
	// Initialize Runner connection manager
	runnerConnMgr := runner.NewRunnerConnectionManager(appLogger.Logger)

	// Setup AgentsProvider for initialization handshake
	agentAdapter := runner.NewAgentServiceAdapter(agentSvc)
	runnerConnMgr.SetAgentsProvider(agentAdapter)
	runnerConnMgr.SetServerVersion("1.0.0") // TODO: Get from build info

	// Start initialization timeout checker (removes connections that don't complete handshake)
	runnerConnMgr.StartInitTimeoutChecker()

	// Initialize Pod router (routes pod commands between frontend and runner)
	podRouter := runner.NewPodRouter(runnerConnMgr, appLogger.Logger)

	// Initialize Heartbeat batcher (batches heartbeat DB writes for high-scale performance)
	heartbeatBatcher := runner.NewHeartbeatBatcher(redisClient, runnerRepo, appLogger.Logger)
	heartbeatBatcher.Start()

	// Initialize Pod coordinator (manages pod lifecycle between backend and runner)
	podCoordinator := runner.NewPodCoordinator(podStore, runnerRepo, runnerConnMgr, podRouter, heartbeatBatcher, appLogger.Logger)

	// Initialize Sandbox query service (handles sandbox status queries to runners)
	sandboxQuerySvc := runner.NewSandboxQueryService(runnerConnMgr)

	return runnerConnMgr, podCoordinator, podRouter, heartbeatBatcher, sandboxQuerySvc
}
