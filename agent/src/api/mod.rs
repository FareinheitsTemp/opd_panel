pub mod backup;
pub mod disks;
pub mod files;
pub mod server;

use std::sync::Arc;
use axum::{
    middleware,
    routing::{delete, get, post, put},
    Router,
};
use tokio::sync::Mutex;

use crate::metrics::SharedMetrics;
use crate::registry::ServerRegistry;

#[derive(Clone)]
pub struct AppState {
    pub registry: Arc<Mutex<ServerRegistry>>,
    pub metrics: SharedMetrics,
    pub secret: String,
}

pub fn build_router(
    registry: Arc<Mutex<ServerRegistry>>,
    metrics: SharedMetrics,
    secret: String,
) -> Router {
    let state = AppState { registry, metrics, secret: secret.clone() };

    Router::new()
        // Disks
        .route("/disks", get(disks::list))
        // Servers
        .route("/servers", get(server::list))
        .route("/servers", post(server::create))
        .route("/servers/:id/start", post(server::start))
        .route("/servers/:id/stop", post(server::stop))
        .route("/servers/:id/restart", post(server::restart))
        .route("/servers/:id/status", get(server::status))
        .route("/servers/:id/metrics", get(server::metrics_get))
        .route("/servers/:id/metrics/history", get(server::metrics_history))
        .route("/servers/:id/console", post(server::console_send))
        .route("/servers/:id/logs", get(server::logs_ws))
        // Files
        .route("/servers/:id/files", get(files::list))
        .route("/servers/:id/files", delete(files::delete))
        .route("/servers/:id/files/content", get(files::read))
        .route("/servers/:id/files/content", put(files::write))
        .route("/servers/:id/files/mkdir", post(files::mkdir))
        .route("/servers/:id/files/upload", post(files::upload))
        .route("/servers/:id/files/download", get(files::download))
        .route("/servers/:id/files/compress", post(files::compress))
        .route("/servers/:id/files/decompress", post(files::decompress))
        // Backups
        .route("/servers/:id/backups", get(backup::list))
        .route("/servers/:id/backups", post(backup::create))
        .route("/servers/:id/backups/download", get(backup::download))
        .route("/servers/:id/backups/restore/:filename", post(backup::restore))
        .route("/servers/:id/backups/:filename", delete(backup::delete))
        .layer(middleware::from_fn_with_state(
            secret,
            crate::auth::hmac_auth,
        ))
        .with_state(state)
}
