# Architecture

## Overview

OPD Panel is split into two binaries and one agent:

- `opdd` — Go daemon, runs as a systemd service
- `opd` — Go CLI, communicates with daemon via Unix socket
- `opd-agent` — Rust daemon, manages Java processes

## Communication

```
opd CLI
  └─ Unix Socket (/var/run/opd/opd.sock)
       └─ opdd daemon
            └─ localhost HTTP (127.0.0.1:7070) + HMAC-SHA256
                 └─ opd-agent
                      └─ Java Process (tokio::process)
```

## Layers (Go — Clean Architecture)

```
Delivery  → cli/, socket/
UseCase   → usecase/
Repo      → repository/
Domain    → domain/
```

## Isolation (no Docker)

- File system: each server lives in /var/lib/opd/servers/{id}/
- RAM/CPU: cgroups v2 (memory.max, cpu.max)
- Sandbox: bubblewrap (optional, phase 2)

## Socket Protocol (CLI <-> Daemon)

JSON over Unix socket:

```json
// Request
{ "id": "req_abc", "action": "server.start", "payload": { "server_id": "paper-1" } }

// Response
{ "id": "req_abc", "ok": true, "data": { "status": "starting" } }

// Stream line (console/logs)
{ "id": "req_abc", "stream": true, "line": "[INFO] Player joined" }
```
