use std::time::Duration;

use rand::Rng;

pub const BASE_RECONNECT_DELAY_MS: u64 = 1000;
pub const MAX_RECONNECT_DELAY_MS: u64 = 30_000;
pub const SNAPSHOT_TIMEOUT_MS: u64 = 2000;
// Consecutive Resync attempts (SNAPSHOT_TIMEOUT_MS apart) tolerated while the
// transport is up but no snapshot has arrived. Past this the connection is
// data-dead — force a reconnect to rebuild it rather than sit Connected-but-
// blank forever. Re-requesting persists across reconnects, so the terminal
// self-heals once the relay/runner path recovers (no manual restart needed).
pub const SNAPSHOT_GIVEUP_ATTEMPTS: u32 = 15;
// Cap a single connect attempt: a hung TCP/WS handshake has no OS-level timeout
// here, so without this the driver could block in connect().await forever and
// never see a queued Disconnect. On timeout we fall through to backoff, which
// does drain commands.
pub const CONNECT_TIMEOUT_MS: u64 = 15_000;
pub const DISCONNECT_DELAY_MS: u64 = 30_000;
pub const RESIZE_DEBOUNCE_MS: u64 = 150;
pub const INPUT_DEDUP_WINDOW_MS: u64 = 50;

pub fn compute_reconnect_delay(attempt: u32, base_delay_ms: u64) -> Duration {
    let exp = base_delay_ms.saturating_mul(1u64 << attempt.min(20));
    let base = exp.min(MAX_RECONNECT_DELAY_MS);
    if base == 0 {
        return Duration::from_millis(0);
    }
    let jitter_range = (base as f64) * 0.4;
    let jitter = rand::rng().random_range(-jitter_range / 2.0..jitter_range / 2.0);
    let delay = (base as f64 + jitter).round().max(0.0) as u64;
    Duration::from_millis(delay)
}

// LCOV_EXCL_START: test-only code
#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn first_delay_near_base() {
        for _ in 0..20 {
            let d = compute_reconnect_delay(0, 1000);
            let ms = d.as_millis() as u64;
            assert!(ms >= 800 && ms <= 1200, "first delay was {ms}ms");
        }
    }

    #[test]
    fn exponential_growth() {
        let d0 = 1000u64;
        let d1 = 2000u64;
        let d2 = 4000u64;

        for _ in 0..20 {
            let a0 = compute_reconnect_delay(0, 1000).as_millis() as u64;
            assert!(a0 >= d0 * 80 / 100 && a0 <= d0 * 120 / 100);

            let a1 = compute_reconnect_delay(1, 1000).as_millis() as u64;
            assert!(a1 >= d1 * 80 / 100 && a1 <= d1 * 120 / 100);

            let a2 = compute_reconnect_delay(2, 1000).as_millis() as u64;
            assert!(a2 >= d2 * 80 / 100 && a2 <= d2 * 120 / 100);
        }
    }

    #[test]
    fn capped_at_30s() {
        for _ in 0..20 {
            let d = compute_reconnect_delay(20, 1000);
            let ms = d.as_millis() as u64;
            assert!(ms <= 36_000, "delay {ms}ms exceeds 30s + jitter");
            assert!(ms >= 24_000, "delay {ms}ms too low for cap");
        }
    }

    #[test]
    fn jitter_range_within_20_percent() {
        let mut min_seen = u64::MAX;
        let mut max_seen = 0u64;
        for _ in 0..200 {
            let ms = compute_reconnect_delay(0, 1000).as_millis() as u64;
            min_seen = min_seen.min(ms);
            max_seen = max_seen.max(ms);
        }
        assert!(
            min_seen < 1000,
            "jitter should go below base: min={min_seen}"
        );
        assert!(
            max_seen > 1000,
            "jitter should go above base: max={max_seen}"
        );
    }

    #[test]
    fn zero_base_delay() {
        let d = compute_reconnect_delay(0, 0);
        assert_eq!(d.as_millis(), 0);
    }

    #[test]
    fn very_high_attempt_does_not_overflow() {
        let d = compute_reconnect_delay(100, 1000);
        let ms = d.as_millis() as u64;
        assert!(ms <= 36_000);
    }

    #[test]
    fn constants_are_correct() {
        assert_eq!(SNAPSHOT_TIMEOUT_MS, 2000);
        assert_eq!(SNAPSHOT_GIVEUP_ATTEMPTS, 15);
        assert_eq!(DISCONNECT_DELAY_MS, 30_000);
        assert_eq!(RESIZE_DEBOUNCE_MS, 150);
        assert_eq!(INPUT_DEDUP_WINDOW_MS, 50);
    }
}
// LCOV_EXCL_STOP
