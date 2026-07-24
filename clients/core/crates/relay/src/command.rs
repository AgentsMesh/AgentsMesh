use futures::channel::oneshot;
use serde_json::Value;

use crate::types::OutputCallback;

/// Messages routed from the pool (any caller) into a pod's driver task. The
/// driver owns all per-connection state; the pool only forwards these and never
/// touches connection internals — that is what makes the link a single-owner,
/// lock-free actor. Status/ACP listeners are NOT commands: they may be
/// registered before the driver exists, so they live pool-side in `PoolRouter`
/// and the driver fans out by reading the router.
pub(crate) enum Command {
    AddSubscriber {
        sub_id: String,
        cb: OutputCallback,
        ready: Option<oneshot::Sender<()>>,
    },
    RemoveSubscriber {
        sub_id: String,
    },
    Send {
        data: String,
    },
    Resize {
        cols: u16,
        rows: u16,
        force: bool,
    },
    SendAcp {
        command: Value,
    },
    Disconnect,
}
