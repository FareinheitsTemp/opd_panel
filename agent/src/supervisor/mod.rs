pub mod process;
pub mod watchdog;

use serde::{Deserialize, Serialize};
use tokio::sync::{broadcast, mpsc};

use crate::config::ServerConfig;

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum ServerStatus {
    Stopped,
    Starting,
    Running,
    Stopping,
    Crashed,
}

pub struct ServerHandle {
    pub config: ServerConfig,
    pub status: ServerStatus,
    pub pid: Option<u32>,
    pub stdin_tx: Option<mpsc::Sender<String>>,
    pub log_tx: broadcast::Sender<String>,
}
