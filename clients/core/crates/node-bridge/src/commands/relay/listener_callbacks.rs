use std::sync::Arc;

use agentsmesh_protocol::MsgType;
use agentsmesh_relay::{
    GenerationAcpCallback, GenerationDisconnectCallback, GenerationStatusCallback, OutputCallback,
    RelayStatusInfo,
};
use napi::threadsafe_function::{ThreadsafeFunction, ThreadsafeFunctionCallMode};

pub(super) fn output(on_output: ThreadsafeFunction<Vec<u8>>) -> OutputCallback {
    let callback = Arc::new(on_output);
    output_with(move |data| {
        callback.call(Ok(data), ThreadsafeFunctionCallMode::NonBlocking);
    })
}

pub(super) fn bound(on_bound: ThreadsafeFunction<u32>) -> Arc<dyn Fn(u32) + Send + Sync> {
    bound_with(move |generation| {
        on_bound.call(Ok(generation), ThreadsafeFunctionCallMode::NonBlocking);
    })
}

pub(super) fn generation_status(on_status: ThreadsafeFunction<String>) -> GenerationStatusCallback {
    let callback = Arc::new(on_status);
    generation_status_with(move |json| {
        callback.call(Ok(json), ThreadsafeFunctionCallMode::NonBlocking);
    })
}

pub(super) fn generation_acp(on_acp: ThreadsafeFunction<String>) -> GenerationAcpCallback {
    let callback = Arc::new(on_acp);
    generation_acp_with(move |json| {
        callback.call(Ok(json), ThreadsafeFunctionCallMode::NonBlocking);
    })
}

pub(super) fn generation_disconnect(
    on_disconnect: ThreadsafeFunction<String>,
) -> GenerationDisconnectCallback {
    let callback = Arc::new(on_disconnect);
    generation_disconnect_with(move |json| {
        callback.call(Ok(json), ThreadsafeFunctionCallMode::NonBlocking);
    })
}

fn output_with(emit: impl Fn(Vec<u8>) + Send + Sync + 'static) -> OutputCallback {
    Arc::new(emit)
}

fn bound_with(emit: impl Fn(u32) + Send + Sync + 'static) -> Arc<dyn Fn(u32) + Send + Sync> {
    Arc::new(emit)
}

fn generation_status_with(
    emit: impl Fn(String) + Send + Sync + 'static,
) -> GenerationStatusCallback {
    Arc::new(move |generation, info: RelayStatusInfo| {
        let json = serde_json::json!({
            "generation": generation,
            "revision": info.revision,
            "status": info.status.to_string(),
            "runnerDisconnected": info.runner_disconnected,
        })
        .to_string();
        emit(json);
    })
}

fn generation_acp_with(emit: impl Fn(String) + Send + Sync + 'static) -> GenerationAcpCallback {
    Arc::new(move |generation, msg_type: MsgType, payload| {
        let json = serde_json::json!({
            "generation": generation,
            "msgType": msg_type as u8,
            "payload": payload,
        })
        .to_string();
        emit(json);
    })
}

fn generation_disconnect_with(
    emit: impl Fn(String) + Send + Sync + 'static,
) -> GenerationDisconnectCallback {
    Arc::new(move |pod_key, generation| {
        let json = serde_json::json!({
            "podKey": pod_key,
            "generation": generation,
        })
        .to_string();
        emit(json);
    })
}

#[cfg(test)]
mod tests {
    use std::sync::Mutex;

    use agentsmesh_relay::RelayStatus;

    use super::*;

    #[test]
    fn output_and_bound_adapters_preserve_values() {
        let output = Arc::new(Mutex::new(Vec::new()));
        let captured = Arc::clone(&output);
        output_with(move |data| captured.lock().unwrap().push(data))(vec![0, 1, 255]);
        assert_eq!(*output.lock().unwrap(), vec![vec![0, 1, 255]]);

        let bound = Arc::new(Mutex::new(Vec::new()));
        let captured = Arc::clone(&bound);
        bound_with(move |generation| captured.lock().unwrap().push(generation))(19);
        assert_eq!(*bound.lock().unwrap(), vec![19]);
    }

    #[test]
    fn generation_status_serializes_the_complete_ordering_contract() {
        let events = Arc::new(Mutex::new(Vec::new()));
        let captured = Arc::clone(&events);
        let callback = generation_status_with(move |json| captured.lock().unwrap().push(json));
        callback(
            7,
            RelayStatusInfo {
                status: RelayStatus::Connecting,
                runner_disconnected: true,
                revision: 23,
            },
        );

        let value: serde_json::Value = serde_json::from_str(&events.lock().unwrap()[0]).unwrap();
        assert_eq!(
            value,
            serde_json::json!({
                "generation": 7,
                "revision": 23,
                "status": "connecting",
                "runnerDisconnected": true,
            })
        );
    }

    #[test]
    fn generation_acp_and_disconnect_payloads_keep_their_generation() {
        let acp_events = Arc::new(Mutex::new(Vec::new()));
        let captured = Arc::clone(&acp_events);
        generation_acp_with(move |json| captured.lock().unwrap().push(json))(
            11,
            MsgType::AcpEvent,
            serde_json::json!({"event": "started"}),
        );
        let acp: serde_json::Value = serde_json::from_str(&acp_events.lock().unwrap()[0]).unwrap();
        assert_eq!(acp["generation"], 11);
        assert_eq!(acp["msgType"], MsgType::AcpEvent as u8);
        assert_eq!(acp["payload"], serde_json::json!({"event": "started"}));

        let disconnects = Arc::new(Mutex::new(Vec::new()));
        let captured = Arc::clone(&disconnects);
        generation_disconnect_with(move |json| captured.lock().unwrap().push(json))(
            "pod-9".to_string(),
            12,
        );
        let disconnect: serde_json::Value =
            serde_json::from_str(&disconnects.lock().unwrap()[0]).unwrap();
        assert_eq!(
            disconnect,
            serde_json::json!({"podKey": "pod-9", "generation": 12})
        );
    }
}
