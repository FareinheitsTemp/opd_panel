use std::sync::Arc;
use tokio::sync::Mutex;
use tokio::time::{sleep, Duration};

use crate::registry::ServerRegistry;
use super::{ServerConfig, ServerStatus};
use super::process;

pub async fn watch(
    server_id: String,
    config: ServerConfig,
    registry: Arc<Mutex<ServerRegistry>>,
    intentional_stop: Arc<tokio::sync::Mutex<bool>>,
) {
    loop {
        sleep(Duration::from_secs(2)).await;

        let pid = {
            let reg = registry.lock().await;
            reg.get(&server_id).and_then(|h| h.pid)
        };

        if let Some(pid) = pid {
            // Check if process is still alive
            let alive = unsafe { libc::kill(pid as i32, 0) == 0 };
            if !alive {
                let stopped = *intentional_stop.lock().await;
                if stopped {
                    tracing::info!("{} stopped intentionally", server_id);
                    break;
                }
                tracing::warn!("{} crashed (PID {} gone), restarting in 5s", server_id, pid);

                {
                    let mut reg = registry.lock().await;
                    if let Some(h) = reg.get_mut(&server_id) {
                        h.status = ServerStatus::Crashed;
                        h.pid = None;
                    }
                }

                sleep(Duration::from_secs(5)).await;

                match process::spawn(config.clone()).await {
                    Ok(handle) => {
                        let mut reg = registry.lock().await;
                        reg.insert(server_id.clone(), handle);
                        tracing::info!("{} restarted", server_id);
                    }
                    Err(e) => {
                        tracing::error!("Failed to restart {}: {}", server_id, e);
                        break;
                    }
                }
            }
        }
    }
}
