use parking_lot::Mutex;

use crate::types::{AcpCallback, RelayStatusInfo, StatusCallback};

pub(crate) struct StatusListenerEntry {
    generation: Option<u32>,
    lease_id: Option<String>,
    callback: StatusCallback,
    delivered_revision: Mutex<Option<u64>>,
}

impl StatusListenerEntry {
    pub(super) fn new(
        generation: Option<u32>,
        lease_id: Option<String>,
        callback: StatusCallback,
    ) -> Self {
        Self {
            generation,
            lease_id,
            callback,
            delivered_revision: Mutex::new(None),
        }
    }

    pub(super) fn generation(&self) -> Option<u32> {
        self.generation
    }

    pub(super) fn lease_id(&self) -> Option<&str> {
        self.lease_id.as_deref()
    }

    pub(crate) fn deliver(&self, info: RelayStatusInfo) {
        let mut delivered = self.delivered_revision.lock();
        if delivered.is_some_and(|revision| revision >= info.revision) {
            return;
        }
        *delivered = Some(info.revision);
        (self.callback)(info);
    }
}

pub(crate) struct AcpListenerEntry {
    generation: Option<u32>,
    lease_id: Option<String>,
    callback: AcpCallback,
}

impl AcpListenerEntry {
    pub(super) fn new(
        generation: Option<u32>,
        lease_id: Option<String>,
        callback: AcpCallback,
    ) -> Self {
        Self {
            generation,
            lease_id,
            callback,
        }
    }

    pub(super) fn generation(&self) -> Option<u32> {
        self.generation
    }

    pub(super) fn lease_id(&self) -> Option<&str> {
        self.lease_id.as_deref()
    }

    pub(crate) fn deliver(
        &self,
        msg_type: agentsmesh_protocol::MsgType,
        payload: serde_json::Value,
    ) {
        (self.callback)(msg_type, payload);
    }
}

// LCOV_EXCL_START: test-only code
#[cfg(test)]
mod tests {
    use std::sync::{Arc, Mutex};

    use crate::types::RelayStatus;

    use super::*;

    #[test]
    fn status_listener_drops_duplicate_and_stale_revisions() {
        let revisions = Arc::new(Mutex::new(Vec::new()));
        let captured = Arc::clone(&revisions);
        let listener = StatusListenerEntry::new(
            None,
            None,
            Arc::new(move |info| captured.lock().unwrap().push(info.revision)),
        );
        let info = |revision| RelayStatusInfo {
            status: RelayStatus::Connected,
            runner_disconnected: false,
            revision,
        };

        listener.deliver(info(2));
        listener.deliver(info(2));
        listener.deliver(info(1));
        listener.deliver(info(3));

        assert_eq!(*revisions.lock().unwrap(), vec![2, 3]);
    }
}
// LCOV_EXCL_STOP
