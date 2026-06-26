use anyhow::Result;
use std::process::Stdio;
use tokio::io::{AsyncBufReadExt, AsyncWriteExt, BufReader};
use tokio::process::Command;
use tokio::sync::{broadcast, mpsc};

use super::{ServerHandle, ServerStatus};
use crate::config::ServerConfig;

pub async fn spawn(config: ServerConfig) -> Result<ServerHandle> {
    let (log_tx, _) = broadcast::channel::<String>(1024);
    let (stdin_tx, mut stdin_rx) = mpsc::channel::<String>(64);

    let java = if config.java_path.is_empty() {
        "java".to_string()
    } else {
        config.java_path.clone()
    };

    let mut child = Command::new(&java)
        .args(config.java_args())
        .current_dir(config.dir())
        .stdin(Stdio::piped())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .spawn()?;

    let pid = child.id();
    tracing::info!("Spawned server {} with PID {:?}", config.id, pid);

    let mut child_stdin = child.stdin.take().unwrap();
    tokio::spawn(async move {
        while let Some(cmd) = stdin_rx.recv().await {
            let line = format!("{}\n", cmd);
            if let Err(e) = child_stdin.write_all(line.as_bytes()).await {
                tracing::error!("stdin write error: {}", e);
                break;
            }
        }
    });

    let child_stdout = child.stdout.take().unwrap();
    let log_tx_clone = log_tx.clone();
    let server_id = config.id.clone();
    tokio::spawn(async move {
        let reader = BufReader::new(child_stdout);
        let mut lines = reader.lines();
        while let Ok(Some(line)) = lines.next_line().await {
            tracing::debug!("[{}] {}", server_id, line);
            let _ = log_tx_clone.send(line);
        }
    });

    Ok(ServerHandle {
        config,
        pid,
        status: ServerStatus::Starting,
        stdin_tx: Some(stdin_tx),
        log_tx,
    })
}
