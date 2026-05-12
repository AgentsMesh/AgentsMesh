// Drift sample: tag=2 and tag=3 swapped vs bad.proto. The validator must
// detect this — a wire decode against this struct would decode `name`
// from bytes the .proto put at `limit`, silently corrupting both fields.
#[derive(Clone, PartialEq, prost::Message)]
pub struct Foo {
    #[prost(int64, tag = "1")]
    pub id: i64,
    #[prost(string, tag = "3")]
    pub name: String,
    #[prost(int32, optional, tag = "2")]
    pub limit: Option<i32>,
}
