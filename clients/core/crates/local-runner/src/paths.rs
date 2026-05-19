use std::path::{Path, PathBuf};

#[derive(Debug, Clone)]
pub struct InstallPaths {
    config_dir: PathBuf,
    install_dir: PathBuf,
}

impl InstallPaths {
    pub fn new(home: impl AsRef<Path>) -> Self {
        let base = home.as_ref().join(".agentsmesh");
        Self {
            config_dir: base.clone(),
            install_dir: base.join("bin"),
        }
    }

    pub fn config_dir(&self) -> &Path {
        &self.config_dir
    }

    pub fn install_dir(&self) -> &Path {
        &self.install_dir
    }

    pub fn binary_path(&self) -> PathBuf {
        self.install_dir.join(binary_name())
    }

    pub fn config_file(&self) -> PathBuf {
        self.config_dir.join("config.yaml")
    }
}

#[cfg(windows)]
fn binary_name() -> &'static str {
    "agentsmesh-runner.exe"
}

#[cfg(not(windows))]
fn binary_name() -> &'static str {
    "agentsmesh-runner"
}
