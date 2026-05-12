// Hand-maintained `prost::Message` mirrors of `proto/token_usage/v1/token_usage.proto`.
// Tag numbers match the .proto byte-for-byte; `tools/validate_prost_tags`
// runs at build time to catch drift. NO `Serialize` / `Deserialize` derives.

#[derive(Clone, PartialEq, prost::Message)]
pub struct GetDashboardRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(string, tag = "2")]
    pub start_time: String,
    #[prost(string, tag = "3")]
    pub end_time: String,
    #[prost(string, tag = "4")]
    pub granularity: String,
    #[prost(string, tag = "5")]
    pub agent_slug: String,
    #[prost(int64, optional, tag = "6")]
    pub user_id: Option<i64>,
    #[prost(string, tag = "7")]
    pub model: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct UsageSummary {
    #[prost(int64, tag = "1")]
    pub input_tokens: i64,
    #[prost(int64, tag = "2")]
    pub output_tokens: i64,
    #[prost(int64, tag = "3")]
    pub cache_creation_tokens: i64,
    #[prost(int64, tag = "4")]
    pub cache_read_tokens: i64,
    #[prost(int64, tag = "5")]
    pub total_tokens: i64,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct TimeSeriesPoint {
    #[prost(string, tag = "1")]
    pub period: String,
    #[prost(int64, tag = "2")]
    pub input_tokens: i64,
    #[prost(int64, tag = "3")]
    pub output_tokens: i64,
    #[prost(int64, tag = "4")]
    pub cache_creation_tokens: i64,
    #[prost(int64, tag = "5")]
    pub cache_read_tokens: i64,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct AgentUsage {
    #[prost(string, tag = "1")]
    pub agent_slug: String,
    #[prost(int64, tag = "2")]
    pub input_tokens: i64,
    #[prost(int64, tag = "3")]
    pub output_tokens: i64,
    #[prost(int64, tag = "4")]
    pub cache_creation_tokens: i64,
    #[prost(int64, tag = "5")]
    pub cache_read_tokens: i64,
    #[prost(int64, tag = "6")]
    pub total_tokens: i64,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct UserUsage {
    #[prost(int64, tag = "1")]
    pub user_id: i64,
    #[prost(string, tag = "2")]
    pub username: String,
    #[prost(string, tag = "3")]
    pub email: String,
    #[prost(int64, tag = "4")]
    pub input_tokens: i64,
    #[prost(int64, tag = "5")]
    pub output_tokens: i64,
    #[prost(int64, tag = "6")]
    pub cache_creation_tokens: i64,
    #[prost(int64, tag = "7")]
    pub cache_read_tokens: i64,
    #[prost(int64, tag = "8")]
    pub total_tokens: i64,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct ModelUsage {
    #[prost(string, tag = "1")]
    pub model: String,
    #[prost(int64, tag = "2")]
    pub input_tokens: i64,
    #[prost(int64, tag = "3")]
    pub output_tokens: i64,
    #[prost(int64, tag = "4")]
    pub cache_creation_tokens: i64,
    #[prost(int64, tag = "5")]
    pub cache_read_tokens: i64,
    #[prost(int64, tag = "6")]
    pub total_tokens: i64,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct GetDashboardResponse {
    #[prost(message, optional, tag = "1")]
    pub summary: Option<UsageSummary>,
    #[prost(message, repeated, tag = "2")]
    pub time_series: Vec<TimeSeriesPoint>,
    #[prost(message, repeated, tag = "3")]
    pub by_agent: Vec<AgentUsage>,
    #[prost(message, repeated, tag = "4")]
    pub by_user: Vec<UserUsage>,
    #[prost(message, repeated, tag = "5")]
    pub by_model: Vec<ModelUsage>,
}

#[cfg(test)]
mod tests {
    use super::*;
    use prost::Message;

    #[test]
    fn dashboard_request_round_trip() {
        let req = GetDashboardRequest {
            org_slug: "acme".into(),
            start_time: "2026-04-01T00:00:00Z".into(),
            end_time: "2026-05-01T00:00:00Z".into(),
            granularity: "day".into(),
            agent_slug: "claude-code".into(),
            user_id: Some(7),
            model: "claude-opus-4-7".into(),
        };
        let bytes = req.encode_to_vec();
        assert_eq!(req, GetDashboardRequest::decode(&*bytes).unwrap());
    }

    #[test]
    fn dashboard_response_round_trip() {
        let resp = GetDashboardResponse {
            summary: Some(UsageSummary {
                input_tokens: 1000,
                output_tokens: 500,
                cache_creation_tokens: 100,
                cache_read_tokens: 50,
                total_tokens: 1650,
            }),
            time_series: vec![TimeSeriesPoint {
                period: "2026-05-01T00:00:00Z".into(),
                input_tokens: 100,
                output_tokens: 50,
                cache_creation_tokens: 10,
                cache_read_tokens: 5,
            }],
            by_agent: vec![AgentUsage {
                agent_slug: "claude-code".into(),
                input_tokens: 500,
                output_tokens: 250,
                cache_creation_tokens: 50,
                cache_read_tokens: 25,
                total_tokens: 825,
            }],
            by_user: vec![UserUsage {
                user_id: 7,
                username: "alice".into(),
                email: "alice@example.com".into(),
                input_tokens: 300,
                output_tokens: 150,
                cache_creation_tokens: 30,
                cache_read_tokens: 15,
                total_tokens: 495,
            }],
            by_model: vec![ModelUsage {
                model: "claude-opus-4-7".into(),
                input_tokens: 800,
                output_tokens: 400,
                cache_creation_tokens: 80,
                cache_read_tokens: 40,
                total_tokens: 1320,
            }],
        };
        let bytes = resp.encode_to_vec();
        let decoded = GetDashboardResponse::decode(&*bytes).unwrap();
        assert_eq!(resp, decoded);
        assert_eq!(decoded.summary.as_ref().unwrap().total_tokens, 1650);
        assert_eq!(decoded.time_series.len(), 1);
    }

    #[test]
    fn user_id_optional_zero_distinguishable_from_absent() {
        let with_zero = GetDashboardRequest {
            user_id: Some(0),
            ..Default::default()
        };
        let absent = GetDashboardRequest {
            user_id: None,
            ..Default::default()
        };
        assert_ne!(with_zero.encode_to_vec(), absent.encode_to_vec());
    }
}
