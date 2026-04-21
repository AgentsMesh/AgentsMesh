use crate::loop_state::LoopState;
use agentsmesh_types::{LoopData, LoopRunData, LoopRunStatus};

fn make_loop(slug: &str, name: &str, enabled: bool) -> LoopData {
    LoopData {
        slug: slug.into(),
        name: name.into(),
        description: None,
        schedule: Some("0 * * * *".into()),
        is_enabled: enabled,
        last_run_at: None,
        created_at: None,
        updated_at: None,
        ..Default::default()
    }
}

fn make_run(id: i64, loop_slug: &str, status: LoopRunStatus) -> LoopRunData {
    LoopRunData {
        id,
        loop_slug: loop_slug.into(),
        status,
        started_at: Some("2026-01-01T00:00:00Z".into()),
        completed_at: None,
        error_message: None,
        ..Default::default()
    }
}

#[test]
fn new_state_is_empty() {
    let state = LoopState::new();
    assert!(state.get_loops().is_empty());
    assert!(state.get_current_loop().is_none());
    assert!(state.get_runs().is_empty());
}

#[test]
fn set_and_get_loops() {
    let mut state = LoopState::new();
    state.set_loops(vec![make_loop("l-1", "Hourly", true), make_loop("l-2", "Daily", false)]);
    assert_eq!(state.get_loops().len(), 2);
    assert_eq!(state.get_loops()[0].name, "Hourly");
}

#[test]
fn get_loop_by_slug() {
    let mut state = LoopState::new();
    state.set_loops(vec![make_loop("l-1", "Hourly", true), make_loop("l-2", "Daily", false)]);
    let found = state.get_loop_by_slug("l-2");
    assert!(found.is_some());
    assert_eq!(found.unwrap().name, "Daily");
    assert!(state.get_loop_by_slug("l-999").is_none());
}

#[test]
fn set_and_get_current_loop() {
    let mut state = LoopState::new();
    assert!(state.get_current_loop().is_none());
    state.set_current_loop(Some(make_loop("l-1", "Active", true)));
    assert_eq!(state.get_current_loop().unwrap().slug, "l-1");
    state.set_current_loop(None);
    assert!(state.get_current_loop().is_none());
}

#[test]
fn add_run() {
    let mut state = LoopState::new();
    state.add_run(make_run(1, "l-1", LoopRunStatus::Running));
    state.add_run(make_run(2, "l-1", LoopRunStatus::Completed));
    assert_eq!(state.get_runs().len(), 2);
    assert_eq!(state.get_runs()[0].status, LoopRunStatus::Running);
    assert_eq!(state.get_runs()[1].status, LoopRunStatus::Completed);
}

#[test]
fn update_run_status() {
    let mut state = LoopState::new();
    state.add_run(make_run(1, "l-1", LoopRunStatus::Running));
    state.update_run_status(1, LoopRunStatus::Completed);
    assert_eq!(state.get_runs()[0].status, LoopRunStatus::Completed);
}

#[test]
fn update_run_status_nonexistent_is_noop() {
    let mut state = LoopState::new();
    state.add_run(make_run(1, "l-1", LoopRunStatus::Running));
    state.update_run_status(999, LoopRunStatus::Failed);
    assert_eq!(state.get_runs()[0].status, LoopRunStatus::Running);
}

#[test]
fn clear_runs() {
    let mut state = LoopState::new();
    state.add_run(make_run(1, "l-1", LoopRunStatus::Completed));
    state.add_run(make_run(2, "l-1", LoopRunStatus::Failed));
    assert_eq!(state.get_runs().len(), 2);
    state.clear_runs();
    assert!(state.get_runs().is_empty());
}

#[test]
fn set_loops_replaces_all() {
    let mut state = LoopState::new();
    state.set_loops(vec![make_loop("l-1", "First", true)]);
    assert_eq!(state.get_loops().len(), 1);
    state.set_loops(vec![make_loop("l-2", "A", true), make_loop("l-3", "B", false)]);
    assert_eq!(state.get_loops().len(), 2);
    assert!(state.get_loop_by_slug("l-1").is_none());
}

#[test]
fn default_impl() {
    let state = LoopState::default();
    assert!(state.get_loops().is_empty());
    assert!(state.get_runs().is_empty());
}

#[test]
fn multiple_runs_different_loops() {
    let mut state = LoopState::new();
    state.add_run(make_run(1, "l-1", LoopRunStatus::Running));
    state.add_run(make_run(2, "l-2", LoopRunStatus::Completed));
    assert_eq!(state.get_runs().len(), 2);
    assert_eq!(state.get_runs()[0].loop_slug, "l-1");
    assert_eq!(state.get_runs()[1].loop_slug, "l-2");
}
