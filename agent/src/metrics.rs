use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::sync::{Arc, Mutex};
use sysinfo::{Pid, Process, System};
use chrono::Utc;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Metrics {
    pub server_id: String,
    pub pid: u32,
    pub ram_used: u64,  // bytes
    pub ram_max: u64,   // bytes (from config)
    pub cpu: f32,       // percent 0-100
    pub uptime: u64,    // seconds
    pub timestamp: i64, // unix timestamp
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct HistoryPoint {
    pub ram_used: u64,
    pub cpu: f32,
    pub timestamp: i64,
}

/// Shared metrics collector — wraps sysinfo::System.
/// Keep one instance to get accurate CPU deltas.
pub struct MetricsCollector {
    sys: System,
    history: HashMap<String, Vec<HistoryPoint>>,
    history_limit: usize,
}

impl MetricsCollector {
    pub fn new() -> Self {
        Self {
            sys: System::new_all(),
            history: HashMap::new(),
            history_limit: 60, // keep last 60 data points
        }
    }

    /// Refresh and collect metrics for a running PID.
    pub fn collect(&mut self, server_id: &str, pid: u32, ram_max: u64) -> Metrics {
        self.sys.refresh_process(Pid::from_u32(pid));

        let (ram_used, cpu, uptime) = self
            .sys
            .process(Pid::from_u32(pid))
            .map(|p| (p.memory(), p.cpu_usage(), p.run_time()))
            .unwrap_or((0, 0.0, 0));

        let ts = Utc::now().timestamp();

        // Push to history ring-buffer
        let history = self.history.entry(server_id.to_string()).or_default();
        history.push(HistoryPoint { ram_used, cpu, timestamp: ts });
        if history.len() > self.history_limit {
            history.remove(0);
        }

        Metrics {
            server_id: server_id.to_string(),
            pid,
            ram_used,
            ram_max,
            cpu,
            uptime,
            timestamp: ts,
        }
    }

    /// Return the last N points of history for a server.
    pub fn history(&self, server_id: &str, n: usize) -> Vec<HistoryPoint> {
        self.history
            .get(server_id)
            .map(|h| h.iter().rev().take(n).cloned().collect::<Vec<_>>().into_iter().rev().collect())
            .unwrap_or_default()
    }

    /// Drop history when a server is removed.
    pub fn clear(&mut self, server_id: &str) {
        self.history.remove(server_id);
    }
}

pub type SharedMetrics = Arc<Mutex<MetricsCollector>>;

pub fn new_shared() -> SharedMetrics {
    Arc::new(Mutex::new(MetricsCollector::new()))
}
