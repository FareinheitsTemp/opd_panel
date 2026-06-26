use axum::{
    extract::State,
    response::IntoResponse,
    Json,
};
use serde::Serialize;
use std::process::Command;

use super::AppState;

#[derive(Serialize)]
pub struct DiskInfo {
    pub mount: String,
    pub total_gb: f64,
    pub used_gb: f64,
    pub free_gb: f64,
    pub used_pct: u8,
}

// GET /disks
pub async fn list(_state: State<AppState>) -> impl IntoResponse {
    // Parse `df -BG --output=target,size,used,avail,pcent`
    let output = Command::new("df")
        .args(["-BG", "--output=target,size,used,avail,pcent"])
        .output();
    let mut disks = Vec::new();
    if let Ok(out) = output {
        let text = String::from_utf8_lossy(&out.stdout);
        for line in text.lines().skip(1) {
            let cols: Vec<&str> = line.split_whitespace().collect();
            if cols.len() < 5 { continue; }
            let mount = cols[0].to_string();
            // Skip virtual filesystems
            if mount.starts_with("/proc") || mount.starts_with("/sys") || mount.starts_with("/dev/pts") || mount == "tmpfs" {
                continue;
            }
            let parse_gb = |s: &str| -> f64 {
                s.trim_end_matches('G').parse::<f64>().unwrap_or(0.0)
            };
            let total = parse_gb(cols[1]);
            let used = parse_gb(cols[2]);
            let free = parse_gb(cols[3]);
            let pct_str = cols[4].trim_end_matches('%');
            let pct = pct_str.parse::<u8>().unwrap_or(0);
            disks.push(DiskInfo { mount, total_gb: total, used_gb: used, free_gb: free, used_pct: pct });
        }
    }
    Json(disks)
}
