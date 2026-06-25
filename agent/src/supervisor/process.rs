use anyhow::Result;
use std::process::Stdio;
use tokio::io::{AsyncBufReadExt, AsyncWriteExt, BufReader};
use tokio::process::Command;
use tokio::sync::{broadcast, mpsc};

use super::{ServerConfig, ServerHandle, ServerStatus};

pub async fn spawn(config: ServerConfig) -> Result<ServerHandle> {
    let (log_tx, _) = broadcast::channel::<String>(1024);
    let (stdin_tx, mut stdin_rx) = mpsc::channel::<String>(64);

    let mut child = Command::new("java")
        .args(config.java_args())
        .current_dir(&config.dir)
        .stdin(Stdio::piped())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .spawn()?;

    let pid = child.id();
    tracing::info!("Spawned server {} with PID {:?}", config.id, pid);

    // Pipe stdin
    let mut child_stdin = child.stdin.take().unwrap();
    tokio::spawn(async move {
        while let Some(cmd) = stdin_rx.recv().await {
            let line = format!("{}", cmd);
            if let Err(e) = child_stdin.write_all(line.as_bytes()).await {
                tracing::error!("stdin write error: {}", e);
                break;
            }
        }
    });

    // Pipe stdout to broadcast channel
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
