// FFI service wrappers: thin UniFFI-exported adapters over `agentsmesh-services`.
// Each file mirrors `crates/wasm/src/service_*.rs` but returns strong-typed
// `Record`/`Enum` from `super::dto` instead of JSON strings.

mod automation;
mod automation_proto_convert;
mod billing;
mod blocks_mesh;
mod channel;
mod channel_messages;
mod channel_proto_convert;
mod loop_service;
mod misc;
mod pod;
mod repository;
mod runner;
mod sso;
mod ticket;
mod user;
