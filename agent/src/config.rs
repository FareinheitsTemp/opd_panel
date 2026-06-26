use std::path::PathBuf;
use anyhow::Result;
use serde::{Deserialize, Serialize};
use tokio::fs;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ServerConfig {
    pub id: String,
    pub name: String,
    pub port: u16,
    pub ram_min_mb: u64,
    pub ram_max_mb: u64,
    pub java_path: String,
    pub jar_file: String,
    pub work_dir: String,
    pub extra_flags: Vec<String>,
}

pub fn config_path(id: &str) -> PathBuf {
    PathBuf::from(format!("/opt/opd/servers/{}/opd.json", id))
}

pub async fn exists(id: &str) -> bool {
    fs::metadata(config_path(id)).await.is_ok()
}

pub async fn load(id: &str) -> Result<ServerConfig> {
    let data = fs::read_to_string(config_path(id)).await?;
    Ok(serde_json::from_str(&data)?)
}
