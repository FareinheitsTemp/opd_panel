# opd-agent

A lightweight Rust daemon that runs on the game server host and manages Minecraft server processes.

## Architecture

```
opd-panel (Go) ──HTTP──▶ opd-agent (Rust)
                          │
                          ├── supervisor   spawn / stop / watchdog
                          ├── metrics      sysinfo (CPU/RAM/uptime)
                          ├── cgroup       Linux cgroup v2 limits
                          ├── config       load opd.json from disk
                          └── api          axum REST + WebSocket logs
```

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/servers` | List all managed servers |
| POST | `/servers` | Register a server by id |
| POST | `/servers/:id/start` | Start a server (loads opd.json) |
| POST | `/servers/:id/stop` | Graceful stop (sends `stop` to stdin) |
| POST | `/servers/:id/restart` | Restart (stop → watchdog auto-restarts) |
| GET | `/servers/:id/status` | Status + live metrics |
| GET | `/servers/:id/metrics` | Live metrics snapshot |
| GET | `/servers/:id/metrics/history?n=30` | Last N metric points |
| POST | `/servers/:id/console` | Send command to stdin |
| GET | `/servers/:id/logs` | WebSocket log stream |

## Auth

All requests require:
```
X-Agent-Token: <hmac-sha256 of timestamp:body>
X-Agent-Timestamp: <unix seconds>
```
Requests older than 30 seconds are rejected (replay protection).

## Server config

Each server needs an `opd.json` at `/var/lib/opd/servers/{id}/opd.json`:

```json
{
  "name": "Survival SMP",
  "port": 25565,
  "ram_min_mb": 1024,
  "ram_max_mb": 4096,
  "java_flags": [],
  "jar": "paper.jar"
}
```

## Running

```bash
OPD_AGENT_SECRET=mysecret OPD_LOG_LEVEL=info cargo run --release
```

Default bind: `127.0.0.1:7070`. Override with `OPD_AGENT_ADDR`.
