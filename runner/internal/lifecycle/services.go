// Package lifecycle provides suture.Service adapters for Runner's core components.
// These adapters wrap existing components to integrate with suture/v4 Supervisor tree
// without modifying the components' internal logic.
package lifecycle

import (
	"context"

	"github.com/AgentsMesh/AgentsMesh/runner/internal/logger"
)

// ConnectionStarter defines the lifecycle interface for the gRPC connection.
// This is a subset of client.Connection to avoid circular imports.
type ConnectionStarter interface {
	Start()
	Stop()
}

// ConnectionService wraps a gRPC connection as a suture.Service.
// The connectionLoop already has complete reconnection logic internally;
// this adapter only manages Start/Stop lifecycle.
type ConnectionService struct {
	Conn ConnectionStarter
}

// Serve implements suture.Service. It starts the connection and blocks until ctx is cancelled.
func (s *ConnectionService) Serve(ctx context.Context) error {
	log := logger.Runner()
	log.Info("ConnectionService starting")

	s.Conn.Start()
	defer func() {
		log.Info("ConnectionService stopping")
		s.Conn.Stop()
	}()

	<-ctx.Done()
	return ctx.Err()
}

// String returns the service name for logging.
func (s *ConnectionService) String() string {
	return "ConnectionService"
}

// HTTPServerLike defines the lifecycle interface for HTTP-based servers (MCP, Console).
type HTTPServerLike interface {
	Start() error
	Stop() error
}

// MCPServerService wraps the MCP HTTP Server as a suture.Service.
type MCPServerService struct {
	Server HTTPServerLike
}

// Serve implements suture.Service.
func (s *MCPServerService) Serve(ctx context.Context) error {
	log := logger.Runner()
	log.Info("MCPServerService starting")

	if err := s.Server.Start(); err != nil {
		return err
	}
	defer func() {
		log.Info("MCPServerService stopping")
		s.Server.Stop()
	}()

	<-ctx.Done()
	return ctx.Err()
}

// String returns the service name for logging.
func (s *MCPServerService) String() string {
	return "MCPServerService"
}

// MonitorStartStopper defines the lifecycle interface for the Monitor.
type MonitorStartStopper interface {
	Start()
	Stop()
}

// MonitorService wraps the process Monitor as a suture.Service.
type MonitorService struct {
	Monitor MonitorStartStopper
}

// Serve implements suture.Service.
func (s *MonitorService) Serve(ctx context.Context) error {
	log := logger.Runner()
	log.Info("MonitorService starting")

	s.Monitor.Start()
	defer func() {
		log.Info("MonitorService stopping")
		s.Monitor.Stop()
	}()

	<-ctx.Done()
	return ctx.Err()
}

// String returns the service name for logging.
func (s *MonitorService) String() string {
	return "MonitorService"
}

// ConsoleService wraps the Console HTTP Server as a suture.Service.
type ConsoleService struct {
	Server HTTPServerLike
}

// Serve implements suture.Service.
func (s *ConsoleService) Serve(ctx context.Context) error {
	log := logger.Runner()
	log.Info("ConsoleService starting")

	if err := s.Server.Start(); err != nil {
		return err
	}
	defer func() {
		log.Info("ConsoleService stopping")
		s.Server.Stop()
	}()

	<-ctx.Done()
	return ctx.Err()
}

// String returns the service name for logging.
func (s *ConsoleService) String() string {
	return "ConsoleService"
}
