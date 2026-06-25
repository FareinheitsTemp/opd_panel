pub mod server;

use std::sync::Arc;
use axum::{
    middleware,
    routing::{get, post},
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
        .layer(middleware::from_fn_with_state(
            secret,
            crate::auth::hmac_auth,
        ))
        .with_state(state)
}
