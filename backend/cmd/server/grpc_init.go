package main

import (
	"context"
	"log/slog"
	"net/url"
	"strings"

	grpcserver "github.com/anthropics/agentsmesh/backend/internal/api/grpc"
	v1 "github.com/anthropics/agentsmesh/backend/internal/api/rest/v1"
	"github.com/anthropics/agentsmesh/backend/internal/config"
	runnerDomain "github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	"github.com/anthropics/agentsmesh/backend/internal/infra/dns"
	"github.com/anthropics/agentsmesh/backend/internal/infra/logger"
	"github.com/anthropics/agentsmesh/backend/internal/infra/pki"
	"github.com/anthropics/agentsmesh/backend/internal/interfaces"
	"github.com/anthropics/agentsmesh/backend/internal/service/agent"
	"github.com/anthropics/agentsmesh/backend/internal/service/organization"
	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
)

func initializePKIAndGRPC(
	cfg *config.Config,
	runnerSvc *runner.Service,
	orgSvc *organization.Service,
	agentSvc *agent.AgentService,
	runnerConnMgr *runner.RunnerConnectionManager,
	appLogger *logger.Logger,
	mcpDeps *grpcserver.MCPDependencies,
) (*grpcserver.Server, *v1.GRPCRunnerHandler) {
	pkiService, err := pki.NewService(&pki.Config{
		CACertFile:     cfg.PKI.CACertFile,
		CAKeyFile:      cfg.PKI.CAKeyFile,
		ServerCertFile: cfg.PKI.ServerCertFile,
		ServerKeyFile:  cfg.PKI.ServerKeyFile,
		ValidityDays:   cfg.PKI.ValidityDays,
		ServerCertSANs: deriveServerCertSANs(cfg),
	})
	if err != nil {
		slog.Error("Failed to initialize PKI service", "error", err)
		slog.Warn("Continuing without gRPC/mTLS support - token management routes still available")
		return nil, v1.NewGRPCRunnerHandler(runnerSvc, nil, cfg)
	}

	slog.Info("PKI service initialized",
		"ca_cert", cfg.PKI.CACertFile,
		"validity_days", cfg.PKI.ValidityDays,
	)

	grpcRunnerHandler := v1.NewGRPCRunnerHandler(runnerSvc, pkiService, cfg)

	grpcServerInst := createGRPCServer(cfg, pkiService, runnerSvc, orgSvc, agentSvc, runnerConnMgr, appLogger, mcpDeps)
	if grpcServerInst == nil {
		return nil, grpcRunnerHandler
	}

	slog.Info("gRPC/mTLS Runner communication enabled", "grpc_address", cfg.GRPC.Address)
	return grpcServerInst, grpcRunnerHandler
}

func createGRPCServer(
	cfg *config.Config,
	pkiService *pki.Service,
	runnerSvc *runner.Service,
	orgSvc *organization.Service,
	agentSvc *agent.AgentService,
	runnerConnMgr *runner.RunnerConnectionManager,
	appLogger *logger.Logger,
	mcpDeps *grpcserver.MCPDependencies,
) *grpcserver.Server {
	runnerServiceAdapter := &grpcRunnerServiceAdapter{svc: runnerSvc}
	orgServiceAdapter := &grpcOrgServiceAdapter{svc: orgSvc}
	agentAdapter := &grpcAgentAdapter{svc: agentSvc}

	grpcServerInst, err := grpcserver.NewServer(&grpcserver.ServerDependencies{
		Logger:             appLogger.Logger,
		Config:             &cfg.GRPC,
		PKIService:         pkiService,
		RunnerService:      runnerServiceAdapter,
		OrgService:         orgServiceAdapter,
		AgentsProvider: agentAdapter,
		ConnManager:        runnerConnMgr,
		MCPDeps:            mcpDeps,
	})
	if err != nil {
		slog.Error("Failed to create gRPC server", "error", err)
		slog.Warn("Continuing without gRPC server")
		return nil
	}

	if err := grpcServerInst.Start(); err != nil {
		slog.Error("Failed to start gRPC server", "error", err)
		slog.Warn("Continuing without gRPC server")
		return nil
	}

	return grpcServerInst
}

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

func (a *grpcRunnerServiceAdapter) GetByNodeIDAndOrgID(ctx context.Context, nodeID string, orgID int64) (grpcserver.RunnerInfo, error) {
	r, err := a.svc.GetByNodeIDAndOrgID(ctx, nodeID, orgID)
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

func (a *grpcRunnerServiceAdapter) UpdateAgentVersions(ctx context.Context, runnerID int64, versions []runnerDomain.AgentVersion) error {
	return a.svc.UpdateAgentVersions(ctx, runnerID, versions)
}

func (a *grpcRunnerServiceAdapter) IsCertificateRevoked(ctx context.Context, serialNumber string) (bool, error) {
	return a.svc.IsCertificateRevoked(ctx, serialNumber)
}

func (a *grpcRunnerServiceAdapter) UpdateRunnerVersionAndHostInfo(ctx context.Context, runnerID int64, version string, hostInfo map[string]interface{}) error {
	return a.svc.UpdateRunnerVersionAndHostInfo(ctx, runnerID, version, hostInfo)
}

func (a *grpcRunnerServiceAdapter) MergeAgentVersions(ctx context.Context, runnerID int64, changes map[string]runnerDomain.AgentVersion) error {
	return a.svc.MergeAgentVersions(ctx, runnerID, changes)
}

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

type grpcAgentAdapter struct {
	svc *agent.AgentService
}

func (a *grpcAgentAdapter) GetAgentsForRunner() []interfaces.AgentInfo {
	agents := a.svc.GetAgentsForRunner()
	result := make([]interfaces.AgentInfo, len(agents))
	for i, ag := range agents {
		result[i] = interfaces.AgentInfo{
			Slug:          ag.Slug,
			Name:          ag.Name,
			Executable:    ag.Executable,
			LaunchCommand: ag.LaunchCommand,
		}
	}
	return result
}

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

func deriveServerCertSANs(cfg *config.Config) []string {
	var sans []string

	if cfg.PrimaryDomain != "" {
		host := cfg.PrimaryDomain
		if idx := strings.LastIndex(host, ":"); idx != -1 {
			host = host[:idx]
		}
		if host != "" {
			sans = append(sans, host)
		}
	}

	if cfg.GRPC.Endpoint != "" {
		if u, err := url.Parse(cfg.GRPC.Endpoint); err == nil && u.Hostname() != "" {
			sans = append(sans, u.Hostname())
		}
	}

	if len(sans) > 0 {
		slog.Info("Server certificate SANs derived from config", "sans", sans)
	}
	return sans
}
