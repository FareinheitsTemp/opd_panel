use anyhow::{Context, Result};
use serde::{Deserialize, Serialize};
use std::path::{Path, PathBuf};
use tokio::fs;

use crate::supervisor::ServerConfig;

/// Each server's config lives at:
///   /var/lib/opd/servers/{id}/opd.json
#[derive(Debug, Deserialize, Serialize)]
struct RawServerConfig {
    pub name: String,
    pub port: u16,
    pub ram_min_mb: u64,
    pub ram_max_mb: u64,
    #[serde(default)]
    pub java_flags: Vec<String>,
    pub jar: Option<String>, // filename, defaults to "server.jar"
}

pub const SERVERS_ROOT: &str = "/var/lib/opd/servers";

pub async fn load(server_id: &str) -> Result<ServerConfig> {
    let dir = PathBuf::from(SERVERS_ROOT).join(server_id);
    let config_path = dir.join("opd.json");

    let raw = fs::read_to_string(&config_path)
        .await
        .with_context(|| format!("read config {:?}", config_path))?;

    let raw: RawServerConfig = serde_json::from_str(&raw)
        .with_context(|| format!("parse config {:?}", config_path))?;

    let jar_name = raw.jar.unwrap_or_else(|| "server.jar".to_string());
    let jar_path = dir.join(&jar_name);

    Ok(ServerConfig {
        id: server_id.to_string(),
        name: raw.name,
        dir,
        jar_path,
        port: raw.port,
        ram_min_mb: raw.ram_min_mb,
        ram_max_mb: raw.ram_max_mb,
        java_flags: raw.java_flags,
    })
}

pub async fn exists(server_id: &str) -> bool {
    let p = PathBuf::from(SERVERS_ROOT)
        .join(server_id)
        .join("opd.json");
    fs::metadata(p).await.is_ok()
}
