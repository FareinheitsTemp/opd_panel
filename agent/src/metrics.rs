use std::{
    collections::HashMap,
    sync::{Arc, Mutex},
    time::{SystemTime, UNIX_EPOCH},
};

use serde::Serialize;
use sysinfo::{Pid, System};

#[derive(Clone, Serialize)]
pub struct Metrics {
    pub server_id: String,
    pub ram_used: u64,
    pub ram_max: u64,
    pub cpu: f32,
    pub uptime: u64,
    pub timestamp: i64,
}

#[derive(Clone, Serialize)]
pub struct HistoryPoint {
    pub ts: i64,
    pub cpu: f32,
    pub ram_used: u64,
}

pub type SharedMetrics = Arc<Mutex<MetricsCollector>>;

pub struct MetricsCollector {
    sys: System,
    history: HashMap<String, Vec<HistoryPoint>>,
    start_times: HashMap<String, u64>,
}

impl MetricsCollector {
    pub fn new() -> Self {
        Self {
            sys: System::new_all(),
            history: HashMap::new(),
            start_times: HashMap::new(),
        }
    }

    pub fn collect(&mut self, server_id: &str, pid: u32, ram_max: u64) -> Metrics {
        self.sys.refresh_all();
        let spid = Pid::from_u32(pid);
        let proc = self.sys.process(spid);

        let ram_used = proc.map(|p| p.memory()).unwrap_or(0);
        let cpu = proc.map(|p| p.cpu_usage()).unwrap_or(0.0);

        let now = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap_or_default()
            .as_secs();

        let start = self.start_times.entry(server_id.to_string()).or_insert(now);
        let uptime = now.saturating_sub(*start);

        let ts = now as i64;
        let point = HistoryPoint { ts, cpu, ram_used };
        self.history
            .entry(server_id.to_string())
            .or_default()
            .push(point);

        // Keep last 60 points
        let h = self.history.get_mut(server_id).unwrap();
        if h.len() > 60 { h.drain(0..h.len() - 60); }

        Metrics {
            server_id: server_id.to_string(),
            ram_used,
            ram_max,
            cpu,
            uptime,
            timestamp: ts,
        }
    }

    pub fn history(&self, server_id: &str, n: usize) -> Vec<HistoryPoint> {
        self.history
            .get(server_id)
            .map(|h| h.iter().rev().take(n).cloned().collect::<Vec<_>>())
            .unwrap_or_default()
    }
}
