use axum::{
    extract::{Path, Query, State, WebSocketUpgrade},
    http::StatusCode,
    response::{IntoResponse, Response},
    Json,
};
use serde::{Deserialize, Serialize};

use super::AppState;
use crate::{
    config,
    metrics::HistoryPoint,
    supervisor::{
        process,
        watchdog,
        ServerStatus,
    },
};
use std::sync::Arc;
use tokio::sync::Mutex;

// ---------- response types ----------

#[derive(Serialize)]
pub struct ServerListItem {
    pub id: String,
    pub name: String,
    pub status: ServerStatus,
    pub pid: Option<u32>,
    pub port: u16,
}

#[derive(Serialize)]
pub struct ServerStatusResponse {
    pub id: String,
    pub name: String,
    pub status: ServerStatus,
    pub pid: Option<u32>,
    pub port: u16,
    pub ram_used: u64,
    pub ram_max: u64,
    pub cpu: f32,
    pub uptime: u64,
    pub timestamp: i64,
}

#[derive(Deserialize)]
pub struct ConsoleRequest {
    pub command: String,
}

#[derive(Deserialize)]
pub struct HistoryQuery {
    pub n: Option<usize>,
}

// ---------- GET /servers ----------

pub async fn list(State(state): State<AppState>) -> impl IntoResponse {
    let reg = state.registry.lock().await;
    let items: Vec<ServerListItem> = reg
        .list()
        .iter()
        .map(|h| ServerListItem {
            id: h.config.id.clone(),
            name: h.config.name.clone(),
            status: h.status.clone(),
            pid: h.pid,
            port: h.config.port,
        })
        .collect();
    Json(items)
}

// ---------- POST /servers (create — validates config exists on disk) ----------

#[derive(Deserialize)]
pub struct CreateRequest {
    pub id: String,
}

pub async fn create(
    State(state): State<AppState>,
    Json(body): Json<CreateRequest>,
) -> impl IntoResponse {
    if !config::exists(&body.id).await {
        return (
            StatusCode::NOT_FOUND,
            Json(serde_json::json!({"error": "no opd.json found for this server id"})),
        )
            .into_response();
    }
    let already = state.registry.lock().await.get(&body.id).is_some();
    if already {
        return (
            StatusCode::CONFLICT,
            Json(serde_json::json!({"error": "server already registered"})),
        )
            .into_response();
    }
    // Just register — start via /start
    Json(serde_json::json!({"status": "registered", "id": body.id})).into_response()
}

// ---------- POST /servers/:id/start ----------

pub async fn start(
    Path(id): Path<String>,
    State(state): State<AppState>,
) -> impl IntoResponse {
    // Load config from disk
    let cfg = match config::load(&id).await {
        Ok(c) => c,
        Err(e) => {
            return (
                StatusCode::NOT_FOUND,
                Json(serde_json::json!({"error": format!("{}", e)})),
            )
                .into_response()
        }
    };

    // Check not already running
    {
        let reg = state.registry.lock().await;
        if let Some(h) = reg.get(&id) {
            if h.status == ServerStatus::Running || h.status == ServerStatus::Starting {
                return (
                    StatusCode::CONFLICT,
                    Json(serde_json::json!({"error": "already running"})),
                )
                    .into_response();
            }
        }
    }

    // Spawn process
    let handle = match process::spawn(cfg.clone()).await {
        Ok(h) => h,
        Err(e) => {
            return (
                StatusCode::INTERNAL_SERVER_ERROR,
                Json(serde_json::json!({"error": format!("{}", e)})),
            )
                .into_response()
        }
    };

    // Insert into registry
    state.registry.lock().await.insert(id.clone(), handle);

    // Start watchdog
    let intentional_stop = Arc::new(Mutex::new(false));
    tokio::spawn(watchdog::watch(
        id.clone(),
        cfg,
        state.registry.clone(),
        intentional_stop,
    ));

    Json(serde_json::json!({"status": "starting", "id": id})).into_response()
}

// ---------- POST /servers/:id/stop ----------

pub async fn stop(
    Path(id): Path<String>,
    State(state): State<AppState>,
) -> impl IntoResponse {
    let mut reg = state.registry.lock().await;
    match reg.get_mut(&id) {
        Some(handle) => {
            if let Some(ref tx) = handle.stdin_tx {
                let _ = tx.try_send("stop\n".to_string());
            }
            handle.status = ServerStatus::Stopping;
            Json(serde_json::json!({"status": "stopping"})).into_response()
        }
        None => (
            StatusCode::NOT_FOUND,
            Json(serde_json::json!({"error": "server not found"})),
        )
            .into_response(),
    }
}

// ---------- POST /servers/:id/restart ----------

