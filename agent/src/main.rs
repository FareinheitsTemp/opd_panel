mod api;
mod auth;
mod cgroup;
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
        .with_env_filter(EnvFilter::from_env("OPD_LOG_LEVEL"))
        .init();

    let secret = std::env::var("OPD_AGENT_SECRET")
        .expect("OPD_AGENT_SECRET must be set");

    let registry = Arc::new(Mutex::new(ServerRegistry::new()));

    let app = api::build_router(registry.clone(), secret);

    let addr = "127.0.0.1:7070";
    let listener = tokio::net::TcpListener::bind(addr).await?;
    tracing::info!("opd-agent listening on {}", addr);

    axum::serve(listener, app).await?;
    Ok(())
}
