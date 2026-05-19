use std::sync::Arc;

use js_sys::Uint8Array;
use wasm_bindgen::JsValue;

/// SAFETY: WASM is single-threaded, so the `Send + Sync` markers are
/// trivially safe.
#[derive(Clone)]
pub(crate) struct JsFunction(pub js_sys::Function);

unsafe impl Send for JsFunction {}
unsafe impl Sync for JsFunction {}

impl JsFunction {
    pub fn call1(&self, arg: &JsValue) {
        let _ = self.0.call1(&JsValue::NULL, arg);
    }

    pub fn call2(&self, a: &JsValue, b: &JsValue) {
        let _ = self.0.call2(&JsValue::NULL, a, b);
    }
}

fn to_js(val: &serde_json::Value) -> JsValue {
    match serde_json::to_string(val) {
        Ok(s) => js_sys::JSON::parse(&s).unwrap_or(JsValue::NULL),
        Err(_) => JsValue::NULL,
    }
}

pub(crate) fn make_output_callback(
    f: js_sys::Function,
) -> agentsmesh_relay::OutputCallback {
    let f = JsFunction(f);
    Arc::new(move |data: Vec<u8>| {
        let arr = Uint8Array::from(data.as_slice());
        f.call1(&arr.into());
    })
}

pub(crate) fn make_status_callback(
    f: js_sys::Function,
) -> agentsmesh_relay::StatusCallback {
    let f = JsFunction(f);
    Arc::new(move |info: agentsmesh_relay::RelayStatusInfo| {
        let obj = js_sys::Object::new();
        let _ = js_sys::Reflect::set(
            &obj,
            &"status".into(),
            &info.status.to_string().into(),
        );
        let _ = js_sys::Reflect::set(
            &obj,
            &"runnerDisconnected".into(),
            &info.runner_disconnected.into(),
        );
        f.call1(&obj.into());
    })
}

pub(crate) fn make_acp_callback(
    f: js_sys::Function,
) -> agentsmesh_relay::AcpCallback {
    let f = JsFunction(f);
    Arc::new(
        move |msg_type: agentsmesh_protocol::MsgType, payload: serde_json::Value| {
            let mt = JsValue::from(msg_type as u8);
            let pl = to_js(&payload);
            f.call2(&mt, &pl);
        },
    )
}

pub(crate) fn make_event_handler(
    f: js_sys::Function,
) -> agentsmesh_events::EventHandler {
    let f = JsFunction(f);
    Arc::new(move |event: &agentsmesh_events::RealtimeEvent| {
        if let Ok(json) = serde_json::to_string(event) {
            if let Ok(val) = js_sys::JSON::parse(&json) {
                f.call1(&val);
            }
        }
    })
}

pub(crate) fn make_state_listener(
    f: js_sys::Function,
) -> agentsmesh_events::StateListener {
    let f = JsFunction(f);
    Arc::new(move |state: agentsmesh_events::ConnectionState| {
        f.call1(&state.to_string().into());
    })
}
