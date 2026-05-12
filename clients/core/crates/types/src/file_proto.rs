// Hand-maintained `prost::Message` mirrors of `proto/file/v1/file.proto`.
// Tag numbers match the .proto byte-for-byte; `tools/validate_prost_tags`
// runs at build time to catch drift. NO `Serialize` / `Deserialize` derives.

#[derive(Clone, PartialEq, prost::Message)]
pub struct PresignUploadRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(string, tag = "2")]
    pub filename: String,
    #[prost(string, tag = "3")]
    pub content_type: String,
    #[prost(int64, tag = "4")]
    pub size: i64,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct PresignUploadResponse {
    #[prost(string, tag = "1")]
    pub put_url: String,
    #[prost(string, tag = "2")]
    pub get_url: String,
}

#[cfg(test)]
mod tests {
    use super::*;
    use prost::Message;

    #[test]
    fn presign_upload_request_round_trip() {
        let req = PresignUploadRequest {
            org_slug: "acme".into(),
            filename: "image.png".into(),
            content_type: "image/png".into(),
            size: 12_345,
        };
        let bytes = req.encode_to_vec();
        assert_eq!(req, PresignUploadRequest::decode(&*bytes).unwrap());
    }

    #[test]
    fn presign_upload_response_round_trip() {
        let resp = PresignUploadResponse {
            put_url: "https://s3.example.com/put?sig=abc".into(),
            get_url: "https://s3.example.com/get?sig=def".into(),
        };
        let bytes = resp.encode_to_vec();
        assert_eq!(resp, PresignUploadResponse::decode(&*bytes).unwrap());
    }
}
