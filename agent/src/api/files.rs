use axum::{
    body::Body,
    extract::{Multipart, Path, Query, State},
    http::{header, StatusCode},
    response::{IntoResponse, Response},
    Json,
};
use serde::{Deserialize, Serialize};
use std::path::PathBuf;
use tokio::fs;

use super::AppState;

#[derive(Deserialize)]
pub struct PathQuery {
    pub path: Option<String>,
}

#[derive(Serialize)]
pub struct FileEntry {
    pub name: String,
    pub is_dir: bool,
    pub size: u64,
    pub modified: i64,
}

fn server_root(id: &str) -> PathBuf {
    PathBuf::from(format!("/opt/opd/servers/{}", id))
}

fn safe_path(base: &PathBuf, rel: &str) -> Option<PathBuf> {
    let joined = base.join(rel.trim_start_matches('/'));
    if joined.starts_with(base) { Some(joined) } else { None }
}

pub async fn list(
    Path(id): Path<String>,
    Query(q): Query<PathQuery>,
    _state: State<AppState>,
) -> impl IntoResponse {
    let base = server_root(&id);
    let rel = q.path.unwrap_or_else(|| "/".to_string());
    let dir = match safe_path(&base, &rel) {
        Some(p) => p,
        None => return (StatusCode::BAD_REQUEST, Json(serde_json::json!({"error": "invalid path"}))).into_response(),
    };
    let mut entries = Vec::new();
    let mut rd = match fs::read_dir(&dir).await {
        Ok(r) => r,
        Err(e) => return (StatusCode::NOT_FOUND, Json(serde_json::json!({"error": e.to_string()}))).into_response(),
    };
    while let Ok(Some(entry)) = rd.next_entry().await {
        let meta = entry.metadata().await.unwrap();
        let modified = meta.modified().ok()
            .and_then(|t| t.duration_since(std::time::UNIX_EPOCH).ok())
            .map(|d| d.as_secs() as i64)
            .unwrap_or(0);
        entries.push(FileEntry {
            name: entry.file_name().to_string_lossy().to_string(),
            is_dir: meta.is_dir(),
            size: meta.len(),
            modified,
        });
    }
    entries.sort_by(|a, b| b.is_dir.cmp(&a.is_dir).then(a.name.cmp(&b.name)));
    Json(entries).into_response()
}

pub async fn read(
    Path(id): Path<String>,
    Query(q): Query<PathQuery>,
    _state: State<AppState>,
) -> impl IntoResponse {
    let base = server_root(&id);
    let rel = q.path.unwrap_or_default();
    let file_path = match safe_path(&base, &rel) {
        Some(p) => p,
        None => return (StatusCode::BAD_REQUEST, "invalid path").into_response(),
    };
    match fs::read_to_string(&file_path).await {
        Ok(content) => (StatusCode::OK, content).into_response(),
        Err(e) => (StatusCode::NOT_FOUND, e.to_string()).into_response(),
    }
}

#[derive(Deserialize)]
pub struct WriteBody {
    pub path: String,
    pub content: String,
}

pub async fn write(
    Path(id): Path<String>,
    _state: State<AppState>,
    Json(body): Json<WriteBody>,
) -> impl IntoResponse {
    let base = server_root(&id);
    let file_path = match safe_path(&base, &body.path) {
        Some(p) => p,
        None => return (StatusCode::BAD_REQUEST, Json(serde_json::json!({"error": "invalid path"}))).into_response(),
    };
    if let Some(parent) = file_path.parent() {
        let _ = fs::create_dir_all(parent).await;
    }
    match fs::write(&file_path, &body.content).await {
        Ok(_) => Json(serde_json::json!({"ok": true})).into_response(),
        Err(e) => (StatusCode::INTERNAL_SERVER_ERROR, Json(serde_json::json!({"error": e.to_string()}))).into_response(),
    }
}

#[derive(Deserialize)]
pub struct DeleteBody {
    pub path: String,
}

pub async fn delete(
    Path(id): Path<String>,
    _state: State<AppState>,
    Json(body): Json<DeleteBody>,
) -> impl IntoResponse {
    let base = server_root(&id);
    let target = match safe_path(&base, &body.path) {
        Some(p) => p,
        None => return (StatusCode::BAD_REQUEST, Json(serde_json::json!({"error": "invalid path"}))).into_response(),
    };
    let result = if target.is_dir() {
        fs::remove_dir_all(&target).await
    } else {
        fs::remove_file(&target).await
    };
    match result {
        Ok(_) => Json(serde_json::json!({"ok": true})).into_response(),
        Err(e) => (StatusCode::INTERNAL_SERVER_ERROR, Json(serde_json::json!({"error": e.to_string()}))).into_response(),
    }
}

#[derive(Deserialize)]
pub struct MkdirBody {
    pub path: String,
}

