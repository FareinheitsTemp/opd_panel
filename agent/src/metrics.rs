use serde::{Deserialize, Serialize};
use sysinfo::{Pid, System};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Metrics {
    pub server_id: String,
    pub pid: u32,
    pub ram_used: u64,   // bytes
    pub ram_max: u64,    // bytes (from config)
    pub cpu: f32,        // percent 0-100
    pub uptime: u64,     // seconds
}

pub struct MetricsCollector {
    sys: System,
}

impl MetricsCollector {
    pub fn new() -> Self {
        Self { sys: System::new_all() }
    }

    pub fn collect(&mut self, server_id: &str, pid: u32, ram_max: u64) -> Metrics {
        self.sys.refresh_process(Pid::from_u32(pid));

        let process = self.sys.process(Pid::from_u32(pid));

        let (ram_used, cpu, uptime) = process
            .map(|p| (p.memory(), p.cpu_usage(), p.run_time()))
            .unwrap_or((0, 0.0, 0));

        Metrics {
            server_id: server_id.to_string(),
            pid,
            ram_used,
            ram_max,
            cpu,
            uptime,
        }
    }
}
