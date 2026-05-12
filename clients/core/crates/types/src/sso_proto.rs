// Hand-maintained `prost::Message` mirrors of `proto/sso/v1/sso.proto`.
// Tag numbers match the .proto byte-for-byte; `tools/validate_prost_tags`
// runs at build time to catch drift (watch list §8). NO
// `Serialize`/`Deserialize` derives — binary wire only (conventions §2.5, §3).
//
// USER-SCOPED + PUBLIC service (conventions §3.5 exception #1): no
// `org_slug` field. The caller is unauthenticated by definition (this
// is the login surface that issues the token).

#[derive(Clone, PartialEq, prost::Message)]
pub struct SsoDiscoverConfig {
    #[prost(string, tag = "1")]
    pub domain: String,
    #[prost(string, tag = "2")]
    pub name: String,
    #[prost(string, tag = "3")]
    pub protocol: String,
    #[prost(bool, tag = "4")]
    pub enforce_sso: bool,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct DiscoverRequest {
    #[prost(string, tag = "1")]
    pub email: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct DiscoverResponse {
    #[prost(message, repeated, tag = "1")]
    pub items: Vec<SsoDiscoverConfig>,
    #[prost(int64, tag = "2")]
    pub total: i64,
    #[prost(int32, tag = "3")]
    pub limit: i32,
    #[prost(int32, tag = "4")]
    pub offset: i32,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct LdapAuthRequest {
    #[prost(string, tag = "1")]
    pub domain: String,
    #[prost(string, tag = "2")]
    pub username: String,
    #[prost(string, tag = "3")]
    pub password: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct LdapAuthUser {
    #[prost(int64, tag = "1")]
    pub id: i64,
    #[prost(string, tag = "2")]
    pub email: String,
    #[prost(string, tag = "3")]
    pub username: String,
    #[prost(string, optional, tag = "4")]
    pub name: Option<String>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct LdapAuthResponse {
    #[prost(string, tag = "1")]
    pub token: String,
    #[prost(string, tag = "2")]
    pub refresh_token: String,
    #[prost(string, tag = "3")]
    pub expires_at: String,
    #[prost(string, tag = "4")]
    pub token_type: String,
    #[prost(message, optional, tag = "5")]
    pub user: Option<LdapAuthUser>,
}

#[cfg(test)]
mod tests {
    use super::*;
    use prost::Message;

    fn sample_config() -> SsoDiscoverConfig {
        SsoDiscoverConfig {
            domain: "acme.com".into(),
            name: "ACME Corp SSO".into(),
            protocol: "oidc".into(),
            enforce_sso: true,
        }
    }

    fn sample_user() -> LdapAuthUser {
        LdapAuthUser {
            id: 42,
            email: "alice@acme.com".into(),
            username: "alice".into(),
            name: Some("Alice Anderson".into()),
        }
    }

    #[test]
    fn sso_discover_config_round_trip() {
        let original = sample_config();
        let bytes = original.encode_to_vec();
        let decoded = SsoDiscoverConfig::decode(&*bytes).unwrap();
        assert_eq!(original, decoded,
            "tag swap or transcription mistake would surface as field-value swap here");
    }

    #[test]
    fn discover_request_round_trip() {
        let original = DiscoverRequest { email: "alice@acme.com".into() };
        let bytes = original.encode_to_vec();
        let decoded = DiscoverRequest::decode(&*bytes).unwrap();
        assert_eq!(original, decoded);
        assert_eq!(decoded.email, "alice@acme.com");
    }

    #[test]
    fn discover_response_round_trip_with_items() {
        let original = DiscoverResponse {
            items: vec![sample_config(), SsoDiscoverConfig {
                domain: "acme.com".into(),
                name: "ACME LDAP".into(),
                protocol: "ldap".into(),
                enforce_sso: false,
            }],
            total: 2,
            limit: 20,
            offset: 0,
        };
        let bytes = original.encode_to_vec();
        let decoded = DiscoverResponse::decode(&*bytes).unwrap();
        assert_eq!(original, decoded);
        assert_eq!(decoded.items.len(), 2);
        assert_eq!(decoded.items[0].protocol, "oidc");
        assert_eq!(decoded.items[1].protocol, "ldap");
    }

    #[test]
    fn discover_response_empty_round_trip() {
        // No SSO configured for the email's domain — return empty items.
        let original = DiscoverResponse {
            items: vec![],
            total: 0,
            limit: 0,
            offset: 0,
        };
        let bytes = original.encode_to_vec();
        let decoded = DiscoverResponse::decode(&*bytes).unwrap();
        assert_eq!(original, decoded);
        assert!(decoded.items.is_empty());
    }

    #[test]
    fn ldap_auth_request_round_trip() {
        let original = LdapAuthRequest {
            domain: "acme.com".into(),
            username: "alice".into(),
            password: "s3cr3t".into(),
        };
        let bytes = original.encode_to_vec();
        let decoded = LdapAuthRequest::decode(&*bytes).unwrap();
        assert_eq!(original, decoded);
    }

    #[test]
    fn ldap_auth_user_round_trip_with_name() {
        let original = sample_user();
        let bytes = original.encode_to_vec();
        let decoded = LdapAuthUser::decode(&*bytes).unwrap();
        assert_eq!(original, decoded);
        assert_eq!(decoded.name, Some("Alice Anderson".into()));
    }

    #[test]
    fn ldap_auth_user_round_trip_without_name() {
        // Backend omits `name` if the LDAP record has no displayName attribute.
        let original = LdapAuthUser {
            id: 42,
            email: "alice@acme.com".into(),
            username: "alice".into(),
            name: None,
        };
        let bytes = original.encode_to_vec();
        let decoded = LdapAuthUser::decode(&*bytes).unwrap();
        assert_eq!(original, decoded);
        assert_eq!(decoded.name, None);
    }

    #[test]
    fn ldap_auth_response_round_trip() {
        let original = LdapAuthResponse {
            token: "access-jwt".into(),
            refresh_token: "refresh-jwt".into(),
            expires_at: "2026-05-13T13:16:10Z".into(),
            token_type: "Bearer".into(),
            user: Some(sample_user()),
        };
        let bytes = original.encode_to_vec();
        let decoded = LdapAuthResponse::decode(&*bytes).unwrap();
        assert_eq!(original, decoded);
        assert_eq!(decoded.token, "access-jwt");
        assert_eq!(decoded.refresh_token, "refresh-jwt");
        assert_eq!(decoded.expires_at, "2026-05-13T13:16:10Z");
        assert!(decoded.user.is_some());
    }

    // Pinned by conventions §3: prost's `Option<T>` distinguishes
    // "absent" from "explicit empty value". For LdapAuthUser.name, a
    // response that omits the field must decode back to None.
    #[test]
    fn optional_name_distinguishable_from_empty() {
        let absent = LdapAuthUser {
            id: 1,
            email: "a@b.com".into(),
            username: "u".into(),
            name: None,
        };
        let explicit_empty = LdapAuthUser {
            id: 1,
            email: "a@b.com".into(),
            username: "u".into(),
            name: Some(String::new()),
        };
        assert_ne!(absent.encode_to_vec(), explicit_empty.encode_to_vec(),
            "absent vs explicit empty string must encode differently");

        let r1 = LdapAuthUser::decode(&*absent.encode_to_vec()).unwrap();
        let r2 = LdapAuthUser::decode(&*explicit_empty.encode_to_vec()).unwrap();
        assert_eq!(r1.name, None);
        assert_eq!(r2.name, Some(String::new()));
    }
}
