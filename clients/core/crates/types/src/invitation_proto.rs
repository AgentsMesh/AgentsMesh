// Hand-maintained `prost::Message` mirrors of `proto/invitation/v1/invitation.proto`.
// Tag numbers match the .proto byte-for-byte; `tools/validate_prost_tags`
// runs at build time to catch drift (watch list §8). NO `Serialize` /
// `Deserialize` derives — binary wire only (conventions §2.5, §3).

// ----- Entities -----

#[derive(Clone, PartialEq, prost::Message)]
pub struct Invitation {
    #[prost(int64, tag = "1")]
    pub id: i64,
    #[prost(int64, tag = "2")]
    pub organization_id: i64,
    #[prost(string, tag = "3")]
    pub email: String,
    #[prost(string, tag = "4")]
    pub role: String,
    #[prost(int64, tag = "5")]
    pub invited_by: i64,
    #[prost(string, tag = "6")]
    pub expires_at: String,
    #[prost(string, optional, tag = "7")]
    pub accepted_at: Option<String>,
    #[prost(string, tag = "8")]
    pub created_at: String,
    #[prost(string, tag = "9")]
    pub updated_at: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct InvitationInfo {
    #[prost(int64, tag = "1")]
    pub id: i64,
    #[prost(string, tag = "2")]
    pub email: String,
    #[prost(string, tag = "3")]
    pub role: String,
    #[prost(int64, tag = "4")]
    pub organization_id: i64,
    #[prost(string, tag = "5")]
    pub organization_name: String,
    #[prost(string, tag = "6")]
    pub organization_slug: String,
    #[prost(string, tag = "7")]
    pub inviter_name: String,
    #[prost(string, tag = "8")]
    pub expires_at: String,
    #[prost(bool, tag = "9")]
    pub is_expired: bool,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct PendingInvitation {
    #[prost(int64, tag = "1")]
    pub id: i64,
    #[prost(int64, tag = "2")]
    pub organization_id: i64,
    #[prost(string, tag = "3")]
    pub organization_name: String,
    #[prost(string, tag = "4")]
    pub organization_slug: String,
    #[prost(string, tag = "5")]
    pub role: String,
    #[prost(string, tag = "6")]
    pub expires_at: String,
    #[prost(string, tag = "7")]
    pub token: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct AcceptedOrgInfo {
    #[prost(int64, tag = "1")]
    pub id: i64,
    #[prost(string, tag = "2")]
    pub name: String,
    #[prost(string, tag = "3")]
    pub slug: String,
}

// ----- Org-scoped Requests/Responses -----

#[derive(Clone, PartialEq, prost::Message)]
pub struct ListInvitationsRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(int32, optional, tag = "2")]
    pub offset: Option<i32>,
    #[prost(int32, optional, tag = "3")]
    pub limit: Option<i32>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct ListInvitationsResponse {
    #[prost(message, repeated, tag = "1")]
    pub items: Vec<Invitation>,
    #[prost(int64, tag = "2")]
    pub total: i64,
    #[prost(int32, tag = "3")]
    pub limit: i32,
    #[prost(int32, tag = "4")]
    pub offset: i32,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct CreateInvitationRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(string, tag = "2")]
    pub email: String,
    #[prost(string, tag = "3")]
    pub role: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct RevokeInvitationRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(int64, tag = "2")]
    pub id: i64,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct RevokeInvitationResponse {
    #[prost(string, tag = "1")]
    pub message: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct ResendInvitationRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(int64, tag = "2")]
    pub id: i64,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct ResendInvitationResponse {
    #[prost(string, tag = "1")]
    pub message: String,
}

// ----- User-scoped Requests/Responses (no org_slug; conventions §3.5 #1) -----

#[derive(Clone, PartialEq, prost::Message)]
pub struct AcceptInvitationRequest {
    #[prost(string, tag = "1")]
    pub token: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct AcceptInvitationResponse {
    #[prost(string, tag = "1")]
    pub message: String,
    #[prost(message, optional, tag = "2")]
    pub organization: Option<AcceptedOrgInfo>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct ListPendingInvitationsRequest {}

#[derive(Clone, PartialEq, prost::Message)]
pub struct ListPendingInvitationsResponse {
    #[prost(message, repeated, tag = "1")]
    pub items: Vec<PendingInvitation>,
    #[prost(int64, tag = "2")]
    pub total: i64,
    #[prost(int32, tag = "3")]
    pub limit: i32,
    #[prost(int32, tag = "4")]
    pub offset: i32,
}

// ----- Public Requests/Responses (no auth, no org_slug) -----

#[derive(Clone, PartialEq, prost::Message)]
pub struct GetInvitationByTokenRequest {
    #[prost(string, tag = "1")]
    pub token: String,
}

#[cfg(test)]
mod tests {
    use super::*;
    use prost::Message;

    fn sample_invitation() -> Invitation {
        Invitation {
            id: 42,
            organization_id: 7,
            email: "alice@example.com".into(),
            role: "admin".into(),
            invited_by: 100,
            expires_at: "2026-05-19T13:16:10Z".into(),
            accepted_at: None,
            created_at: "2026-05-12T13:16:10Z".into(),
            updated_at: "2026-05-12T13:16:10Z".into(),
        }
    }

    fn sample_pending_invitation() -> PendingInvitation {
        PendingInvitation {
            id: 42,
            organization_id: 7,
            organization_name: "Acme".into(),
            organization_slug: "acme".into(),
            role: "admin".into(),
            expires_at: "2026-05-19T13:16:10Z".into(),
            token: "abc123def456".into(),
        }
    }

    #[test]
    fn invitation_round_trip_preserves_every_field() {
        let original = sample_invitation();
        let bytes = original.encode_to_vec();
        let decoded = Invitation::decode(&*bytes).unwrap();
        assert_eq!(original, decoded,
            "tag swap or transcription mistake would surface as field-value swap here");
    }

    #[test]
    fn invitation_with_accepted_at_round_trips() {
        let original = Invitation {
            accepted_at: Some("2026-05-13T00:00:00Z".into()),
            ..sample_invitation()
        };
        let bytes = original.encode_to_vec();
        let decoded = Invitation::decode(&*bytes).unwrap();
        assert_eq!(original, decoded);
        assert_eq!(decoded.accepted_at.as_deref(), Some("2026-05-13T00:00:00Z"));
    }

    #[test]
    fn invitation_info_round_trip() {
        let original = InvitationInfo {
            id: 42,
            email: "alice@example.com".into(),
            role: "member".into(),
            organization_id: 7,
            organization_name: "Acme".into(),
            organization_slug: "acme".into(),
            inviter_name: "Bob Smith".into(),
            expires_at: "2026-05-19T13:16:10Z".into(),
            is_expired: false,
        };
        let bytes = original.encode_to_vec();
        let decoded = InvitationInfo::decode(&*bytes).unwrap();
        assert_eq!(original, decoded);
    }

    #[test]
    fn invitation_info_expired_flag_round_trips() {
        let expired = InvitationInfo {
            id: 1,
            email: "old@example.com".into(),
            role: "member".into(),
            organization_id: 7,
            organization_name: "Acme".into(),
            organization_slug: "acme".into(),
            inviter_name: "Bob".into(),
            expires_at: "2026-04-01T00:00:00Z".into(),
            is_expired: true,
        };
        let bytes = expired.encode_to_vec();
        let decoded = InvitationInfo::decode(&*bytes).unwrap();
        assert_eq!(expired, decoded);
        assert!(decoded.is_expired);
    }

    #[test]
    fn pending_invitation_round_trip_includes_token() {
        let original = sample_pending_invitation();
        let bytes = original.encode_to_vec();
        let decoded = PendingInvitation::decode(&*bytes).unwrap();
        assert_eq!(original, decoded);
        assert_eq!(decoded.token, "abc123def456");
    }

    #[test]
    fn accepted_org_info_round_trip() {
        let original = AcceptedOrgInfo {
            id: 7,
            name: "Acme".into(),
            slug: "acme".into(),
        };
        let bytes = original.encode_to_vec();
        let decoded = AcceptedOrgInfo::decode(&*bytes).unwrap();
        assert_eq!(original, decoded);
    }

    #[test]
    fn list_invitations_response_round_trip() {
        let original = ListInvitationsResponse {
            items: vec![sample_invitation()],
            total: 1,
            limit: 20,
            offset: 0,
        };
        let bytes = original.encode_to_vec();
        let decoded = ListInvitationsResponse::decode(&*bytes).unwrap();
        assert_eq!(original, decoded);
        assert_eq!(decoded.items.len(), 1);
    }

    #[test]
    fn list_pending_invitations_response_round_trip() {
        let original = ListPendingInvitationsResponse {
            items: vec![sample_pending_invitation()],
            total: 1,
            limit: 50,
            offset: 0,
        };
        let bytes = original.encode_to_vec();
        let decoded = ListPendingInvitationsResponse::decode(&*bytes).unwrap();
        assert_eq!(original, decoded);
    }

    #[test]
    fn create_invitation_request_round_trip() {
        let original = CreateInvitationRequest {
            org_slug: "acme".into(),
            email: "bob@example.com".into(),
            role: "member".into(),
        };
        let bytes = original.encode_to_vec();
        let decoded = CreateInvitationRequest::decode(&*bytes).unwrap();
        assert_eq!(original, decoded);
    }

    #[test]
    fn accept_invitation_response_with_org_round_trips() {
        let original = AcceptInvitationResponse {
            message: "Successfully joined the organization".into(),
            organization: Some(AcceptedOrgInfo {
                id: 7,
                name: "Acme".into(),
                slug: "acme".into(),
            }),
        };
        let bytes = original.encode_to_vec();
        let decoded = AcceptInvitationResponse::decode(&*bytes).unwrap();
        assert_eq!(original, decoded);
        let nested = decoded.organization.unwrap();
        assert_eq!(nested.slug, "acme");
    }

    #[test]
    fn list_invitations_optional_offset_zero_distinguishable_from_absent() {
        let with_zero = ListInvitationsRequest {
            org_slug: "acme".into(),
            offset: Some(0),
            limit: None,
        };
        let absent = ListInvitationsRequest {
            org_slug: "acme".into(),
            offset: None,
            limit: None,
        };
        assert_ne!(with_zero.encode_to_vec(), absent.encode_to_vec(),
            "explicit zero must encode different bytes from absent field");
        let r1 = ListInvitationsRequest::decode(&*with_zero.encode_to_vec()).unwrap();
        let r2 = ListInvitationsRequest::decode(&*absent.encode_to_vec()).unwrap();
        assert_eq!(r1.offset, Some(0));
        assert_eq!(r2.offset, None);
    }

    #[test]
    fn revoke_invitation_response_round_trip() {
        let req = RevokeInvitationRequest { org_slug: "acme".into(), id: 42 };
        let bytes = req.encode_to_vec();
        assert_eq!(req, RevokeInvitationRequest::decode(&*bytes).unwrap());

        let resp = RevokeInvitationResponse { message: "Invitation revoked successfully".into() };
        let resp_bytes = resp.encode_to_vec();
        assert_eq!(resp, RevokeInvitationResponse::decode(&*resp_bytes).unwrap());
    }

    #[test]
    fn resend_invitation_request_response_round_trip() {
        let req = ResendInvitationRequest { org_slug: "acme".into(), id: 42 };
        let bytes = req.encode_to_vec();
        assert_eq!(req, ResendInvitationRequest::decode(&*bytes).unwrap());

        let resp = ResendInvitationResponse { message: "Invitation resent successfully".into() };
        let resp_bytes = resp.encode_to_vec();
        assert_eq!(resp, ResendInvitationResponse::decode(&*resp_bytes).unwrap());
    }

    #[test]
    fn accept_invitation_request_round_trip() {
        let req = AcceptInvitationRequest { token: "abc123def456".into() };
        let bytes = req.encode_to_vec();
        assert_eq!(req, AcceptInvitationRequest::decode(&*bytes).unwrap());
    }

    #[test]
    fn get_invitation_by_token_request_round_trip() {
        let req = GetInvitationByTokenRequest { token: "abc123def456".into() };
        let bytes = req.encode_to_vec();
        assert_eq!(req, GetInvitationByTokenRequest::decode(&*bytes).unwrap());
    }

    #[test]
    fn list_pending_invitations_request_is_empty_on_wire() {
        let req = ListPendingInvitationsRequest {};
        let bytes = req.encode_to_vec();
        assert!(bytes.is_empty(), "user-scoped list-pending has no payload");
        assert_eq!(req, ListPendingInvitationsRequest::decode(&*bytes).unwrap());
    }
}
