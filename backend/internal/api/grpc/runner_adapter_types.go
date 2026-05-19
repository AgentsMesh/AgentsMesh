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
	blockstoreservice "github.com/anthropics/agentsmesh/backend/internal/service/blockstore"
	"github.com/anthropics/agentsmesh/backend/internal/service/channel"
	loopService "github.com/anthropics/agentsmesh/backend/internal/service/loop"
	"github.com/anthropics/agentsmesh/backend/internal/service/repository"
	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
	"github.com/anthropics/agentsmesh/backend/internal/service/ticket"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

const certRevocationCheckInterval = 5 * time.Minute

var _ runnerv1.RunnerServiceServer = (*GRPCRunnerAdapter)(nil)

type GRPCRunnerAdapter struct {
	runnerv1.UnimplementedRunnerServiceServer

	logger             *slog.Logger
	db                 *gorm.DB
	runnerService      RunnerServiceInterface
	orgService         OrganizationServiceInterface
	pkiService         *pki.Service
	agentsProvider interfaces.AgentsProvider

	connManager *runner.RunnerConnectionManager

	podService        *agentpod.PodService
	mcpPodService     *agentpod.PodService
	podOrchestrator   *agentpod.PodOrchestrator
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
	blockstoreService    *blockstoreservice.Service
}

type MCPDependencies struct {
	PodService        *agentpod.PodService
	PodOrchestrator   *agentpod.PodOrchestrator
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
	BlockstoreService *blockstoreservice.Service
}

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
		adapter.blockstoreService = mcpDeps.BlockstoreService
	}

	return adapter
}
