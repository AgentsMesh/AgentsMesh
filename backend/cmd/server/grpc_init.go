package main

import (
	"context"
	"log/slog"

	grpcserver "github.com/anthropics/agentsmesh/backend/internal/api/grpc"
	v1 "github.com/anthropics/agentsmesh/backend/internal/api/rest/v1"
	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/infra/dns"
	"github.com/anthropics/agentsmesh/backend/internal/infra/logger"
	"github.com/anthropics/agentsmesh/backend/internal/infra/pki"
	"github.com/anthropics/agentsmesh/backend/internal/interfaces"
	"github.com/anthropics/agentsmesh/backend/internal/service/agent"
	"github.com/anthropics/agentsmesh/backend/internal/service/organization"
	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
)

// initializePKIAndGRPC initializes PKI service, gRPC Server, and gRPC runner handler.
// Returns nil values if initialization fails.
func initializePKIAndGRPC(
	cfg *config.Config,
	runnerSvc *runner.Service,
	orgSvc *organization.Service,
	agentTypeSvc *agent.AgentTypeService,
	runnerConnMgr *runner.RunnerConnectionManager,
	appLogger *logger.Logger,
	mcpDeps *grpcserver.MCPDependencies,
) (*grpcserver.Server, *v1.GRPCRunnerHandler) {
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

	// Create and start gRPC server
	grpcServerInst := createGRPCServer(cfg, pkiService, runnerSvc, orgSvc, agentTypeSvc, runnerConnMgr, appLogger, mcpDeps)
	if grpcServerInst == nil {
		return nil, grpcRunnerHandler
	}

	slog.Info("gRPC/mTLS Runner communication enabled", "grpc_address", cfg.GRPC.Address)
	return grpcServerInst, grpcRunnerHandler
}

// createGRPCServer creates and starts the gRPC server
func createGRPCServer(
	cfg *config.Config,
	pkiService *pki.Service,
	runnerSvc *runner.Service,
	orgSvc *organization.Service,
	agentTypeSvc *agent.AgentTypeService,
	runnerConnMgr *runner.RunnerConnectionManager,
	appLogger *logger.Logger,
	mcpDeps *grpcserver.MCPDependencies,
) *grpcserver.Server {
	// Create service adapters
	runnerServiceAdapter := &grpcRunnerServiceAdapter{svc: runnerSvc}
	orgServiceAdapter := &grpcOrgServiceAdapter{svc: orgSvc}
	agentTypesAdapter := &grpcAgentTypesAdapter{svc: agentTypeSvc}

	// Create gRPC server
	grpcServerInst, err := grpcserver.NewServer(&grpcserver.ServerDependencies{
		Logger:             appLogger.Logger,
		Config:             &cfg.GRPC,
		PKIService:         pkiService,
		RunnerService:      runnerServiceAdapter,
		OrgService:         orgServiceAdapter,
		AgentTypesProvider: agentTypesAdapter,
		ConnManager:        runnerConnMgr,
		MCPDeps:            mcpDeps,
	})
	if err != nil {
		slog.Error("Failed to create gRPC server", "error", err)
		slog.Warn("Continuing without gRPC server")
		return nil
	}

	// Start gRPC server
	if err := grpcServerInst.Start(); err != nil {
		slog.Error("Failed to start gRPC server", "error", err)
		slog.Warn("Continuing without gRPC server")
		return nil
	}

	return grpcServerInst
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

// grpcAgentTypesAdapter adapts agent.AgentTypeService to interfaces.AgentTypesProvider
type grpcAgentTypesAdapter struct {
	svc *agent.AgentTypeService
}

func (a *grpcAgentTypesAdapter) GetAgentTypesForRunner() []interfaces.AgentTypeInfo {
	types := a.svc.GetAgentTypesForRunner()
	result := make([]interfaces.AgentTypeInfo, len(types))
	for i, t := range types {
		result[i] = interfaces.AgentTypeInfo{
			Slug:          t.Slug,
			Name:          t.Name,
			Executable:    t.Executable,
			LaunchCommand: t.LaunchCommand,
		}
	}
	return result
}

// createDNSProvider creates a DNS provider based on configuration
func createDNSProvider(relayCfg config.RelayConfig) dns.Provider {
	switch relayCfg.DNS.Provider {
	case string(config.DNSProviderCloudflare):
		if relayCfg.DNS.CloudflareAPIToken == "" || relayCfg.DNS.CloudflareZoneID == "" {
			return nil
		}
		return dns.NewCloudflareProvider(relayCfg.DNS.CloudflareAPIToken, relayCfg.DNS.CloudflareZoneID)
	case string(config.DNSProviderAliyun):
		if relayCfg.DNS.AliyunAccessKeyID == "" || relayCfg.DNS.AliyunAccessKeySecret == "" {
			return nil
		}
		return dns.NewAliyunProvider(relayCfg.DNS.AliyunAccessKeyID, relayCfg.DNS.AliyunAccessKeySecret)
	default:
		return nil
	}
}