pub async fn mkdir(
    Path(id): Path<String>,
    _state: State<AppState>,
    Json(body): Json<MkdirBody>,
) -> impl IntoResponse {
    let base = server_root(&id);
    let dir = match safe_path(&base, &body.path) {
        Some(p) => p,
        None => return (StatusCode::BAD_REQUEST, Json(serde_json::json!({"error": "invalid path"}))).into_response(),
    };
    match fs::create_dir_all(&dir).await {
        Ok(_) => Json(serde_json::json!({"ok": true})).into_response(),
        Err(e) => (StatusCode::INTERNAL_SERVER_ERROR, Json(serde_json::json!({"error": e.to_string()}))).into_response(),
    }
}

pub async fn upload(
    Path(id): Path<String>,
    _state: State<AppState>,
    mut multipart: Multipart,
) -> impl IntoResponse {
    let base = server_root(&id);
    let mut uploaded = Vec::new();
    while let Ok(Some(field)) = multipart.next_field().await {
        let dest_rel = field.name().unwrap_or("upload").to_string();
        let filename = field.file_name().unwrap_or("file").to_string();
        let dest = match safe_path(&base, &format!("{}/{}", dest_rel, filename)) {
            Some(p) => p,
            None => continue,
        };
        if let Some(parent) = dest.parent() {
            let _ = fs::create_dir_all(parent).await;
        }
        let data = match field.bytes().await {
            Ok(b) => b,
            Err(_) => continue,
        };
        if fs::write(&dest, &data).await.is_ok() {
            uploaded.push(filename);
        }
    }
    Json(serde_json::json!({"uploaded": uploaded})).into_response()
}

pub async fn download(
    Path(id): Path<String>,
    Query(q): Query<PathQuery>,
    _state: State<AppState>,
) -> impl IntoResponse {
    let base = server_root(&id);
    let rel = q.path.unwrap_or_default();
    let file_path = match safe_path(&base, &rel) {
        Some(p) => p,
        None => return (StatusCode::BAD_REQUEST, "invalid path").into_response(),
    };
    match fs::read(&file_path).await {
        Ok(data) => {
            let filename = file_path.file_name().unwrap_or_default().to_string_lossy().to_string();
            Response::builder()
                .header(header::CONTENT_TYPE, "application/octet-stream")
                .header(header::CONTENT_DISPOSITION, format!("attachment; filename=\"{}\"", filename))
                .body(Body::from(data))
                .unwrap()
                .into_response()
        }
        Err(e) => (StatusCode::NOT_FOUND, e.to_string()).into_response(),
    }
}

#[derive(Deserialize)]
pub struct CompressBody {
    pub paths: Vec<String>,
    pub dest: String,
}

pub async fn compress(
    Path(id): Path<String>,
    _state: State<AppState>,
    Json(body): Json<CompressBody>,
) -> impl IntoResponse {
    let base = server_root(&id);
    let dest = match safe_path(&base, &body.dest) {
        Some(p) => p,
        None => return (StatusCode::BAD_REQUEST, Json(serde_json::json!({"error": "invalid dest"}))).into_response(),
    };
    let mut args = vec!["-czf".to_string(), dest.to_string_lossy().to_string()];
    for p in &body.paths {
        if let Some(abs) = safe_path(&base, p) {
            args.push(abs.to_string_lossy().to_string());
        }
    }
    let output = tokio::process::Command::new("tar").args(&args).output().await;
    match output {
        Ok(o) if o.status.success() => Json(serde_json::json!({"ok": true, "dest": body.dest})).into_response(),
        Ok(o) => (StatusCode::INTERNAL_SERVER_ERROR, Json(serde_json::json!({"error": String::from_utf8_lossy(&o.stderr).to_string()}))).into_response(),
        Err(e) => (StatusCode::INTERNAL_SERVER_ERROR, Json(serde_json::json!({"error": e.to_string()}))).into_response(),
    }
}

#[derive(Deserialize)]
pub struct DecompressBody {
    pub path: String,
    pub dest: String,
}

pub async fn decompress(
    Path(id): Path<String>,
    _state: State<AppState>,
    Json(body): Json<DecompressBody>,
) -> impl IntoResponse {
    let base = server_root(&id);
    let src = match safe_path(&base, &body.path) {
        Some(p) => p,
        None => return (StatusCode::BAD_REQUEST, Json(serde_json::json!({"error": "invalid path"}))).into_response(),
    };
    let dest_dir = match safe_path(&base, &body.dest) {
        Some(p) => p,
        None => return (StatusCode::BAD_REQUEST, Json(serde_json::json!({"error": "invalid dest"}))).into_response(),
    };
    let _ = fs::create_dir_all(&dest_dir).await;
    let output = tokio::process::Command::new("tar")
        .args(["-xzf", &src.to_string_lossy(), "-C", &dest_dir.to_string_lossy()])
        .output().await;
    match output {
        Ok(o) if o.status.success() => Json(serde_json::json!({"ok": true})).into_response(),
        Ok(o) => (StatusCode::INTERNAL_SERVER_ERROR, Json(serde_json::json!({"error": String::from_utf8_lossy(&o.stderr).to_string()}))).into_response(),
        Err(e) => (StatusCode::INTERNAL_SERVER_ERROR, Json(serde_json::json!({"error": e.to_string()}))).into_response(),
    }
}
