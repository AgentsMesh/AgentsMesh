package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/anthropics/agentsmesh/relay/internal/config"
	"github.com/anthropics/agentsmesh/relay/internal/server"
)

func main() {
	// Set up structured logging
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug, // Enable debug logging for troubleshooting
	})
	slog.SetDefault(slog.New(handler))

	slog.Info("Starting AgentsMesh Relay Server")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	slog.Info("Configuration loaded",
		"relay_id", cfg.Relay.ID,
		"relay_url", cfg.Relay.URL,
		"region", cfg.Relay.Region,
		"capacity", cfg.Relay.Capacity,
		"server_address", cfg.Server.Address())

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		slog.Info("Received shutdown signal", "signal", sig)
		cancel()
	}()

	// Create and start server
	srv := server.New(cfg)
	if err := srv.Start(ctx); err != nil {
		slog.Error("Server error", "error", err)
		os.Exit(1)
	}

	slog.Info("Server stopped")
}
