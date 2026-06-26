use axum::{
    body::Body,
    extract::{Path, Query, State},
    http::{header, StatusCode},
    response::{IntoResponse, Response},
    Json,
};
use chrono::Utc;
use serde::{Deserialize, Serialize};
use std::path::PathBuf;
use tokio::fs;

use super::AppState;

fn backup_dir(id: &str) -> PathBuf {
    PathBuf::from(format!("/opt/opd/backups/{}", id))
}

fn server_root(id: &str) -> PathBuf {
    PathBuf::from(format!("/opt/opd/servers/{}", id))
}

#[derive(Serialize)]
pub struct BackupEntry {
    pub filename: String,
    pub size: u64,
    pub created_at: i64,
}

// GET /servers/:id/backups
pub async fn list(
    Path(id): Path<String>,
    _state: State<AppState>,
) -> impl IntoResponse {
    let dir = backup_dir(&id);
    let _ = fs::create_dir_all(&dir).await;
    let mut entries = Vec::new();
    let mut rd = match fs::read_dir(&dir).await {
        Ok(r) => r,
        Err(e) => return (StatusCode::INTERNAL_SERVER_ERROR, Json(serde_json::json!({"error": e.to_string()}))).into_response(),
    };
    while let Ok(Some(entry)) = rd.next_entry().await {
        if let Ok(meta) = entry.metadata().await {
            if meta.is_file() {
                let modified = meta.modified().ok()
                    .and_then(|t| t.duration_since(std::time::UNIX_EPOCH).ok())
                    .map(|d| d.as_secs() as i64)
                    .unwrap_or(0);
                entries.push(BackupEntry {
                    filename: entry.file_name().to_string_lossy().to_string(),
                    size: meta.len(),
                    created_at: modified,
                });
            }
        }
    }
    entries.sort_by(|a, b| b.created_at.cmp(&a.created_at));
    Json(entries).into_response()
}

// POST /servers/:id/backups
pub async fn create(
    Path(id): Path<String>,
    _state: State<AppState>,
) -> impl IntoResponse {
    let server_dir = server_root(&id);
    let dir = backup_dir(&id);
    let _ = fs::create_dir_all(&dir).await;
    let filename = format!("{}-{}.tar.gz", id, Utc::now().format("%Y%m%d-%H%M%S"));
    let dest = dir.join(&filename);
    let output = tokio::process::Command::new("tar")
        .args(["-czf", &dest.to_string_lossy(), "-C", &server_dir.to_string_lossy(), "."])
        .output()
        .await;
    match output {
        Ok(o) if o.status.success() => {
            let size = fs::metadata(&dest).await.map(|m| m.len()).unwrap_or(0);
            Json(serde_json::json!({"ok": true, "filename": filename, "size": size})).into_response()
        }
        Ok(o) => (StatusCode::INTERNAL_SERVER_ERROR, Json(serde_json::json!({"error": String::from_utf8_lossy(&o.stderr).to_string()}))).into_response(),
        Err(e) => (StatusCode::INTERNAL_SERVER_ERROR, Json(serde_json::json!({"error": e.to_string()}))).into_response(),
    }
}

#[derive(Deserialize)]
pub struct FilenameQuery {
    pub filename: String,
}

// GET /servers/:id/backups/download?filename=xxx.tar.gz
pub async fn download(
    Path(id): Path<String>,
    Query(q): Query<FilenameQuery>,
    _state: State<AppState>,
) -> impl IntoResponse {
    let dir = backup_dir(&id);
    // Sanitize: no path separators in filename
    let safe_name = PathBuf::from(&q.filename);
    if safe_name.components().count() != 1 {
        return (StatusCode::BAD_REQUEST, "invalid filename").into_response();
    }
    let path = dir.join(&q.filename);
    match fs::read(&path).await {
        Ok(data) => Response::builder()
            .header(header::CONTENT_TYPE, "application/gzip")
            .header(header::CONTENT_DISPOSITION, format!("attachment; filename=\"{}\"", q.filename))
            .body(Body::from(data))
            .unwrap()
            .into_response(),
        Err(e) => (StatusCode::NOT_FOUND, e.to_string()).into_response(),
    }
}

// DELETE /servers/:id/backups/:filename
pub async fn delete(
    Path((id, filename)): Path<(String, String)>,
    _state: State<AppState>,
) -> impl IntoResponse {
    let dir = backup_dir(&id);
    let safe_name = PathBuf::from(&filename);
    if safe_name.components().count() != 1 {
        return (StatusCode::BAD_REQUEST, Json(serde_json::json!({"error": "invalid filename"}))).into_response();
    }
    let path = dir.join(&filename);
    match fs::remove_file(&path).await {
        Ok(_) => Json(serde_json::json!({"ok": true})).into_response(),
        Err(e) => (StatusCode::NOT_FOUND, Json(serde_json::json!({"error": e.to_string()}))).into_response(),
    }
}

// POST /servers/:id/backups/restore/:filename
pub async fn restore(
    Path((id, filename)): Path<(String, String)>,
    _state: State<AppState>,
) -> impl IntoResponse {
    let dir = backup_dir(&id);
    let safe_name = PathBuf::from(&filename);
    if safe_name.components().count() != 1 {
        return (StatusCode::BAD_REQUEST, Json(serde_json::json!({"error": "invalid filename"}))).into_response();
    }
    let src = dir.join(&filename);
    let dest = server_root(&id);
    let output = tokio::process::Command::new("tar")
        .args(["-xzf", &src.to_string_lossy(), "-C", &dest.to_string_lossy()])
        .output()
        .await;
    match output {
        Ok(o) if o.status.success() => Json(serde_json::json!({"ok": true})).into_response(),
        Ok(o) => (StatusCode::INTERNAL_SERVER_ERROR, Json(serde_json::json!({"error": String::from_utf8_lossy(&o.stderr).to_string()}))).into_response(),
        Err(e) => (StatusCode::INTERNAL_SERVER_ERROR, Json(serde_json::json!({"error": e.to_string()}))).into_response(),
    }
}
