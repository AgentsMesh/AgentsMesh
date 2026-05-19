use std::time::Duration;

pub struct ReconnectPolicy {
    initial_delay_ms: u64,
    max_delay_ms: u64,
    max_attempts: u32,
    attempts: u32,
}

impl ReconnectPolicy {
    pub fn new(initial_delay_ms: u64, max_delay_ms: u64, max_attempts: u32) -> Self {
        Self {
            initial_delay_ms,
            max_delay_ms,
            max_attempts,
            attempts: 0,
        }
    }

    pub fn next_delay(&mut self) -> Option<Duration> {
        if self.attempts >= self.max_attempts {
            return None;
        }
        self.attempts += 1;
        let base = self.initial_delay_ms as f64 * 2_f64.powi(self.attempts as i32 - 1);
        let jitter = rand_jitter();
        let delay_ms = (base + jitter).min(self.max_delay_ms as f64) as u64;
        Some(Duration::from_millis(delay_ms))
    }

    pub fn reset(&mut self) {
        self.attempts = 0;
    }

    pub fn attempts(&self) -> u32 {
        self.attempts
    }

    pub fn max_attempts(&self) -> u32 {
        self.max_attempts
    }

    pub fn is_exhausted(&self) -> bool {
        self.attempts >= self.max_attempts
    }
}

fn rand_jitter() -> f64 {
    use std::collections::hash_map::DefaultHasher;
    use std::hash::{Hash, Hasher};
    use web_time::SystemTime;

    let mut hasher = DefaultHasher::new();
    SystemTime::now()
        .duration_since(SystemTime::UNIX_EPOCH)
        .unwrap_or_default()
        .as_nanos()
        .hash(&mut hasher);
    let hash = hasher.finish();
    (hash % 1000) as f64
}

pub fn should_reconnect(close_code: Option<u16>) -> bool {
    close_code != Some(1000)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_first_delay_is_around_initial() {
        let mut policy = ReconnectPolicy::new(1000, 30000, 10);
        let delay = policy.next_delay().unwrap();
        assert!(delay.as_millis() >= 1000);
        assert!(delay.as_millis() <= 2000);
    }

    #[test]
    fn test_exponential_growth() {
        let mut policy = ReconnectPolicy::new(1000, 100_000, 10);
        let d1 = policy.next_delay().unwrap();
        let d2 = policy.next_delay().unwrap();
        let d3 = policy.next_delay().unwrap();
        assert!(d2.as_millis() > d1.as_millis());
        assert!(d3.as_millis() > d2.as_millis());
    }

    #[test]
    fn test_max_delay_cap() {
        let mut policy = ReconnectPolicy::new(1000, 5000, 10);
        for _ in 0..8 {
            let _ = policy.next_delay();
        }
        let delay = policy.next_delay().unwrap();
        assert!(delay.as_millis() <= 5000);
    }

    #[test]
    fn test_max_attempts_exhaustion() {
        let mut policy = ReconnectPolicy::new(1000, 30000, 3);
        assert!(policy.next_delay().is_some());
        assert!(policy.next_delay().is_some());
        assert!(policy.next_delay().is_some());
        assert!(policy.next_delay().is_none());
        assert!(policy.is_exhausted());
    }

    #[test]
    fn test_reset() {
        let mut policy = ReconnectPolicy::new(1000, 30000, 2);
        let _ = policy.next_delay();
        let _ = policy.next_delay();
        assert!(policy.is_exhausted());
        policy.reset();
        assert_eq!(policy.attempts(), 0);
        assert!(!policy.is_exhausted());
        assert!(policy.next_delay().is_some());
    }

    #[test]
    fn test_should_reconnect() {
        assert!(!should_reconnect(Some(1000)));
        assert!(should_reconnect(Some(4000)));
        assert!(should_reconnect(Some(1006)));
        assert!(should_reconnect(None));
    }
}
