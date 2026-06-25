use anyhow::{Context, Result};
use std::fs;
use std::path::PathBuf;

/// CGroup v2 manager for a single server process.
pub struct CGroup {
    pub path: PathBuf,
}

impl CGroup {
    /// Create a new cgroup for the given server.
    pub fn create(server_id: &str, ram_mb: u64, cpu_quota_pct: u64) -> Result<Self> {
        let path = PathBuf::from(format!("/sys/fs/cgroup/opd-{}", server_id));
        fs::create_dir_all(&path)
            .with_context(|| format!("create cgroup dir {:?}", path))?;

        // RAM limit
        let ram_bytes = ram_mb * 1024 * 1024;
        fs::write(path.join("memory.max"), ram_bytes.to_string())
            .context("set memory.max")?;

        // CPU limit: cpu_quota_pct% of one core
        // Format: "{quota_us} 100000"
        let quota = cpu_quota_pct * 1000; // e.g. 50% -> 50000
        fs::write(path.join("cpu.max"), format!("{} 100000", quota))
            .context("set cpu.max")?;

        Ok(Self { path })
    }

    /// Add a PID to this cgroup.
    pub fn add_pid(&self, pid: u32) -> Result<()> {
        fs::write(self.path.join("cgroup.procs"), pid.to_string())
            .context("add pid to cgroup")?;
        Ok(())
    }

    /// Remove the cgroup (cleanup on server delete).
    pub fn remove(&self) -> Result<()> {
        // cgroup must be empty (no procs) before removal
        fs::remove_dir(&self.path)
            .with_context(|| format!("remove cgroup {:?}", self.path))?;
        Ok(())
    }
}
