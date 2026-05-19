use js_sys::ArrayBuffer;
use wasm_bindgen::prelude::*;
use web_sys::{BinaryType, CloseEvent, ErrorEvent, MessageEvent, WebSocket};

use crate::js_bridge::JsFunction;

#[wasm_bindgen]
pub struct WasmWebSocket {
    ws: WebSocket,
    _on_open: Closure<dyn FnMut()>,
    _on_message: Closure<dyn FnMut(MessageEvent)>,
    _on_close: Closure<dyn FnMut(CloseEvent)>,
    _on_error: Closure<dyn FnMut(ErrorEvent)>,
}

#[wasm_bindgen]
impl WasmWebSocket {
    pub fn connect(
        url: &str,
        on_open: js_sys::Function,
        on_message: js_sys::Function,
        on_close: js_sys::Function,
        on_error: js_sys::Function,
    ) -> Result<WasmWebSocket, String> {
        let ws = WebSocket::new(url).map_err(|e| format!("{e:?}"))?;
        ws.set_binary_type(BinaryType::Arraybuffer);

        let open_fn = JsFunction(on_open);
        let on_open_cb = Closure::wrap(Box::new(move || {
            open_fn.call1(&JsValue::NULL);
        }) as Box<dyn FnMut()>);

        let msg_fn = JsFunction(on_message);
        let on_msg_cb = Closure::wrap(Box::new(move |e: MessageEvent| {
            let data = e.data();
            if let Ok(buf) = data.clone().dyn_into::<ArrayBuffer>() {
                msg_fn.call1(&buf.into());
            } else if let Some(text) = data.as_string() {
                msg_fn.call1(&JsValue::from_str(&text));
            }
        }) as Box<dyn FnMut(MessageEvent)>);

        let close_fn = JsFunction(on_close);
        let on_close_cb = Closure::wrap(Box::new(move |e: CloseEvent| {
            let obj = js_sys::Object::new();
            let _ = js_sys::Reflect::set(&obj, &"code".into(), &e.code().into());
            let _ = js_sys::Reflect::set(&obj, &"reason".into(), &e.reason().into());
            close_fn.call1(&obj.into());
        }) as Box<dyn FnMut(CloseEvent)>);

        let err_fn = JsFunction(on_error);
        let on_err_cb = Closure::wrap(Box::new(move |_: ErrorEvent| {
            err_fn.call1(&JsValue::NULL);
        }) as Box<dyn FnMut(ErrorEvent)>);

        ws.set_onopen(Some(on_open_cb.as_ref().unchecked_ref()));
        ws.set_onmessage(Some(on_msg_cb.as_ref().unchecked_ref()));
        ws.set_onclose(Some(on_close_cb.as_ref().unchecked_ref()));
        ws.set_onerror(Some(on_err_cb.as_ref().unchecked_ref()));

        Ok(Self {
            ws,
            _on_open: on_open_cb,
            _on_message: on_msg_cb,
            _on_close: on_close_cb,
            _on_error: on_err_cb,
        })
    }

    pub fn send_binary(&self, data: &[u8]) -> Result<(), String> {
        self.ws
            .send_with_u8_array(data)
            .map_err(|e| format!("{e:?}"))
    }

    pub fn send_text(&self, text: &str) -> Result<(), String> {
        self.ws
            .send_with_str(text)
            .map_err(|e| format!("{e:?}"))
    }

    pub fn close(&self) {
        self.ws.set_onmessage(None);
        self.ws.set_onerror(None);
        self.ws.set_onclose(None);

        if self.ws.ready_state() == WebSocket::CONNECTING {
            // Workaround: calling ws.close() during the handshake makes the
            // browser log "WebSocket is closed before the connection is
            // established" — a spec-mandated warning whenever a CONNECTING
            // socket is aborted. Defer until onopen so readyState is OPEN
            // at close time. Fires under React Strict Mode's double-mount:
            // first mount's ws is still negotiating when
            // EventSubscriptionManager.disconnect() runs.
            let ws_clone = self.ws.clone();
            let close_after_open = Closure::wrap(Box::new(move || {
                ws_clone.set_onopen(None);
                let _ = ws_clone.close();
            }) as Box<dyn FnMut()>);
            self.ws
                .set_onopen(Some(close_after_open.as_ref().unchecked_ref()));
            close_after_open.forget();
            return;
        }

        self.ws.set_onopen(None);
        let _ = self.ws.close();
    }

    pub fn is_open(&self) -> bool {
        self.ws.ready_state() == WebSocket::OPEN
    }

    pub fn is_closed(&self) -> bool {
        let s = self.ws.ready_state();
        s == WebSocket::CLOSED || s == WebSocket::CLOSING
    }
}
