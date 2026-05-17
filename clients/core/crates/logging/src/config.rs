use std::path::PathBuf;

/// Logging configuration handed to [`crate::init`]. Hosts pick the construct
/// that matches their platform — `console` for transient output, `file` for
/// rolling-file persistence, `wasm_console` for browsers.
#[derive(Debug, Clone)]
pub struct LogConfig {
    pub level: String,
    pub file: Option<FileSink>,
    pub json: bool,
}

/// Rolling-file sink parameters. The crate creates `<dir>/<prefix>.YYYY-MM-DD`
/// and retains the newest `max_files` rotations; older files are removed.
#[derive(Debug, Clone)]
pub struct FileSink {
    pub dir: PathBuf,
    pub prefix: String,
    pub max_files: usize,
}

impl LogConfig {
    /// Stderr only — no persistence. Useful for tests and for early-boot logs
    /// before a writable directory is known.
    pub fn console(level: impl Into<String>) -> Self {
        Self {
            level: level.into(),
            file: None,
            json: false,
        }
    }

    /// Rolling daily file + stderr. The directory will be created if missing.
    /// JSON output is enabled so logs can be parsed by tooling downstream.
    pub fn file(dir: impl Into<PathBuf>, level: impl Into<String>) -> Self {
        Self {
            level: level.into(),
            file: Some(FileSink {
                dir: dir.into(),
                prefix: "agentsmesh".into(),
                max_files: 7,
            }),
            json: true,
        }
    }

    /// WASM browser console output. The `file` field is ignored on this target
    /// since the wasm sandbox has no filesystem.
    pub fn wasm_console(level: impl Into<String>) -> Self {
        Self::console(level)
    }
}
