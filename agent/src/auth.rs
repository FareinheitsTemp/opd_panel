use axum::{
    extract::State,
    http::{Request, StatusCode},
    middleware::Next,
    response::Response,
};
use hmac::{Hmac, Mac};
use sha2::Sha256;
use std::time::{SystemTime, UNIX_EPOCH};

pub type HmacSha256 = Hmac<Sha256>;

pub async fn hmac_auth<B>(
    State(secret): State<String>,
    req: Request<B>,
    next: Next<B>,
) -> Result<Response, StatusCode> {
    let headers = req.headers();

    let token = headers
        .get("X-Agent-Token")
        .and_then(|v| v.to_str().ok())
        .ok_or(StatusCode::UNAUTHORIZED)?;

    let ts_str = headers
        .get("X-Agent-Timestamp")
        .and_then(|v| v.to_str().ok())
        .ok_or(StatusCode::UNAUTHORIZED)?;

    // Replay protection: reject if older than 30 seconds
    let ts: u64 = ts_str.parse().map_err(|_| StatusCode::UNAUTHORIZED)?;
    let now = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap()
        .as_secs();
    if now.saturating_sub(ts) > 30 {
        return Err(StatusCode::UNAUTHORIZED);
    }

    // For GET requests with no body, we still verify timestamp
    // Full body HMAC verification is done per-handler for POST requests
    let _ = (token, secret); // TODO: verify HMAC per request body in handlers

    Ok(next.run(req).await)
}
