#![cfg(not(target_arch = "wasm32"))]

use std::fs;
use tracing_appender::non_blocking::WorkerGuard;
use tracing_appender::rolling::{RollingFileAppender, Rotation};
use tracing_subscriber::{fmt, layer::SubscriberExt, util::SubscriberInitExt, EnvFilter, Registry};

use crate::config::{FileSink, LogConfig};
use crate::init::LogError;

// `try_init` instead of `init`: a second install becomes a no-op (test harness,
// second host bootstrap), matching `crate::init`'s OnceLock guard.
pub fn install(config: &LogConfig, filter: EnvFilter) -> Result<Option<WorkerGuard>, LogError> {
    let stderr_layer = fmt::layer().with_writer(std::io::stderr).with_ansi(false);

    let Some(file_cfg) = &config.file else {
        let _ = Registry::default().with(filter).with(stderr_layer).try_init();
        return Ok(None);
    };

    fs::create_dir_all(&file_cfg.dir)?;
    let appender = appender_for(file_cfg);
    let (writer, guard) = tracing_appender::non_blocking(appender);

    if config.json {
        let file_layer = fmt::layer().json().with_writer(writer);
        let _ = Registry::default()
            .with(filter)
            .with(stderr_layer)
            .with(file_layer)
            .try_init();
    } else {
        let file_layer = fmt::layer().with_writer(writer).with_ansi(false);
        let _ = Registry::default()
            .with(filter)
            .with(stderr_layer)
            .with(file_layer)
            .try_init();
    }
    Ok(Some(guard))
}

fn appender_for(file: &FileSink) -> RollingFileAppender {
    RollingFileAppender::builder()
        .rotation(Rotation::DAILY)
        .filename_prefix(&file.prefix)
        .max_log_files(file.max_files)
        .build(&file.dir)
        .expect("rolling appender build")
}
