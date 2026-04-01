package grpc

import (
	"log/slog"
	"time"

	"gorm.io/gorm"

	"github.com/anthropics/agentsmesh/backend/internal/infra/pki"
	"github.com/anthropics/agentsmesh/backend/internal/interfaces"
	"github.com/anthropics/agentsmesh/backend/internal/service/agent"
	"github.com/anthropics/agentsmesh/backend/internal/service/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/service/binding"
	"github.com/anthropics/agentsmesh/backend/internal/service/channel"
	loopService "github.com/anthropics/agentsmesh/backend/internal/service/loop"
	"github.com/anthropics/agentsmesh/backend/internal/service/repository"
	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
	"github.com/anthropics/agentsmesh/backend/internal/service/ticket"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// Certificate revocation check interval
const certRevocationCheckInterval = 5 * time.Minute

// Ensure GRPCRunnerAdapter implements the generated interface
var _ runnerv1.RunnerServiceServer = (*GRPCRunnerAdapter)(nil)

// GRPCRunnerAdapter implements the gRPC Runner service.
// It acts as a thin protocol adapter layer, handling:
// - gRPC service registration
// - mTLS identity validation
// - Proto <-> internal type conversion
//
// All connection management and business logic is delegated to RunnerConnectionManager.
//
// Code is split across multiple files:
// - runner_adapter_types.go: Core types and constructor
// - runner_adapter.go: Connect method and stream handling
// - runner_adapter_send*.go: Send* methods for sending commands to runners
// - runner_adapter_message.go: handleProtoMessage and related handlers
type GRPCRunnerAdapter struct {
	runnerv1.UnimplementedRunnerServiceServer

	logger             *slog.Logger
	db                 *gorm.DB
	runnerService      RunnerServiceInterface
	orgService         OrganizationServiceInterface
	pkiService         *pki.Service
	agentsProvider interfaces.AgentsProvider

	// Delegate connection management to RunnerConnectionManager
	connManager *runner.RunnerConnectionManager

	// MCP service dependencies (for handling MCP requests from Runner)
	podService        *agentpod.PodService
	mcpPodService     *agentpod.PodService // alias for podService, used by discovery/create
	podOrchestrator   *agentpod.PodOrchestrator // Unified Pod creation logic
	channelService    *channel.Service
	bindingService    *binding.Service
	ticketService     *ticket.Service
	repositoryService repository.RepositoryServiceInterface
	runnerMcpService  *runner.Service
	agentSvc      *agent.AgentService
	userConfigSvc     *agent.UserConfigService
	podRouter       PodRouterForMCP // *runner.PodRouter, optional
	loopService          *loopService.LoopService
	loopRunService       *loopService.LoopRunService
	loopOrchestrator     *loopService.LoopOrchestrator
}

// MCPDependencies holds optional MCP service dependencies for the gRPC adapter.
type MCPDependencies struct {
	PodService        *agentpod.PodService
	PodOrchestrator   *agentpod.PodOrchestrator // Unified Pod creation logic
	ChannelService    *channel.Service
	BindingService    *binding.Service
	TicketService     *ticket.Service
	RepositoryService repository.RepositoryServiceInterface
	RunnerService     *runner.Service
	AgentSvc      *agent.AgentService
	UserConfigSvc     *agent.UserConfigService
	PodRouter    PodRouterForMCP // *runner.PodRouter, optional
	LoopService       *loopService.LoopService
	LoopRunService    *loopService.LoopRunService
	LoopOrchestrator  *loopService.LoopOrchestrator
}

// NewGRPCRunnerAdapter creates a new gRPC Runner adapter.
func NewGRPCRunnerAdapter(
	logger *slog.Logger,
	db *gorm.DB,
	runnerService RunnerServiceInterface,
	orgService OrganizationServiceInterface,
	pkiService *pki.Service,
	agentsProvider interfaces.AgentsProvider,
	connManager *runner.RunnerConnectionManager,
	mcpDeps *MCPDependencies,
) *GRPCRunnerAdapter {
	adapter := &GRPCRunnerAdapter{
		logger:             logger,
		db:                 db,
		runnerService:      runnerService,
		orgService:         orgService,
		pkiService:         pkiService,
		agentsProvider: agentsProvider,
		connManager:        connManager,
	}

	// Wire MCP dependencies if provided
	if mcpDeps != nil {
		adapter.podService = mcpDeps.PodService
		adapter.mcpPodService = mcpDeps.PodService
		adapter.podOrchestrator = mcpDeps.PodOrchestrator
		adapter.channelService = mcpDeps.ChannelService
		adapter.bindingService = mcpDeps.BindingService
		adapter.ticketService = mcpDeps.TicketService
		adapter.repositoryService = mcpDeps.RepositoryService
		adapter.runnerMcpService = mcpDeps.RunnerService
		adapter.agentSvc = mcpDeps.AgentSvc
		adapter.userConfigSvc = mcpDeps.UserConfigSvc
		adapter.podRouter = mcpDeps.PodRouter
		adapter.loopService = mcpDeps.LoopService
		adapter.loopRunService = mcpDeps.LoopRunService
		adapter.loopOrchestrator = mcpDeps.LoopOrchestrator
	}

	return adapter
}
