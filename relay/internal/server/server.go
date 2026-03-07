package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/anthropics/agentsmesh/relay/internal/auth"
	"github.com/anthropics/agentsmesh/relay/internal/backend"
	"github.com/anthropics/agentsmesh/relay/internal/channel"
	"github.com/anthropics/agentsmesh/relay/internal/config"
)

// Server is the main relay server
type Server struct {
	cfg            *config.Config
	httpServer     *http.Server
	channelManager *channel.ChannelManager
	backendClient  *backend.Client
	handler        *Handler

	// Graceful shutdown control
	acceptingConnections atomic.Bool

	logger *slog.Logger
}

// New creates a new relay server
func New(cfg *config.Config) *Server {
	// Create backend client with full configuration
	backendClient := backend.NewClientWithConfig(backend.ClientConfig{
		BaseURL:           cfg.Backend.URL,
		InternalAPISecret: cfg.Backend.InternalAPISecret,
		RelayID:           cfg.Relay.ID,
		RelayName:         cfg.Relay.Name,
		RelayURL:          cfg.Relay.URL,
		RelayRegion:       cfg.Relay.Region,
		RelayCapacity:     cfg.Relay.Capacity,
		AutoIP:            cfg.Relay.AutoIP,
		CertFile:          cfg.Server.TLS.CertFile,
		KeyFile:           cfg.Server.TLS.KeyFile,
	})

	// Create server instance first (for closure capture)
	s := &Server{
		cfg:           cfg,
		backendClient: backendClient,
		logger:        slog.With("component", "server"),
	}
	s.acceptingConnections.Store(true)

	// Create callback for when all subscribers leave a channel
	// Uses closure to capture server instance instead of global variable
	onAllSubscribersGone := func(podKey string) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Notify backend that all subscribers left this pod's channel
		// Note: We no longer have sessionID, just use podKey
		if err := s.backendClient.NotifySessionClosed(ctx, podKey, ""); err != nil {
			s.logger.Warn("Failed to notify backend of channel closed", "pod_key", podKey, "error", err)
		}
	}

	// Create channel manager with full configuration
	// Note: SessionConfig fields map to ChannelManagerConfig
	managerCfg := channel.ChannelManagerConfig{
		KeepAliveDuration:          cfg.Session.KeepAliveDuration,
		MaxSubscribersPerPod:       cfg.Session.MaxBrowsersPerPod,
		PublisherReconnectTimeout:  cfg.Session.RunnerReconnectTimeout,
		SubscriberReconnectTimeout: cfg.Session.BrowserReconnectTimeout,
		PendingConnectionTimeout:   cfg.Session.PendingConnectionTimeout,
		OutputBufferSize:           cfg.Session.OutputBufferSize,
		OutputBufferCount:          cfg.Session.OutputBufferCount,
	}
	s.channelManager = channel.NewChannelManagerWithConfig(managerCfg, onAllSubscribersGone)

	// Create token validator
	tokenValidator := auth.NewTokenValidator(cfg.JWT.Secret, cfg.JWT.Issuer)

	// Create handler
	s.handler = NewHandler(s.channelManager, tokenValidator)

	// Share the acceptingConnections flag between server and handler
	s.handler.acceptingConnections = &s.acceptingConnections

	return s
}

