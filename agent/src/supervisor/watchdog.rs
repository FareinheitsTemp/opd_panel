use std::sync::Arc;
use std::time::Duration;
use tokio::sync::Mutex;
use tokio::time::sleep;

use crate::{
    config::ServerConfig,
    registry::ServerRegistry,
    supervisor::{process, ServerStatus},
};

pub async fn watch(
    id: String,
    cfg: ServerConfig,
    registry: Arc<Mutex<ServerRegistry>>,
    intentional_stop: Arc<Mutex<bool>>,
) {
    loop {
        sleep(Duration::from_secs(2)).await;

        let (status, pid) = {
            let reg = registry.lock().await;
            match reg.get(&id) {
                Some(h) => (h.status.clone(), h.pid),
                None => return,
            }
        };

        if status == ServerStatus::Stopping || status == ServerStatus::Stopped {
            // Check if process actually died
            let dead = match pid {
                Some(p) => !is_alive(p),
                None => true,
            };
            if dead {
                let mut reg = registry.lock().await;
                if let Some(h) = reg.get_mut(&id) {
                    h.status = ServerStatus::Stopped;
                    h.pid = None;
                    h.stdin_tx = None;
                }
                let intentional = *intentional_stop.lock().await;
                if intentional {
                    return;
                }
                // Auto-restart after crash
                sleep(Duration::from_secs(5)).await;
                tracing::info!("Auto-restarting crashed server {}", id);
                match process::spawn(cfg.clone()).await {
                    Ok(handle) => {
                        let mut reg = registry.lock().await;
                        reg.insert(id.clone(), handle);
                    }
                    Err(e) => {
                        tracing::error!("Failed to restart {}: {}", id, e);
                        return;
                    }
                }
            }
        }

        if status == ServerStatus::Running {
            if let Some(p) = pid {
                if !is_alive(p) {
                    tracing::warn!("Server {} crashed (pid {})", id, p);
                    let mut reg = registry.lock().await;
                    if let Some(h) = reg.get_mut(&id) {
                        h.status = ServerStatus::Crashed;
                        h.pid = None;
                        h.stdin_tx = None;
                    }
                }
            }
        }
    }
}

fn is_alive(pid: u32) -> bool {
    #[cfg(unix)]
    {
        unsafe { libc::kill(pid as i32, 0) == 0 }
    }
    #[cfg(windows)]
    {
        use std::process::Command;
        Command::new("tasklist")
            .args(["/FI", &format!("PID eq {}", pid), "/NH"])
            .output()
            .map(|o| String::from_utf8_lossy(&o.stdout).contains(&pid.to_string()))
            .unwrap_or(false)
    }
}
