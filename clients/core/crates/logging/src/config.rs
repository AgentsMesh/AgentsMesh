use std::path::PathBuf;

#[derive(Debug, Clone)]
pub struct LogConfig {
    pub level: String,
    pub file: Option<FileSink>,
    pub json: bool,
}

#[derive(Debug, Clone)]
pub struct FileSink {
    pub dir: PathBuf,
    pub prefix: String,
    pub max_files: usize,
}

impl LogConfig {
    pub fn console(level: impl Into<String>) -> Self {
        Self {
            level: level.into(),
            file: None,
            json: false,
        }
    }

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

    pub fn wasm_console(level: impl Into<String>) -> Self {
        Self::console(level)
    }
}
