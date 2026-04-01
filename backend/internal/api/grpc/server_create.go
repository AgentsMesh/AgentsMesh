package grpc

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	_ "google.golang.org/grpc/encoding/gzip" // Register gzip compressor/decompressor
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

// NewServer creates a new gRPC server for Runner communication.
// The server handles mTLS directly for TLS passthrough mode.
func NewServer(deps *ServerDependencies) (*Server, error) {
	if deps == nil {
		return nil, fmt.Errorf("dependencies are required")
	}
	if deps.Config == nil {
		return nil, fmt.Errorf("gRPC config is required")
	}
	if deps.Logger == nil {
		deps.Logger = slog.Default()
	}

	opts := buildServerOptions(deps)
	grpcServer := grpc.NewServer(opts...)

	// Create and register Runner service adapter (delegates to RunnerConnectionManager)
	runnerAdapter := NewGRPCRunnerAdapter(
		deps.Logger,
		deps.DB,
		deps.RunnerService,
		deps.OrgService,
		deps.PKIService,
		deps.AgentsProvider,
		deps.ConnManager,
		deps.MCPDeps,
	)

	// Register RunnerService with gRPC server
	runnerAdapter.Register(grpcServer)

	// Enable reflection for debugging/testing
	reflection.Register(grpcServer)

	return &Server{
		grpcServer:    grpcServer,
		logger:        deps.Logger,
		config:        deps.Config,
		pkiService:    deps.PKIService,
		runnerAdapter: runnerAdapter,
	}, nil
}

// buildServerOptions constructs gRPC server options including TLS and keepalive.
func buildServerOptions(deps *ServerDependencies) []grpc.ServerOption {
	opts := []grpc.ServerOption{
		// Keepalive configuration for long-running streams
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     0,                 // Never close idle connections
			MaxConnectionAge:      0,                 // Never close connections due to age
			MaxConnectionAgeGrace: 0,                 // No grace period
			Time:                  30 * time.Second,  // 30 seconds ping interval
			Timeout:               10 * time.Second,  // 10 seconds ping timeout
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             10 * time.Second, // 10 seconds minimum between client pings
			PermitWithoutStream: true,             // Allow pings without active streams
		}),
		// Message size limits
		grpc.MaxRecvMsgSize(16 * 1024 * 1024), // 16MB max receive
		grpc.MaxSendMsgSize(16 * 1024 * 1024), // 16MB max send
		// Interceptors
		grpc.ChainUnaryInterceptor(
			loggingUnaryInterceptor(deps.Logger),
		),
		grpc.ChainStreamInterceptor(
			loggingStreamInterceptor(deps.Logger),
		),
	}

	// Configure mTLS if PKI service is available
	if deps.PKIService != nil {
		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{deps.PKIService.ServerCert()},
			ClientCAs:    deps.PKIService.CACertPool(),
			ClientAuth:   tls.RequireAndVerifyClientCert,
			MinVersion:   tls.VersionTLS13,
		}
		creds := credentials.NewTLS(tlsConfig)
		opts = append(opts, grpc.Creds(creds))
		deps.Logger.Info("gRPC server configured with mTLS")
	} else {
		deps.Logger.Warn("gRPC server running without TLS (PKI service not available)")
	}

	return opts
}

// Start starts the gRPC server on the configured address.
func (s *Server) Start() error {
	addr := s.config.Address
	if addr == "" {
		addr = ":9090"
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}
	s.listener = listener

	s.logger.Info("gRPC server starting", "address", addr)

	// Serve in goroutine
	go func() {
		if err := s.grpcServer.Serve(listener); err != nil {
			s.logger.Error("gRPC server error", "error", err)
		}
	}()

	return nil
}
