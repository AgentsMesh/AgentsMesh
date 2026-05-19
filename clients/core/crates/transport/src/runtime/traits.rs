use std::future::Future;
use std::pin::Pin;
use std::time::Duration;

#[cfg(not(target_arch = "wasm32"))]
pub type BoxFuture<T> = Pin<Box<dyn Future<Output = T> + Send>>;
#[cfg(target_arch = "wasm32")]
pub type BoxFuture<T> = Pin<Box<dyn Future<Output = T>>>;

#[cfg(not(target_arch = "wasm32"))]
pub trait TaskHandle: Send + Sync + 'static {
    fn abort(&self);
}

#[cfg(target_arch = "wasm32")]
pub trait TaskHandle: 'static {
    fn abort(&self);
}

#[cfg(not(target_arch = "wasm32"))]
pub trait Runtime: Clone + Send + Sync + 'static {
    type TaskHandle: TaskHandle;

    fn spawn(&self, fut: Pin<Box<dyn Future<Output = ()> + Send + 'static>>) -> Self::TaskHandle;
    fn sleep(&self, duration: Duration) -> BoxFuture<()>;
}

#[cfg(target_arch = "wasm32")]
pub trait Runtime: Clone + 'static {
    type TaskHandle: TaskHandle;

    fn spawn(&self, fut: Pin<Box<dyn Future<Output = ()> + 'static>>) -> Self::TaskHandle;
    fn sleep(&self, duration: Duration) -> BoxFuture<()>;
}

/// Free function rather than a `Runtime` trait method: the bound on the
/// future's `Output` would force every Runtime impl to spell out the
/// generic; `R: Runtime` keeps the trait dyn-safe.
pub async fn timeout<R, F>(runtime: &R, duration: Duration, fut: F) -> Result<F::Output, Elapsed>
where
    R: Runtime,
    F: Future,
{
    use futures::future::{select, Either};
    let sleep = runtime.sleep(duration);
    futures::pin_mut!(fut);
    futures::pin_mut!(sleep);
    match select(fut, sleep).await {
        Either::Left((output, _)) => Ok(output),
        Either::Right(_) => Err(Elapsed),
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Elapsed;

impl std::fmt::Display for Elapsed {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.write_str("operation timed out")
    }
}

impl std::error::Error for Elapsed {}