// Start starts the relay server
func (s *Server) Start(ctx context.Context) error {
	// Register with backend. Docker Compose `depends_on: backend: condition: service_healthy`
	// ensures the backend is ready before the relay starts — no retry logic needed here.
	if err := s.backendClient.Register(ctx); err != nil {
		s.channelManager.Close() // clean up cleanup goroutine started in constructor
		return fmt.Errorf("failed to register with backend (ensure backend is running and healthy): %w", err)
	}

	// Start heartbeat loop with automatic restart on unexpected exit
	go s.runHeartbeat(ctx)

	// Set up HTTP routes
	mux := http.NewServeMux()
	mux.HandleFunc("/runner/terminal", s.handler.HandleRunnerWS)
	mux.HandleFunc("/browser/terminal", s.handler.HandleBrowserWS)
	mux.HandleFunc("/health", s.handler.HandleHealth)
	mux.HandleFunc("/stats", s.handler.HandleStats)

	s.httpServer = &http.Server{
		Addr:              s.cfg.Server.Address(),
		Handler:           mux,
		ReadTimeout:       s.cfg.Server.ReadTimeout,
		ReadHeaderTimeout: 10 * time.Second,
		// WriteTimeout intentionally omitted: it applies globally including
		// to upgraded WebSocket connections, killing them after the timeout.
	}

	// Start server in goroutine
	errCh := make(chan error, 1)

	// Check if we should use TLS
	useTLS := s.cfg.Server.TLS.Enabled

	if useTLS {
		// Use GetCertificate callback to dynamically load certificate from backend
		// This allows certificate to be updated via heartbeat without server restart
		s.httpServer.TLSConfig = &tls.Config{
			GetCertificate: func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
				// First try to get certificate from backend client (ACME)
				if s.backendClient.HasTLSCertificate() {
					certPEM, keyPEM := s.backendClient.GetTLSCertificate()
					cert, err := tls.X509KeyPair([]byte(certPEM), []byte(keyPEM))
					if err != nil {
						s.logger.Error("Failed to load TLS certificate from backend", "error", err)
					} else {
						return &cert, nil
					}
				}

				// Fall back to certificate files if configured
				if s.cfg.Server.TLS.CertFile != "" && s.cfg.Server.TLS.KeyFile != "" {
					cert, err := tls.LoadX509KeyPair(s.cfg.Server.TLS.CertFile, s.cfg.Server.TLS.KeyFile)
					if err != nil {
						return nil, fmt.Errorf("failed to load certificate files: %w", err)
					}
					return &cert, nil
				}

				return nil, fmt.Errorf("no TLS certificate available")
			},
		}

		s.logger.Info("Starting relay server with TLS (dynamic certificate loading)",
			"address", s.cfg.Server.Address())
		go func() {
			// ListenAndServeTLS with empty cert/key paths uses TLSConfig.GetCertificate
			if err := s.httpServer.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
				errCh <- err
			}
		}()
	} else {
		s.logger.Info("Starting relay server", "address", s.cfg.Server.Address())
		go func() {
			if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				errCh <- err
			}
		}()
	}

	// Wait for context cancellation or server error
	// Signal handling is done by the caller (main.go) which cancels the context.
	select {
	case <-ctx.Done():
		return s.gracefulShutdown("context_cancelled")
	case err := <-errCh:
		return err
	}
}

// runHeartbeat runs the heartbeat loop with automatic restart on unexpected exit.
// This provides resilience against panics or unexpected goroutine termination.
func (s *Server) runHeartbeat(ctx context.Context) {
	const restartDelay = 5 * time.Second

	for {
		s.backendClient.StartHeartbeat(ctx, s.cfg.Backend.HeartbeatInterval, func() int {
			stats := s.channelManager.Stats()
			return stats.ActiveChannels
		})

		// If context is cancelled, don't restart
		if ctx.Err() != nil {
			return
		}

		// Unexpected exit (e.g., recovered panic), restart after delay
		s.logger.Error("Heartbeat goroutine exited unexpectedly, restarting",
			"restart_delay", restartDelay)
		select {
		case <-time.After(restartDelay):
			s.logger.Info("Restarting heartbeat goroutine")
		case <-ctx.Done():
			return
		}
	}
}

// gracefulShutdown performs a graceful shutdown of the relay server
func (s *Server) gracefulShutdown(reason string) error {
	s.logger.Info("Starting graceful shutdown...", "reason", reason)

	// 1. Stop accepting new connections first, so no new sessions are created
	// while we notify the backend that this relay is going offline.
	s.acceptingConnections.Store(false)
	s.logger.Info("Stopped accepting new connections")

	// 2. Notify backend that this relay is going offline
	unregCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := s.backendClient.Unregister(unregCtx, reason); err != nil {
		s.logger.Warn("Failed to unregister from backend", "error", err)
	}
	cancel()

	// 3. Wait for existing channels to close (with timeout)
	gracePeriod := 30 * time.Second
	deadline := time.Now().Add(gracePeriod)

	for time.Now().Before(deadline) {
		stats := s.channelManager.Stats()
		if stats.ActiveChannels == 0 {
			s.logger.Info("All channels closed")
			break
		}
		s.logger.Info("Waiting for channels to close",
			"remaining", stats.ActiveChannels,
			"time_left", time.Until(deadline).Round(time.Second))
		time.Sleep(1 * time.Second)
	}

	// 4. Shutdown HTTP server
	s.logger.Info("Shutting down HTTP server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		s.logger.Error("HTTP server shutdown error", "error", err)
		return err
	}

	// 5. Stop channel manager cleanup goroutine
	s.channelManager.Close()

	s.logger.Info("Graceful shutdown completed")
	return nil
}

// IsAcceptingConnections returns whether the server is accepting new connections
func (s *Server) IsAcceptingConnections() bool {
	return s.acceptingConnections.Load()
}

// Stats returns server statistics
func (s *Server) Stats() channel.ChannelStats {
	return s.channelManager.Stats()
}
