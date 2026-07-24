use std::cell::RefCell;
use std::rc::Rc;

use futures_channel::mpsc;
use js_sys::{ArrayBuffer, Uint8Array};
use tracing::{debug, warn};
use wasm_bindgen::prelude::*;
use web_sys::{CloseEvent, Event, MessageEvent, WebSocket};

use crate::callback_owner::CallbackOwner;
use crate::error::TransportError;
use crate::message::WsMessage;

pub(super) struct WsCallbacks {
    _on_message: Closure<dyn FnMut(MessageEvent)>,
    _on_error: Closure<dyn FnMut(Event)>,
    _on_close: Closure<dyn FnMut(CloseEvent)>,
    owner: CallbackOwner,
}

impl Drop for WsCallbacks {
    fn drop(&mut self) {
        self.owner.detach();
    }
}

struct OpenCallbacks {
    on_open: Closure<dyn FnMut()>,
    on_error: Closure<dyn FnMut(Event)>,
    on_close: Closure<dyn FnMut(CloseEvent)>,
    owner: CallbackOwner,
}

impl Drop for OpenCallbacks {
    fn drop(&mut self) {
        self.owner.detach();
    }
}

pub(super) fn attach_callbacks(
    ws: &WebSocket,
    read_tx: mpsc::UnboundedSender<WsMessage>,
) -> Result<WsCallbacks, TransportError> {
    let tx = read_tx.clone();
    let on_message = Closure::wrap(Box::new(move |e: MessageEvent| {
        let data = e.data();
        if let Ok(buf) = data.clone().dyn_into::<ArrayBuffer>() {
            let array = Uint8Array::new(&buf);
            let _ = tx.unbounded_send(WsMessage::Binary(array.to_vec()));
        } else if let Some(text) = data.as_string() {
            let _ = tx.unbounded_send(WsMessage::Text(text));
        }
    }) as Box<dyn FnMut(MessageEvent)>);

    let on_error = Closure::wrap(Box::new(move |_e: Event| {
        warn!("websocket error");
    }) as Box<dyn FnMut(Event)>);

    let tx = read_tx;
    let on_close = Closure::wrap(Box::new(move |e: CloseEvent| {
        debug!("websocket closed");
        let _ = tx.unbounded_send(WsMessage::Close(Some(e.code())));
    }) as Box<dyn FnMut(CloseEvent)>);

    let detach_ws = ws.clone();
    let owner = CallbackOwner::new(move || {
        detach_ws.set_onmessage(None);
        detach_ws.set_onerror(None);
        detach_ws.set_onclose(None);
    });

    ws.set_onmessage(Some(on_message.as_ref().unchecked_ref()));
    ws.set_onerror(Some(on_error.as_ref().unchecked_ref()));
    ws.set_onclose(Some(on_close.as_ref().unchecked_ref()));

    Ok(WsCallbacks {
        _on_message: on_message,
        _on_error: on_error,
        _on_close: on_close,
        owner,
    })
}

pub(super) async fn wait_for_open(ws: &WebSocket) -> Result<(), TransportError> {
    let (tx, rx) = futures_channel::oneshot::channel::<Result<(), String>>();
    let tx = Rc::new(RefCell::new(Some(tx)));

    let tx_open = Rc::clone(&tx);
    let on_open = Closure::wrap(Box::new(move || {
        if let Some(tx) = tx_open.borrow_mut().take() {
            let _ = tx.send(Ok(()));
        }
    }) as Box<dyn FnMut()>);

    let tx_err = Rc::clone(&tx);
    let on_error = Closure::wrap(Box::new(move |_e: Event| {
        if let Some(tx) = tx_err.borrow_mut().take() {
            let _ = tx.send(Err("websocket error before open".to_string()));
        }
    }) as Box<dyn FnMut(Event)>);

    let tx_close = tx;
    let on_close = Closure::wrap(Box::new(move |e: CloseEvent| {
        if let Some(tx) = tx_close.borrow_mut().take() {
            let _ = tx.send(Err(format!("websocket closed before open: {}", e.code())));
        }
    }) as Box<dyn FnMut(CloseEvent)>);

    let detach_ws = ws.clone();
    let owner = CallbackOwner::new(move || {
        detach_ws.set_onopen(None);
        detach_ws.set_onerror(None);
        detach_ws.set_onclose(None);
    });
    let callbacks = OpenCallbacks {
        on_open,
        on_error,
        on_close,
        owner,
    };

    ws.set_onopen(Some(callbacks.on_open.as_ref().unchecked_ref()));
    ws.set_onerror(Some(callbacks.on_error.as_ref().unchecked_ref()));
    ws.set_onclose(Some(callbacks.on_close.as_ref().unchecked_ref()));

    rx.await
        .map_err(|_| TransportError::ConnectionFailed("open cancelled".into()))?
        .map_err(TransportError::ConnectionFailed)
}
