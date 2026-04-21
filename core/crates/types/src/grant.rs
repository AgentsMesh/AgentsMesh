use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ResourceGrantUserBrief {
    pub id: i64,
    pub email: String,
    pub username: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub name: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ResourceGrant {
    pub id: i64,
    pub resource_type: String,
    pub resource_id: String,
    pub user_id: i64,
    pub granted_by: i64,
    pub created_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub user: Option<ResourceGrantUserBrief>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub granted_by_user: Option<ResourceGrantUserBrief>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ResourceGrantListResponse {
    pub grants: Vec<ResourceGrant>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateResourceGrantRequest {
    pub user_id: i64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ResourceGrantResponse {
    pub grant: ResourceGrant,
}
