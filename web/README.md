# OPD Panel — Web UI

Next.js 14 web interface for the OPD daemon.

## Setup

```bash
cd web
npm install
npm run dev
```

Open http://localhost:3000

## Requirements

The OPD daemon must be running first:

```
opd daemon
```

The daemon exposes:
- `127.0.0.1:51200` — CLI IPC
- `127.0.0.1:51201` — HTTP REST + SSE API for web UI

## Features

- Real-time server list with status, CPU, RAM, uptime
- Start / Stop / Restart buttons
- Live log streaming via Server-Sent Events
- Command input with Enter key support
- Auto-refresh every 3 seconds
- Dark red theme
