use crate::ApiClient;
use crate::connect_call::connect_call;
use crate::error::ApiError;
use agentsmesh_types::proto_loop_v1 as lp;

// =============================================================================
// Connect-RPC (binary wire). See proto-naming-conventions.md §2.5.
// =============================================================================

impl ApiClient {
    pub async fn list_loops_connect(
        &self,
        req: &lp::ListLoopsRequest,
    ) -> Result<lp::ListLoopsResponse, ApiError> {
        connect_call(self, "/proto.loop.v1.LoopService/ListLoops", req).await
    }

    pub async fn get_loop_connect(
        &self,
        req: &lp::GetLoopRequest,
    ) -> Result<lp::Loop, ApiError> {
        connect_call(self, "/proto.loop.v1.LoopService/GetLoop", req).await
    }

    pub async fn create_loop_connect(
        &self,
        req: &lp::CreateLoopRequest,
    ) -> Result<lp::Loop, ApiError> {
        connect_call(self, "/proto.loop.v1.LoopService/CreateLoop", req).await
    }

    pub async fn update_loop_connect(
        &self,
        req: &lp::UpdateLoopRequest,
    ) -> Result<lp::Loop, ApiError> {
        connect_call(self, "/proto.loop.v1.LoopService/UpdateLoop", req).await
    }

    pub async fn delete_loop_connect(
        &self,
        req: &lp::DeleteLoopRequest,
    ) -> Result<lp::DeleteLoopResponse, ApiError> {
        connect_call(self, "/proto.loop.v1.LoopService/DeleteLoop", req).await
    }

    pub async fn enable_loop_connect(
        &self,
        req: &lp::LoopActionRequest,
    ) -> Result<lp::Loop, ApiError> {
        connect_call(self, "/proto.loop.v1.LoopService/EnableLoop", req).await
    }

    pub async fn disable_loop_connect(
        &self,
        req: &lp::LoopActionRequest,
    ) -> Result<lp::Loop, ApiError> {
        connect_call(self, "/proto.loop.v1.LoopService/DisableLoop", req).await
    }

    pub async fn trigger_loop_connect(
        &self,
        req: &lp::TriggerLoopRequest,
    ) -> Result<lp::TriggerLoopResponse, ApiError> {
        connect_call(self, "/proto.loop.v1.LoopService/TriggerLoop", req).await
    }

    pub async fn list_loop_runs_connect(
        &self,
        req: &lp::ListRunsRequest,
    ) -> Result<lp::ListRunsResponse, ApiError> {
        connect_call(self, "/proto.loop.v1.LoopService/ListRuns", req).await
    }

    pub async fn cancel_loop_run_connect(
        &self,
        req: &lp::CancelRunRequest,
    ) -> Result<lp::CancelRunResponse, ApiError> {
        connect_call(self, "/proto.loop.v1.LoopService/CancelRun", req).await
    }
}
