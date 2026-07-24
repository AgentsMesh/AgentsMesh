use std::cell::RefCell;
use std::rc::Rc;

use futures_channel::mpsc;
use futures_channel::mpsc::UnboundedReceiver;
use futures_util::StreamExt;
use js_sys::{ArrayBuffer, Uint8Array};
use tracing::{debug, warn};
use wasm_bindgen::prelude::*;
use web_sys::{BinaryType, CloseEvent, Event, MessageEvent, WebSocket};

use crate::callback_owner::CallbackOwner;
use crate::error::TransportError;
use crate::message::WsMessage;

pub struct WebSocketConnection {
    ws: WebSocket,
    read_rx: UnboundedReceiver<WsMessage>,
    #[allow(dead_code)]
    callbacks: WsCallbacks,
}

struct WsCallbacks {
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
    _on_open: Closure<dyn FnMut()>,
    _on_error: Closure<dyn FnMut(Event)>,
    _on_close: Closure<dyn FnMut(CloseEvent)>,
    owner: CallbackOwner,
}

impl Drop for OpenCallbacks {
    fn drop(&mut self) {
        self.owner.detach();
    }
}

impl WebSocketConnection {
    pub async fn connect(url: &str) -> Result<Self, TransportError> {
        let ws =
            WebSocket::new(url).map_err(|e| TransportError::ConnectionFailed(format!("{e:?}")))?;
        ws.set_binary_type(BinaryType::Arraybuffer);

        wait_for_open(&ws).await?;
        let (read_tx, read_rx) = mpsc::unbounded::<WsMessage>();
        let callbacks = attach_callbacks(&ws, read_tx)?;

        debug!("websocket connected to {url}");
        Ok(Self {
            ws,
            read_rx,
            callbacks,
        })
    }

    pub fn send_binary(&self, data: Vec<u8>) -> Result<(), TransportError> {
        self.ws
            .send_with_u8_array(&data)
            .map_err(|e| TransportError::SendFailed(format!("{e:?}")))
    }

    pub fn send_text(&self, text: String) -> Result<(), TransportError> {
        self.ws
            .send_with_str(&text)
            .map_err(|e| TransportError::SendFailed(format!("{e:?}")))
    }

    pub async fn recv(&mut self) -> Result<WsMessage, TransportError> {
        self.read_rx.next().await.ok_or(TransportError::Closed)
    }

    pub fn close(&self) {
        let _ = self.ws.close();
    }

    pub fn is_closed(&self) -> bool {
        self.ws.ready_state() == WebSocket::CLOSED || self.ws.ready_state() == WebSocket::CLOSING
    }

    pub fn into_split(self) -> (WsSender, WsReceiver) {
        (
            WsSender {
                ws: self.ws.clone(),
                _callbacks: self.callbacks,
            },
            WsReceiver {
                read_rx: self.read_rx,
            },
        )
    }
}

pub struct WsSender {
    ws: WebSocket,
    #[allow(dead_code)]
    _callbacks: WsCallbacks,
}

impl WsSender {
    pub fn send_binary(&self, data: Vec<u8>) -> Result<(), TransportError> {
        self.ws
            .send_with_u8_array(&data)
            .map_err(|e| TransportError::SendFailed(format!("{e:?}")))
    }

    pub fn send_text(&self, text: String) -> Result<(), TransportError> {
        self.ws
            .send_with_str(&text)
            .map_err(|e| TransportError::SendFailed(format!("{e:?}")))
    }

    pub fn close(&self) {
        let _ = self.ws.close();
    }
}

pub struct WsReceiver {
    read_rx: UnboundedReceiver<WsMessage>,
}

impl WsReceiver {
    pub async fn recv(&mut self) -> Result<WsMessage, TransportError> {
        self.read_rx.next().await.ok_or(TransportError::Closed)
    }
}

fn attach_callbacks(
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

async fn wait_for_open(ws: &WebSocket) -> Result<(), TransportError> {
    let (tx, rx) = futures_channel::oneshot::channel::<Result<(), String>>();
    let tx = Rc::new(RefCell::new(Some(tx)));

    let tx_open = tx.clone();
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
    let _callbacks = OpenCallbacks {
        _on_open: on_open,
        _on_error: on_error,
        _on_close: on_close,
        owner,
    };

    ws.set_onopen(Some(_callbacks._on_open.as_ref().unchecked_ref()));
    ws.set_onerror(Some(_callbacks._on_error.as_ref().unchecked_ref()));
    ws.set_onclose(Some(_callbacks._on_close.as_ref().unchecked_ref()));

    rx.await
        .map_err(|_| TransportError::ConnectionFailed("open cancelled".into()))?
        .map_err(TransportError::ConnectionFailed)
}
