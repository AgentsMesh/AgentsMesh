// Hand-maintained `prost::Message` mirrors of `proto/notification/v1/notification.proto`.
// Tag numbers match the .proto byte-for-byte; `tools/validate_prost_tags`
// runs at build time to catch drift (watch list §8). NO
// `Serialize`/`Deserialize` derives — binary wire only (conventions §2.5, §3).
//
// ORG-SCOPED service (conventions §3.5): every request carries
// `org_slug = 1`. The auth interceptor + `ResolveOrgScope` helper in
// `interceptors/org_scope.go` consume the field on the server side.

use std::collections::HashMap;

#[derive(Clone, PartialEq, prost::Message)]
pub struct NotificationPreference {
    #[prost(string, tag = "1")]
    pub source: String,
    #[prost(string, optional, tag = "2")]
    pub entity_id: Option<String>,
    #[prost(bool, tag = "3")]
    pub is_muted: bool,
    #[prost(map = "string, bool", tag = "4")]
    pub channels: HashMap<String, bool>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct ListPreferencesRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct ListPreferencesResponse {
    #[prost(message, repeated, tag = "1")]
    pub items: Vec<NotificationPreference>,
    #[prost(int64, tag = "2")]
    pub total: i64,
    #[prost(int32, tag = "3")]
    pub limit: i32,
    #[prost(int32, tag = "4")]
    pub offset: i32,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct SetPreferenceRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(string, tag = "2")]
    pub source: String,
    #[prost(string, optional, tag = "3")]
    pub entity_id: Option<String>,
    #[prost(bool, tag = "4")]
    pub is_muted: bool,
    #[prost(map = "string, bool", tag = "5")]
    pub channels: HashMap<String, bool>,
}

#[cfg(test)]
mod tests {
    use super::*;
    use prost::Message;

    fn sample_channels() -> HashMap<String, bool> {
        let mut m = HashMap::new();
        m.insert("toast".into(), true);
        m.insert("browser".into(), false);
        m
    }

    #[test]
    fn notification_preference_round_trip() {
        let original = NotificationPreference {
            source: "channel:message".into(),
            entity_id: Some("42".into()),
            is_muted: false,
            channels: sample_channels(),
        };
        let bytes = original.encode_to_vec();
        let decoded = NotificationPreference::decode(&*bytes).unwrap();
        assert_eq!(original, decoded,
            "tag swap or transcription mistake would surface as field-value swap here");
        assert_eq!(decoded.channels.get("toast"), Some(&true));
        assert_eq!(decoded.channels.get("browser"), Some(&false));
    }

    #[test]
    fn notification_preference_absent_entity_id() {
        // Source-level preference: entity_id is absent. Round-trips as None.
        let original = NotificationPreference {
            source: "terminal:osc".into(),
            entity_id: None,
            is_muted: true,
            channels: HashMap::new(),
        };
        let bytes = original.encode_to_vec();
        let decoded = NotificationPreference::decode(&*bytes).unwrap();
        assert_eq!(original, decoded);
        assert_eq!(decoded.entity_id, None);
        assert!(decoded.is_muted);
        assert!(decoded.channels.is_empty());
    }

    #[test]
    fn notification_preference_absent_vs_empty_entity_id() {
        // None vs Some(String::new()) must serialize differently — proto3
        // `optional` distinguishes "absent" from "explicit empty value".
        let absent = NotificationPreference {
            source: "s".into(),
            entity_id: None,
            is_muted: false,
            channels: HashMap::new(),
        };
        let explicit_empty = NotificationPreference {
            source: "s".into(),
            entity_id: Some(String::new()),
            is_muted: false,
            channels: HashMap::new(),
        };
        assert_ne!(absent.encode_to_vec(), explicit_empty.encode_to_vec(),
            "absent vs explicit empty string must encode differently");

        let r1 = NotificationPreference::decode(&*absent.encode_to_vec()).unwrap();
        let r2 = NotificationPreference::decode(&*explicit_empty.encode_to_vec()).unwrap();
        assert_eq!(r1.entity_id, None);
        assert_eq!(r2.entity_id, Some(String::new()));
    }

    #[test]
    fn list_preferences_request_round_trip() {
        let original = ListPreferencesRequest {
            org_slug: "acme".into(),
        };
        let bytes = original.encode_to_vec();
        let decoded = ListPreferencesRequest::decode(&*bytes).unwrap();
        assert_eq!(original, decoded);
        assert_eq!(decoded.org_slug, "acme");
    }

    #[test]
    fn list_preferences_response_round_trip() {
        let original = ListPreferencesResponse {
            items: vec![
                NotificationPreference {
                    source: "channel:message".into(),
                    entity_id: Some("42".into()),
                    is_muted: false,
                    channels: sample_channels(),
                },
                NotificationPreference {
                    source: "terminal:osc".into(),
                    entity_id: None,
                    is_muted: true,
                    channels: HashMap::new(),
                },
            ],
            total: 2,
            limit: 20,
            offset: 0,
        };
        let bytes = original.encode_to_vec();
        let decoded = ListPreferencesResponse::decode(&*bytes).unwrap();
        assert_eq!(original, decoded);
        assert_eq!(decoded.items.len(), 2);
        assert_eq!(decoded.items[0].source, "channel:message");
        assert_eq!(decoded.items[1].source, "terminal:osc");
    }

    #[test]
    fn list_preferences_response_empty() {
        // No preferences configured yet — empty items but proto3 zero
        // fields all serialize identically. Decoding must still produce
        // an empty Vec, not a None.
        let original = ListPreferencesResponse {
            items: vec![],
            total: 0,
            limit: 0,
            offset: 0,
        };
        let bytes = original.encode_to_vec();
        let decoded = ListPreferencesResponse::decode(&*bytes).unwrap();
        assert_eq!(original, decoded);
        assert!(decoded.items.is_empty());
    }

    #[test]
    fn set_preference_request_round_trip() {
        let original = SetPreferenceRequest {
            org_slug: "acme".into(),
            source: "channel:mention".into(),
            entity_id: Some("99".into()),
            is_muted: true,
            channels: sample_channels(),
        };
        let bytes = original.encode_to_vec();
        let decoded = SetPreferenceRequest::decode(&*bytes).unwrap();
        assert_eq!(original, decoded);
        assert_eq!(decoded.source, "channel:mention");
        assert_eq!(decoded.entity_id, Some("99".into()));
    }

    #[test]
    fn set_preference_request_empty_channels_means_default() {
        // Empty channels map is the legitimate "use server defaults"
        // semantic — round-trip must preserve emptiness, not insert
        // anything.
        let original = SetPreferenceRequest {
            org_slug: "acme".into(),
            source: "channel:message".into(),
            entity_id: None,
            is_muted: false,
            channels: HashMap::new(),
        };
        let bytes = original.encode_to_vec();
        let decoded = SetPreferenceRequest::decode(&*bytes).unwrap();
        assert_eq!(original, decoded);
        assert!(decoded.channels.is_empty());
    }
}
