# opd CLI

Minecraft server process manager — pure Go CLI + daemon.

## Quick start

```bash
# 1. Build
cd cli && go build -o opd .

# 2. Create a server config
mkdir -p /var/lib/opd/servers/survival
cp ../agent/example-server-config/opd.json /var/lib/opd/servers/survival/
# (put your server.jar there too)

# 3. Start the daemon
sudo ./opd daemon

# 4. Use CLI from another terminal
./opd status
./opd start survival
./opd logs survival
./opd metrics survival --watch
./opd console survival
./opd tui
```

## Commands

| Command | Description |
|---------|-------------|
| `opd daemon` | Start the supervisor daemon (keep running) |
| `opd status` | Table of all servers with RAM/CPU/uptime |
| `opd start <id>` | Start a server (reads `/var/lib/opd/servers/{id}/opd.json`) |
| `opd stop <id>` | Gracefully stop (sends `stop` to stdin) |
| `opd restart <id>` | Restart a server |
| `opd logs <id>` | Stream live logs to stdout |
| `opd console <id>` | Interactive stdin console + live logs |
| `opd metrics <id>` | CPU/RAM/uptime snapshot |
| `opd metrics <id> -w` | Live metrics refresh every 2s |
| `opd tui` | Full TUI dashboard |

## TUI keybindings

| Key | Action |
|-----|--------|
| `↑` / `↓` or `k` / `j` | Navigate server list |
| `s` | Start selected server |
| `x` | Stop selected server |
| `r` | Restart selected server |
| `l` | Open log stream view |
| `c` | Open interactive console |
| `q` / `Ctrl+C` | Quit |

## Architecture

```
opd daemon          — supervisor process, listens on /run/opd.sock
  └── supervisor   — manages server Handle map
        └── process.Handle  — one per server
              ├── exec.Cmd  — java process
              ├── broadcast — fan-out log channel
              └── gopsutil  — CPU/RAM metrics

opd <cmd>           — connects to /run/opd.sock, sends JSON IPC request
```

## IPC protocol

Newline-delimited JSON over a Unix domain socket.

Request: `{"cmd": "start", "server_id": "survival"}`  
Response: `{"type": "ok", "message": "starting survival"}`

Log stream: persistent connection, daemon pushes `{"type": "log", "message": "..."}` lines.
