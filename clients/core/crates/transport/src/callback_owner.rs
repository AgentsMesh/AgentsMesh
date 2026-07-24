pub(crate) struct CallbackOwner {
    cleanup: Option<Box<dyn FnOnce()>>,
}

impl CallbackOwner {
    pub(crate) fn new(cleanup: impl FnOnce() + 'static) -> Self {
        Self {
            cleanup: Some(Box::new(cleanup)),
        }
    }

    pub(crate) fn detach(&mut self) {
        if let Some(cleanup) = self.cleanup.take() {
            cleanup();
        }
    }
}

impl Drop for CallbackOwner {
    fn drop(&mut self) {
        self.detach();
    }
}

#[cfg(test)]
mod tests {
    use std::cell::Cell;
    use std::rc::Rc;

    use super::*;

    #[test]
    fn drop_runs_cleanup_once() {
        let calls = Rc::new(Cell::new(0));
        {
            let calls = Rc::clone(&calls);
            let _owner = CallbackOwner::new(move || calls.set(calls.get() + 1));
        }
        assert_eq!(calls.get(), 1);
    }

    #[test]
    fn explicit_detach_is_idempotent_and_disarms_drop() {
        let calls = Rc::new(Cell::new(0));
        {
            let calls_for_cleanup = Rc::clone(&calls);
            let mut owner =
                CallbackOwner::new(move || calls_for_cleanup.set(calls_for_cleanup.get() + 1));
            owner.detach();
            owner.detach();
            assert_eq!(calls.get(), 1);
        }
        assert_eq!(calls.get(), 1);
    }
}
