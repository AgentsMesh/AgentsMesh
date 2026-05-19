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
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
)

func initializeInfrastructure(cfg *config.Config, appLogger *logger.Logger) (*websocket.Hub, *eventbus.EventBus, *redis.Client) {
	hub := websocket.NewHub()

	redisClient := initializeRedis(cfg)

	eventBus := eventbus.NewEventBus(redisClient, appLogger.Logger)

	return hub, eventBus, redisClient
}

func initializeRedis(cfg *config.Config) *redis.Client {
	if cfg.Redis.URL != "" {
		return initializeRedisFromURL(cfg.Redis.URL)
	}
	if cfg.Redis.Host != "" {
		return initializeRedisFromHost(cfg)
	}
	return nil
}

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

	instrumentRedis(client)
	slog.Info("Redis connected", "url", url)
	return client
}

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

	instrumentRedis(client)
	slog.Info("Redis connected", "host", cfg.Redis.Host, "port", cfg.Redis.Port)
	return client
}

func initializeRunnerComponents(
	podStore runner.PodStore,
	runnerRepo runnerDomain.RunnerRepository,
	redisClient *redis.Client,
	appLogger *logger.Logger,
	agentSvc *agent.AgentService,
) (*runner.RunnerConnectionManager, *runner.PodCoordinator, *runner.PodRouter, *runner.HeartbeatBatcher, *runner.SandboxQueryService) {
	runnerConnMgr := runner.NewRunnerConnectionManager(appLogger.Logger)

	agentAdapter := runner.NewAgentServiceAdapter(agentSvc)
	runnerConnMgr.SetAgentsProvider(agentAdapter)
	runnerConnMgr.SetServerVersion("1.0.0") // TODO: Get from build info

	runnerConnMgr.StartInitTimeoutChecker()

	podRouter := runner.NewPodRouter(runnerConnMgr, appLogger.Logger)

	heartbeatBatcher := runner.NewHeartbeatBatcher(redisClient, runnerRepo, appLogger.Logger)
	heartbeatBatcher.Start()

	podCoordinator := runner.NewPodCoordinator(podStore, runnerRepo, runnerConnMgr, podRouter, heartbeatBatcher, appLogger.Logger)

	sandboxQuerySvc := runner.NewSandboxQueryService(runnerConnMgr)

	return runnerConnMgr, podCoordinator, podRouter, heartbeatBatcher, sandboxQuerySvc
}

func instrumentRedis(client *redis.Client) {
	if err := redisotel.InstrumentTracing(client); err != nil {
		slog.Warn("failed to enable Redis tracing", "error", err)
	}
	if err := redisotel.InstrumentMetrics(client); err != nil {
		slog.Warn("failed to enable Redis metrics", "error", err)
	}
}
