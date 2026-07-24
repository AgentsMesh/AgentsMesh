use futures_channel::mpsc;
use futures_channel::mpsc::UnboundedReceiver;
use futures_util::StreamExt;
use tracing::debug;
use web_sys::{BinaryType, WebSocket};

use crate::error::TransportError;
use crate::message::WsMessage;

mod callbacks;

use callbacks::{attach_callbacks, wait_for_open, WsCallbacks};

pub struct WebSocketConnection {
    ws: WebSocket,
    read_rx: UnboundedReceiver<WsMessage>,
    #[allow(dead_code)]
    callbacks: WsCallbacks,
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
