pub mod server;

use std::sync::Arc;
use axum::{
    routing::{get, post},
    Router,
};
use tokio::sync::Mutex;

use crate::registry::ServerRegistry;

#[derive(Clone)]
pub struct AppState {
    pub registry: Arc<Mutex<ServerRegistry>>,
    pub secret: String,
}

pub fn build_router(registry: Arc<Mutex<ServerRegistry>>, secret: String) -> Router {
    let state = AppState { registry, secret };

    Router::new()
        .route("/servers", get(server::list))
        .route("/servers/:id/start", post(server::start))
        .route("/servers/:id/stop", post(server::stop))
        .route("/servers/:id/restart", post(server::restart))
        .route("/servers/:id/status", get(server::status))
        .route("/servers/:id/console", post(server::console_send))
        .route("/servers/:id/logs", get(server::logs_ws))
        .with_state(state)
}
