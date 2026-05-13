// DTO layer: UniFFI `Record` / `Enum` definitions that map from internal
// `agentsmesh-types` structs to FFI-friendly shapes (no HashMap, no
// chrono, no lifetimes). Populated per service migration in Phase 1.
//
// Re-exports are kept broad so the generated Swift surface contains every
// named record; unused warnings are suppressed because uniffi consumes the
// types via `#[uniffi::export]` in service files, not via direct imports.
#![allow(unused_imports)]

mod automation;
mod billing_dto;
mod blocks_mesh;
mod channel;
mod message;
mod misc;
mod pod;
mod repository_dto;
mod runner_dto;
mod ticket;
mod user;

pub use automation::{
    AutopilotControllerDto, AutopilotIterationDto, AutopilotIterationListResponseDto,
    AutopilotListResponseDto, AutopilotStatusDto, CreateAutopilotRequestDto, CreateLoopRequestDto,
    LoopDataDto, LoopListResponseDto, LoopRunDataDto, LoopRunListResponseDto, LoopRunStatusDto,
    UpdateLoopRequestDto,
};
pub(crate) use automation::approve_autopilot_req;
pub use billing_dto::{
    BillingOverviewDto, ChangeBillingCycleRequestDto, CheckoutStatusDto,
    CreateCheckoutRequestDto, CreateCheckoutResponseDto, CreateSubscriptionRequestDto,
    DeploymentInfoDto, InvoiceDto, InvoiceListResponseDto, PlanListResponseDto,
    PublicPlanPricingDto, PublicPricingResponseDto, SeatUsageDto, SubscriptionDto,
    SubscriptionPlanDto, UpdateSubscriptionRequestDto, UpgradeSubscriptionRequestDto,
    UsageOverviewDto,
};
pub(crate) use billing_dto::{invoice_list_from_proto, plan_list_from_proto};
pub use blocks_mesh::{
    ActorTypeDto, ApplyOpsRequestDto, ApplyOpsResultDto, BlockDto, BlockOpDto, BlockRefDto,
    ChildrenResultDto, MeshChannelInfoDto, MeshEdgeDto, MeshNodeDto, MeshRunnerInfoDto,
    MeshTopologyDto, NotificationPreferenceDto, NotificationPreferenceListResponseDto, OpEnvelopeDto,
    OpKindDto, SearchHitDto, SemanticSearchRequestDto, SetNotificationPreferenceRequestDto,
    WorkspaceDto,
};
pub use misc::{
    InvitationDto, InvitationListResponseDto, PresignRequestDto, PresignResponseDto,
    ResourceGrantDto, ResourceGrantListResponseDto, ResourceGrantResponseDto,
    ResourceGrantUserBriefDto,
};
pub(crate) use misc::{create_invitation_req, create_resource_grant_req};

pub use channel::{
    ChannelDto, ChannelListResponseDto, ChannelMemberDto, ChannelMemberListResponseDto,
    ChannelMessageDto, ChannelMessageListResponseDto, ChannelUnreadResponseDto,
    CreateChannelRequestDto, MessagePreviewDto, SenderAgentInfoDto, SenderPodInfoDto,
    UpdateChannelRequestDto,
};
pub use message::{
    DeadLetterEntryDto, DeadLetterListResponseDto, DirectMessageDto, DirectMessageListResponseDto,
    ReplayDeadLetterResponseDto, SendDirectMessageRequestDto, UnreadCountResponseDto,
};
pub(crate) use message::mark_messages_read_req;
pub use ticket::{
    BoardColumnDto, BoardResponseDto, CreateLabelRequestDto, CreateTicketCommentRequestDto,
    CreateTicketRelationRequestDto, CreateTicketRequestDto, LabelDto, LabelListResponseDto,
    LinkTicketCommitRequestDto, TicketCommentDto, TicketCommentListResponseDto, TicketCommitDto,
    TicketCommitListResponseDto, TicketDto, TicketListResponseDto, TicketPriorityDto,
    TicketRelationDto, TicketRelationListResponseDto, TicketStatusDto, UpdateLabelRequestDto,
    UpdateTicketCommentRequestDto, UpdateTicketRequestDto,
};
pub(crate) use ticket::{add_assignee_req, add_ticket_label_req, update_ticket_status_req};

pub use pod::{
    CreatePodRequestDto, CreatePodResponseDto, PodAgentInfoDto, PodConnectionInfoDto,
    PodCreatedByInfoDto, PodDto, PodListResponseDto, PodLoopInfoDto, PodRepositoryInfoDto,
    PodRunnerInfoDto, PodStatusDto, PodTicketInfoDto,
};
pub(crate) use pod::build_create_pod_proto_request;
pub use user::{
    AuthSessionDto, AuthTokensDto, BootstrapCleanupReasonDto, BootstrapResultDto,
    OrganizationDto, SSOConfigDto, UserDto, UserIdentityDto,
};
pub use repository_dto::{
    BranchDto, CreateRepositoryRequestDto, MergeRequestListResponseDto, RepositoryDto,
    RepositoryListResponseDto, RepositoryMergeRequestDto, UpdateRepositoryRequestDto,
    WebhookSecretDto, WebhookStatusDto,
};
pub(crate) use repository_dto::{
    build_create_repository_proto_request, build_update_repository_proto_request,
    list_branches_from_proto, merge_request_list_from_proto, repository_list_from_proto,
};
pub use runner_dto::{
    AuthorizeRunnerRequestDto, CreateRunnerTokenRequestDto, GrpcRegistrationTokenDto,
    RunnerAuthStatusDto, RunnerDto, RunnerListResponseDto, RunnerLogDto, RunnerLogListResponseDto,
    RunnerStatusDto, RunnerTokenListResponseDto, UpdateRunnerRequestDto, UpgradeRunnerRequestDto,
};
pub(crate) use runner_dto::{
    runner_list_from_proto, runner_log_list_from_proto, runner_token_list_from_proto,
};
