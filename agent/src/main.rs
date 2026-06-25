mod api;
mod auth;
mod cgroup;
mod config;
mod registry;
mod supervisor;
mod metrics;
mod logs;

use std::sync::Arc;
use anyhow::Result;
use tokio::sync::Mutex;
use tracing_subscriber::EnvFilter;

use registry::ServerRegistry;

#[tokio::main]
async fn main() -> Result<()> {
    tracing_subscriber::fmt()
        .with_env_filter(
            EnvFilter::try_from_env("OPD_LOG_LEVEL")
                .unwrap_or_else(|_| EnvFilter::new("info")),
        )
        .init();

    let secret = std::env::var("OPD_AGENT_SECRET")
        .expect("OPD_AGENT_SECRET must be set");

    let registry = Arc::new(Mutex::new(ServerRegistry::new()));
    let metrics = metrics::new_shared();

    let app = api::build_router(registry.clone(), metrics, secret);

    let bind = std::env::var("OPD_AGENT_ADDR")
        .unwrap_or_else(|_| "127.0.0.1:7070".to_string());

    let listener = tokio::net::TcpListener::bind(&bind).await?;
    tracing::info!("opd-agent listening on {}", bind);

    axum::serve(listener, app).await?;
    Ok(())
}
