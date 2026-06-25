use axum::{
    extract::{Path, State, WebSocketUpgrade},
    http::StatusCode,
    response::{IntoResponse, Response},
    Json,
};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;

use super::AppState;
use crate::supervisor::ServerStatus;

#[derive(Serialize)]
pub struct ServerStatusResponse {
    pub id: String,
    pub status: ServerStatus,
    pub pid: Option<u32>,
    pub ram_used: u64,
    pub ram_max: u64,
    pub cpu: f32,
    pub uptime: u64,
}

#[derive(Deserialize)]
pub struct ConsoleRequest {
    pub command: String,
}

// GET /servers
pub async fn list(State(state): State<AppState>) -> impl IntoResponse {
    let reg = state.registry.lock().await;
    let servers: Vec<_> = reg
        .list()
        .iter()
        .map(|h| ServerStatusResponse {
            id: h.config.id.clone(),
            status: h.status.clone(),
            pid: h.pid,
            ram_used: 0, // TODO: metrics collector
            ram_max: h.config.ram_max_mb * 1024 * 1024,
            cpu: 0.0,
            uptime: 0,
        })
        .collect();
    Json(servers)
}

// POST /servers/:id/start
pub async fn start(Path(id): Path<String>, State(state): State<AppState>) -> impl IntoResponse {
    let reg = state.registry.lock().await;
    if reg.get(&id).is_some() {
        return (StatusCode::CONFLICT, Json(serde_json::json!({"error": "already managed"}))).into_response();
    }
    drop(reg);
    // TODO: load config from disk, call supervisor::process::spawn
    (StatusCode::OK, Json(serde_json::json!({"status": "starting"}))).into_response()
}

// POST /servers/:id/stop
pub async fn stop(Path(id): Path<String>, State(state): State<AppState>) -> impl IntoResponse {
    let mut reg = state.registry.lock().await;
    if let Some(handle) = reg.get_mut(&id) {
        if let Some(ref tx) = handle.stdin_tx {
            let _ = tx.try_send("stop".to_string());
        }
        handle.status = ServerStatus::Stopping;
        (StatusCode::OK, Json(serde_json::json!({"status": "stopping"}))).into_response()
    } else {
        (StatusCode::NOT_FOUND, Json(serde_json::json!({"error": "server not found"}))).into_response()
    }
}

// POST /servers/:id/restart
pub async fn restart(Path(id): Path<String>, State(state): State<AppState>) -> impl IntoResponse {
    // stop then start
    stop(Path(id.clone()), State(state.clone())).await;
    start(Path(id), State(state)).await
}

// GET /servers/:id/status
pub async fn status(Path(id): Path<String>, State(state): State<AppState>) -> impl IntoResponse {
    let reg = state.registry.lock().await;
    if let Some(handle) = reg.get(&id) {
        let resp = ServerStatusResponse {
            id: handle.config.id.clone(),
            status: handle.status.clone(),
            pid: handle.pid,
            ram_used: 0, // TODO: metrics
            ram_max: handle.config.ram_max_mb * 1024 * 1024,
            cpu: 0.0,
            uptime: 0,
        };
        Json(resp).into_response()
    } else {
        (StatusCode::NOT_FOUND, Json(serde_json::json!({"error": "not found"}))).into_response()
    }
}

// POST /servers/:id/console
pub async fn console_send(
    Path(id): Path<String>,
    State(state): State<AppState>,
    Json(body): Json<ConsoleRequest>,
) -> impl IntoResponse {
    let reg = state.registry.lock().await;
    if let Some(handle) = reg.get(&id) {
        if let Some(ref tx) = handle.stdin_tx {
            let cmd = format!("{}\n", body.command);
            let _ = tx.try_send(cmd);
            return (StatusCode::OK, Json(serde_json::json!({"ok": true}))).into_response();
        }
    }
    (StatusCode::NOT_FOUND, Json(serde_json::json!({"error": "server not running"}))).into_response()
}

// GET /servers/:id/logs (WebSocket)
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
            while let Ok(line) = rx.recv().await {
                use axum::extract::ws::Message;
                if socket.send(Message::Text(line)).await.is_err() {
                    break;
                }
            }
        }
    })
}
