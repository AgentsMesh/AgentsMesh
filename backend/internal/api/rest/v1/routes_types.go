package v1

import (
	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
	"github.com/anthropics/agentsmesh/backend/internal/infra/acme"
	"github.com/anthropics/agentsmesh/backend/internal/infra/email"
	"github.com/anthropics/agentsmesh/backend/internal/infra/eventbus"
	"github.com/anthropics/agentsmesh/backend/internal/infra/websocket"
	"github.com/anthropics/agentsmesh/backend/internal/service/agent"
	"github.com/anthropics/agentsmesh/backend/internal/service/agentpod"
	apikeyservice "github.com/anthropics/agentsmesh/backend/internal/service/apikey"
	"github.com/anthropics/agentsmesh/backend/internal/service/auth"
	"github.com/anthropics/agentsmesh/backend/internal/service/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/binding"
	"github.com/anthropics/agentsmesh/backend/internal/service/channel"
	"github.com/anthropics/agentsmesh/backend/internal/service/geo"
	extensionservice "github.com/anthropics/agentsmesh/backend/internal/service/extension"
	fileservice "github.com/anthropics/agentsmesh/backend/internal/service/file"
	"github.com/anthropics/agentsmesh/backend/internal/service/invitation"
	"github.com/anthropics/agentsmesh/backend/internal/service/license"
	loop "github.com/anthropics/agentsmesh/backend/internal/service/loop"
	"github.com/anthropics/agentsmesh/backend/internal/service/mesh"
	"github.com/anthropics/agentsmesh/backend/internal/service/organization"
	"github.com/anthropics/agentsmesh/backend/internal/service/promocode"
	"github.com/anthropics/agentsmesh/backend/internal/service/relay"
	"github.com/anthropics/agentsmesh/backend/internal/service/repository"
	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
	supportticketservice "github.com/anthropics/agentsmesh/backend/internal/service/supportticket"
	"github.com/anthropics/agentsmesh/backend/internal/service/ticket"
	"github.com/anthropics/agentsmesh/backend/internal/service/user"
)

// MessageService is a type alias for agent.MessageService
type MessageService = agent.MessageService

// Services holds all service dependencies for API handlers
type Services struct {
	Auth *auth.Service
	User *user.Service
	Org  *organization.Service
	// Agent services (split by responsibility)
	AgentType          *agent.AgentTypeService
	CredentialProfile  *agent.CredentialProfileService
	UserConfig         *agent.UserConfigService
	Repository         *repository.Service
	Webhook            *repository.WebhookService // Webhook management for repositories
	Runner             *runner.Service
	RunnerConnMgr      *runner.RunnerConnectionManager // Runner gRPC connection manager
	PodCoordinator     *runner.PodCoordinator          // Pod lifecycle coordinator
	TerminalRouter     *runner.TerminalRouter          // Terminal data router
	Pod                *agentpod.PodService
	PodOrchestrator    *agentpod.PodOrchestrator            // Unified Pod creation orchestrator
	Autopilot          *agentpod.AutopilotControllerService // AutopilotController automation service
	Channel            *channel.Service
	Binding            *binding.Service
	Ticket             *ticket.Service
	MRSync             *ticket.MRSyncService // MR sync for webhook events
	Mesh               *mesh.Service
	AgentPodSettings   *agentpod.SettingsService   // AgentPod user settings
	AgentPodAIProvider *agentpod.AIProviderService // AgentPod AI provider management
	Billing            *billing.Service
	Message            *MessageService                  // Agent-to-agent messaging
	Hub                *websocket.Hub                   // WebSocket hub for real-time communication
	EventBus           *eventbus.EventBus               // Event bus for real-time events
	Email              email.Service                    // Email service
	Invitation         *invitation.Service              // Organization invitations
	File               *fileservice.Service             // File storage service
	PromoCode          *promocode.Service               // Promo code management
	License            *license.Service                 // License service for OnPremise
	APIKey             *apikeyservice.Service           // API key management for third-party access
	APIKeyAdapter      *apikeyservice.MiddlewareAdapter // API key middleware adapter
	// NOTE: GitProvider and SSHKey services have been removed (moved to user-level settings)

	// gRPC/mTLS Runner registration handler (optional, only when PKI is enabled)
	GRPCRunnerHandler *GRPCRunnerHandler

	// Sandbox query services
	SandboxQueryService *runner.SandboxQueryService // Sandbox status query service
	SandboxQuerySender  runner.SandboxQuerySender   // Sandbox query sender (gRPC adapter)

	// Relay services for terminal data streaming
	RelayManager        *relay.Manager        // Relay server management
	RelayTokenGenerator *relay.TokenGenerator // Relay token generation
	RelayDNSService     *relay.DNSService     // Relay DNS management
	RelayACMEManager    *acme.Manager         // ACME certificate management for Relay TLS

	// GeoIP resolver for geo-aware relay selection
	GeoResolver geo.Resolver

	// Runner version checker (optional, checks GitHub Releases for latest version)
	VersionChecker *runner.VersionChecker

	// Extension services (Skills marketplace, MCP servers)
	Extension         *extensionservice.Service
	ExtensionRepo     extension.Repository
	MarketplaceWorker *extensionservice.MarketplaceWorker

	// Loop services
	Loop             *loop.LoopService
	LoopRun          *loop.LoopRunService
	LoopOrchestrator *loop.LoopOrchestrator
	LoopScheduler    *loop.LoopScheduler

	// Support ticket service (user-level, no org scope)
	SupportTicket *supportticketservice.Service
}
