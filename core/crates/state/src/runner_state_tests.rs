use crate::runner_state::RunnerState;
use agentsmesh_types::{Runner, RunnerStatus};

fn make_runner(id: i64, name: &str, status: RunnerStatus, enabled: bool, max: i32, active: i32) -> Runner {
    Runner {
        id,
        name: name.into(),
        status,
        version: None,
        max_concurrent_pods: max,
        active_pod_count: active,
        is_enabled: enabled,
        host_info: None,
        created_at: None,
        updated_at: None,
        ..Default::default()
    }
}

#[test]
fn new_state_is_empty() {
    let s = RunnerState::new();
    assert!(s.runners().is_empty());
    assert!(s.available_runners().is_empty());
    assert!(s.current_runner().is_none());
}

#[test]
fn set_runners() {
    let mut s = RunnerState::new();
    s.set_runners(vec![make_runner(1, "r1", RunnerStatus::Online, true, 4, 0)]);
    assert_eq!(s.runners().len(), 1);
}

#[test]
fn set_available_runners() {
    let mut s = RunnerState::new();
    s.set_available_runners(vec![make_runner(1, "r1", RunnerStatus::Online, true, 4, 0)]);
    assert_eq!(s.available_runners().len(), 1);
}

#[test]
fn get_runner_by_id() {
    let mut s = RunnerState::new();
    s.set_runners(vec![
        make_runner(1, "r1", RunnerStatus::Online, true, 4, 0),
        make_runner(2, "r2", RunnerStatus::Offline, false, 2, 0),
    ]);
    assert_eq!(s.get_runner(1).unwrap().name, "r1");
    assert_eq!(s.get_runner(2).unwrap().name, "r2");
    assert!(s.get_runner(99).is_none());
}

#[test]
fn set_current_runner() {
    let mut s = RunnerState::new();
    s.set_current_runner(Some(make_runner(1, "r1", RunnerStatus::Online, true, 4, 0)));
    assert_eq!(s.current_runner().unwrap().id, 1);
    s.set_current_runner(None);
    assert!(s.current_runner().is_none());
}

#[test]
fn update_runner_status() {
    let mut s = RunnerState::new();
    s.set_runners(vec![make_runner(1, "r1", RunnerStatus::Online, true, 4, 0)]);
    s.update_runner_status(1, RunnerStatus::Offline);
    assert_eq!(s.get_runner(1).unwrap().status, RunnerStatus::Offline);
}

#[test]
fn update_runner_status_nonexistent() {
    let mut s = RunnerState::new();
    s.update_runner_status(99, RunnerStatus::Online);
}

#[test]
fn update_runner_status_removes_from_available() {
    let mut s = RunnerState::new();
    let r = make_runner(1, "r1", RunnerStatus::Online, true, 4, 0);
    s.set_runners(vec![r.clone()]);
    s.set_available_runners(vec![r]);
    s.update_runner_status(1, RunnerStatus::Offline);
    assert!(s.available_runners().is_empty());
}

#[test]
fn can_accept_pods_true() {
    let r = make_runner(1, "r1", RunnerStatus::Online, true, 4, 2);
    assert!(RunnerState::can_accept_pods(&r));
}

#[test]
fn can_accept_pods_disabled() {
    let r = make_runner(1, "r1", RunnerStatus::Online, false, 4, 0);
    assert!(!RunnerState::can_accept_pods(&r));
}

#[test]
fn can_accept_pods_not_online() {
    let r = make_runner(1, "r1", RunnerStatus::Offline, true, 4, 0);
    assert!(!RunnerState::can_accept_pods(&r));
}

#[test]
fn can_accept_pods_at_capacity() {
    let r = make_runner(1, "r1", RunnerStatus::Online, true, 4, 4);
    assert!(!RunnerState::can_accept_pods(&r));
}

#[test]
fn can_accept_pods_over_capacity() {
    let r = make_runner(1, "r1", RunnerStatus::Online, true, 2, 5);
    assert!(!RunnerState::can_accept_pods(&r));
}