pub async fn restart(
    Path(id): Path<String>,
    State(state): State<AppState>,
) -> impl IntoResponse {
    {
        let mut reg = state.registry.lock().await;
        if let Some(handle) = reg.get_mut(&id) {
            if let Some(ref tx) = handle.stdin_tx {
                let _ = tx.try_send("stop\n".to_string());
            }
            handle.status = ServerStatus::Stopping;
        }
    }
    // Watchdog will auto-restart on process death
    Json(serde_json::json!({"status": "restarting"})).into_response()
}

// ---------- GET /servers/:id/status ----------

pub async fn status(
    Path(id): Path<String>,
    State(state): State<AppState>,
) -> impl IntoResponse {
    let reg = state.registry.lock().await;
    match reg.get(&id) {
        None => (
            StatusCode::NOT_FOUND,
            Json(serde_json::json!({"error": "not found"})),
        )
            .into_response(),
        Some(handle) => {
            let (ram_used, cpu, uptime, ts) = if let Some(pid) = handle.pid {
                let mut mc = state.metrics.lock().unwrap();
                let m = mc.collect(&id, pid, handle.config.ram_max_mb * 1024 * 1024);
                (m.ram_used, m.cpu, m.uptime, m.timestamp)
            } else {
                (0, 0.0, 0, 0)
            };

            Json(ServerStatusResponse {
                id: handle.config.id.clone(),
                name: handle.config.name.clone(),
                status: handle.status.clone(),
                pid: handle.pid,
                port: handle.config.port,
                ram_used,
                ram_max: handle.config.ram_max_mb * 1024 * 1024,
                cpu,
                uptime,
                timestamp: ts,
            })
            .into_response()
        }
    }
}

// ---------- GET /servers/:id/metrics ----------

pub async fn metrics_get(
    Path(id): Path<String>,
    State(state): State<AppState>,
) -> impl IntoResponse {
    let reg = state.registry.lock().await;
    match reg.get(&id) {
        None => (
            StatusCode::NOT_FOUND,
            Json(serde_json::json!({"error": "not found"})),
        )
            .into_response(),
        Some(handle) => match handle.pid {
            None => Json(serde_json::json!({"error": "not running"})).into_response(),
            Some(pid) => {
                let m = state
                    .metrics
                    .lock()
                    .unwrap()
                    .collect(&id, pid, handle.config.ram_max_mb * 1024 * 1024);
                Json(m).into_response()
            }
        },
    }
}

// ---------- GET /servers/:id/metrics/history?n=30 ----------

pub async fn metrics_history(
    Path(id): Path<String>,
    Query(q): Query<HistoryQuery>,
    State(state): State<AppState>,
) -> impl IntoResponse {
    let n = q.n.unwrap_or(30).min(60);
    let history: Vec<HistoryPoint> = state.metrics.lock().unwrap().history(&id, n);
    Json(history)
}

// ---------- POST /servers/:id/console ----------

pub async fn console_send(
    Path(id): Path<String>,
    State(state): State<AppState>,
    Json(body): Json<ConsoleRequest>,
) -> impl IntoResponse {
    let reg = state.registry.lock().await;
    match reg.get(&id) {
        Some(handle) => match &handle.stdin_tx {
            Some(tx) => {
                let cmd = format!("{}\n", body.command);
                match tx.try_send(cmd) {
                    Ok(_) => Json(serde_json::json!({"ok": true})).into_response(),
                    Err(_) => (
                        StatusCode::SERVICE_UNAVAILABLE,
                        Json(serde_json::json!({"error": "stdin buffer full"})),
                    )
                        .into_response(),
                }
            }
            None => (
                StatusCode::CONFLICT,
                Json(serde_json::json!({"error": "server not running"})),
            )
                .into_response(),
        },
        None => (
            StatusCode::NOT_FOUND,
            Json(serde_json::json!({"error": "server not found"})),
        )
            .into_response(),
    }
}

// ---------- GET /servers/:id/logs (WebSocket) ----------

pub async fn logs_ws(
    Path(id): Path<String>,
    State(state): State<AppState>,
    ws: WebSocketUpgrade,
) -> Response {
    ws.on_upgrade(move |mut socket| async move {
        let rx = {
            let reg = state.registry.lock().await;
            reg.get(&id).map(|h| h.log_tx.subscribe())
        };
        if let Some(mut rx) = rx {
            loop {
                match rx.recv().await {
                    Ok(line) => {
                        use axum::extract::ws::Message;
                        if socket.send(Message::Text(line)).await.is_err() {
                            break;
                        }
                    }
                    Err(tokio::sync::broadcast::error::RecvError::Lagged(n)) => {
                        tracing::warn!("WS client lagged by {} messages", n);
                    }
                    Err(_) => break,
                }
            }
        }
    })
}
