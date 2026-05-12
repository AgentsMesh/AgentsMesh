#[derive(Clone, PartialEq, prost::Message)]
pub struct Foo {
    #[prost(int64, tag = "1")]
    pub id: i64,
    #[prost(string, tag = "2")]
    pub name: String,
    #[prost(int32, optional, tag = "3")]
    pub limit: Option<i32>,
}
