pub mod process;
pub mod watchdog;
pub mod cgroup;

use std::path::PathBuf;
use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ServerConfig {
    pub id: String,
    pub name: String,
    pub dir: PathBuf,          // /var/lib/opd/servers/{id}/
    pub jar_path: PathBuf,     // dir/server.jar
    pub port: u16,
    pub ram_min_mb: u64,
    pub ram_max_mb: u64,
    pub java_flags: Vec<String>,
}

impl ServerConfig {
    pub fn java_args(&self) -> Vec<String> {
        let mut args = vec![
            format!("-Xms{}M", self.ram_min_mb),
            format!("-Xmx{}M", self.ram_max_mb),
            // Recommended Aikar flags for Paper
            "-XX:+UseG1GC".into(),
            "-XX:+ParallelRefProcEnabled".into(),
            "-XX:MaxGCPauseMillis=200".into(),
            "-XX:+UnlockExperimentalVMOptions".into(),
            "-XX:+DisableExplicitGC".into(),
            "-XX:G1NewSizePercent=30".into(),
            "-XX:G1MaxNewSizePercent=40".into(),
            "-XX:G1HeapRegionSize=8M".into(),
            "-XX:G1ReservePercent=20".into(),
            "-XX:G1HeapWastePercent=5".into(),
            "-XX:G1MixedGCCountTarget=4".into(),
            "-XX:InitiatingHeapOccupancyPercent=15".into(),
            "-XX:G1MixedGCLiveThresholdPercent=90".into(),
            "-XX:G1RSetUpdatingPauseTimePercent=5".into(),
            "-XX:SurvivorRatio=32".into(),
            "-XX:+PerfDisableSharedMem".into(),
            "-XX:MaxTenuringThreshold=1".into(),
        ];
        args.extend(self.java_flags.clone());
        args.extend(["-jar".into(), self.jar_path.to_string_lossy().into(), "nogui".into()]);
        args
    }
}

#[derive(Debug)]
pub struct ServerHandle {
    pub config: ServerConfig,
    pub pid: Option<u32>,
    pub status: ServerStatus,
    pub stdin_tx: Option<tokio::sync::mpsc::Sender<String>>,
    pub log_tx: tokio::sync::broadcast::Sender<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "lowercase")]
pub enum ServerStatus {
    Running,
    Stopped,
    Starting,
    Stopping,
    Crashed,
}
