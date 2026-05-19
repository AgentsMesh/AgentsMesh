package v1

import (
	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
	"github.com/anthropics/agentsmesh/backend/internal/infra/acme"
	"github.com/anthropics/agentsmesh/backend/internal/infra/email"
	"github.com/anthropics/agentsmesh/backend/internal/infra/eventbus"
	"github.com/anthropics/agentsmesh/backend/internal/infra/websocket"
	"github.com/redis/go-redis/v9"
	"github.com/anthropics/agentsmesh/backend/internal/service/agent"
	"github.com/anthropics/agentsmesh/backend/internal/service/agentpod"
	apikeyservice "github.com/anthropics/agentsmesh/backend/internal/service/apikey"
	"github.com/anthropics/agentsmesh/backend/internal/service/auth"
	"github.com/anthropics/agentsmesh/backend/internal/service/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/binding"
	blockstoreservice "github.com/anthropics/agentsmesh/backend/internal/service/blockstore"
	"github.com/anthropics/agentsmesh/backend/internal/service/channel"
	extensionservice "github.com/anthropics/agentsmesh/backend/internal/service/extension"
	fileservice "github.com/anthropics/agentsmesh/backend/internal/service/file"
	"github.com/anthropics/agentsmesh/backend/internal/service/geo"
	grantservice "github.com/anthropics/agentsmesh/backend/internal/service/grant"
	"github.com/anthropics/agentsmesh/backend/internal/service/invitation"
	"github.com/anthropics/agentsmesh/backend/internal/service/license"
	loop "github.com/anthropics/agentsmesh/backend/internal/service/loop"
	"github.com/anthropics/agentsmesh/backend/internal/service/mesh"
	notifService "github.com/anthropics/agentsmesh/backend/internal/service/notification"
	"github.com/anthropics/agentsmesh/backend/internal/service/organization"
	"github.com/anthropics/agentsmesh/backend/internal/service/promocode"
	"github.com/anthropics/agentsmesh/backend/internal/service/relay"
	"github.com/anthropics/agentsmesh/backend/internal/service/repository"
	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
	runnerlogservice "github.com/anthropics/agentsmesh/backend/internal/service/runnerlog"
	ssoservice "github.com/anthropics/agentsmesh/backend/internal/service/sso"
	supportticketservice "github.com/anthropics/agentsmesh/backend/internal/service/supportticket"
	"github.com/anthropics/agentsmesh/backend/internal/service/ticket"
	tokenusagesvc "github.com/anthropics/agentsmesh/backend/internal/service/tokenusage"
	"github.com/anthropics/agentsmesh/backend/internal/service/user"
)

type MessageService = agent.MessageService

type Services struct {
	Auth *auth.Service
	User *user.Service
	Org  *organization.Service
	AgentSvc           *agent.AgentService
	CredentialProfile  *agent.CredentialProfileService
	UserConfig         *agent.UserConfigService
	Repository         *repository.Service
	Webhook            *repository.WebhookService
	Runner             *runner.Service
	RunnerConnMgr      *runner.RunnerConnectionManager
	PodCoordinator     *runner.PodCoordinator
	Pod                *agentpod.PodService
	PodOrchestrator    *agentpod.PodOrchestrator
	Autopilot          *agentpod.AutopilotControllerService
	Channel            *channel.Service
	Binding            *binding.Service
	Ticket             *ticket.Service
	MRSync             *ticket.MRSyncService // MR sync for webhook events
	Mesh               *mesh.Service
	AgentPodSettings   *agentpod.SettingsService
	AgentPodAIProvider *agentpod.AIProviderService
	Billing            *billing.Service
	Message            *MessageService                  // Agent-to-agent messaging
	Hub                *websocket.Hub
	EventBus           *eventbus.EventBus
	Email              email.Service
	Invitation         *invitation.Service
	File               *fileservice.Service
	PromoCode          *promocode.Service
	License            *license.Service                 // License service for OnPremise
	APIKey             *apikeyservice.Service           // API key management for third-party access
	APIKeyAdapter      *apikeyservice.MiddlewareAdapter

	// gRPC/mTLS Runner registration handler (optional, only when PKI is enabled)
	GRPCRunnerHandler *GRPCRunnerHandler

	SandboxQueryService  *runner.SandboxQueryService

	UpgradeCommandSender runner.UpgradeCommandSender

	LogUploadSender  runner.LogUploadCommandSender
	LogUploadService *runnerlogservice.Service

	RelayManager        *relay.Manager
	RelayTokenGenerator *relay.TokenGenerator
	RelayDNSService     *relay.DNSService
	RelayACMEManager    *acme.Manager         // ACME certificate management for Relay TLS

	GeoResolver geo.Resolver

	VersionChecker *runner.VersionChecker

	Extension         *extensionservice.Service
	ExtensionRepo     extension.Repository
	MarketplaceWorker *extensionservice.MarketplaceWorker

	Loop             *loop.LoopService
	LoopRun          *loop.LoopRunService
	LoopOrchestrator *loop.LoopOrchestrator
	LoopScheduler    *loop.LoopScheduler

	SSO *ssoservice.Service

	SupportTicket *supportticketservice.Service

	NotificationPrefStore *notifService.PreferenceStore

	TokenUsage *tokenusagesvc.Service

	Grant *grantservice.Service

	Blockstore *blockstoreservice.Service
	Redis *redis.Client
}
