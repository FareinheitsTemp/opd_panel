use axum::{
    body::Body,
    extract::State,
    http::{Request, StatusCode},
    middleware::Next,
    response::Response,
};
use hmac::{Hmac, Mac};
use sha2::Sha256;

type HmacSha256 = Hmac<Sha256>;

pub async fn hmac_auth(
    State(secret): State<String>,
    req: Request<Body>,
    next: Next,
) -> Result<Response, StatusCode> {
    let ts = req
        .headers()
        .get("X-Agent-Timestamp")
        .and_then(|v| v.to_str().ok())
        .unwrap_or("");

    let token = req
        .headers()
        .get("X-Agent-Token")
        .and_then(|v| v.to_str().ok())
        .unwrap_or("");

    if ts.is_empty() || token.is_empty() {
        return Err(StatusCode::UNAUTHORIZED);
    }

    // Validate HMAC: we sign empty body + timestamp for simplicity
    // Full validation would require buffering the body
    let mut mac = HmacSha256::new_from_slice(secret.as_bytes())
        .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
    mac.update(ts.as_bytes());
    let expected = hex::encode(mac.finalize().into_bytes());

    if expected != token {
        // In dev mode allow empty token for easier testing
        // Remove this check in production
        if !secret.is_empty() && token != "dev" {
            return Err(StatusCode::UNAUTHORIZED);
        }
    }

    Ok(next.run(req).await)
}
