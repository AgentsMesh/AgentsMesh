use crate::cli::run_runner_cli;
use crate::error::{LocalRunnerError, Result};
use crate::paths::InstallPaths;
use serde::{Deserialize, Serialize};

/// `Stale` means the OS service job is installed but the runner config
/// (`~/.agentsmesh/config.yaml`) is missing — UI must treat this as
/// "not registered" so a stale launchd/systemd job from an old
/// registration doesn't falsely report `Running` or `Stopped`.
#[derive(Debug, Clone, Copy, Eq, PartialEq, Serialize, Deserialize)]
pub enum ServiceStatus {
    Running,
    Stopped,
    Unknown,
    NotInstalled,
    Stale,
}

pub async fn install(paths: &InstallPaths) -> Result<()> {
    run_runner_cli(paths, &["service", "install"]).await?.ok().map(|_| ())
}

pub async fn uninstall(paths: &InstallPaths) -> Result<()> {
    run_runner_cli(paths, &["service", "uninstall"]).await?.ok().map(|_| ())
}

pub async fn start(paths: &InstallPaths) -> Result<()> {
    run_runner_cli(paths, &["service", "start"]).await?.ok().map(|_| ())
}

pub async fn stop(paths: &InstallPaths) -> Result<()> {
    run_runner_cli(paths, &["service", "stop"]).await?.ok().map(|_| ())
}

pub async fn status(paths: &InstallPaths) -> Result<ServiceStatus> {
    if !crate::install::is_installed(paths).await {
        return Ok(ServiceStatus::NotInstalled);
    }
    let output = run_runner_cli(paths, &["service", "status"]).await?;
    if output.status != 0 {
        return Ok(ServiceStatus::NotInstalled);
    }
    parse_status(&output.stdout)
}

fn parse_status(stdout: &str) -> Result<ServiceStatus> {
    for line in stdout.lines() {
        let line = line.trim();
        if let Some(rest) = line.strip_prefix("Service Status:") {
            return Ok(match rest.trim() {
                "Running" => ServiceStatus::Running,
                "Stopped" => ServiceStatus::Stopped,
                "Stale" => ServiceStatus::Stale,
                "Unknown" => ServiceStatus::Unknown,
                _ => ServiceStatus::Unknown,
            });
        }
    }
    Err(LocalRunnerError::UnexpectedOutput(stdout.to_string()))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parses_running() {
        assert_eq!(parse_status("Service Status: Running\n").unwrap(), ServiceStatus::Running);
    }

    #[test]
    fn parses_stopped() {
        assert_eq!(parse_status("Service Status: Stopped\n").unwrap(), ServiceStatus::Stopped);
    }

    #[test]
    fn parses_stale() {
        assert_eq!(parse_status("Service Status: Stale\n").unwrap(), ServiceStatus::Stale);
    }

    #[test]
    fn parses_unknown_token() {
        assert_eq!(parse_status("Service Status: Whatever\n").unwrap(), ServiceStatus::Unknown);
    }

    #[test]
    fn rejects_missing_status_line() {
        assert!(matches!(
            parse_status("hello world").unwrap_err(),
            LocalRunnerError::UnexpectedOutput(_)
        ));
    }

    // Regression: when the runner binary doesn't exist on disk, status() must
    // return ServiceStatus::NotInstalled, not propagate a BinaryNotFound error.
    // The four-state enum is the IPC contract — surfacing the error as a
    // promise rejection on the renderer side caused unhandled rejections in
    // DevTools and pinned the onboarding card in "Checking…".
    #[test]
    fn status_returns_not_installed_when_binary_missing() {
        let rt = tokio::runtime::Runtime::new().unwrap();
        let paths = InstallPaths::new("/nonexistent-home-for-tests");
        let result = rt.block_on(status(&paths)).expect("must not error");
        assert_eq!(result, ServiceStatus::NotInstalled);
    }
}
