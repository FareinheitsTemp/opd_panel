# OPD — Minecraft Server Process Manager

A lightweight CLI daemon for managing Minecraft server processes.  
Built with Go. Works on **Windows**, Linux, and macOS.

## Requirements

- [Go 1.22+](https://go.dev/dl/)
- Java 17+ (for running Minecraft servers)

## Install

### Windows (PowerShell)

```powershell
git clone https://github.com/FareinheitsTemp/opd_panel.git
cd opd_panel\cli
go mod tidy
go build -o opd.exe .

# Add opd to PATH permanently (run once, then restart terminal)
$dir = "$env:USERPROFILE\AppData\Local\Programs\opd"
New-Item -ItemType Directory -Force -Path $dir | Out-Null
Copy-Item .\opd.exe "$dir\opd.exe"
[Environment]::SetEnvironmentVariable("Path", $env:Path + ";$dir", "User")
```

Restart your terminal — `opd` now works from anywhere.

### Linux / macOS

```bash
git clone https://github.com/FareinheitsTemp/opd_panel.git
cd opd_panel
make install
```

Or manually:

```bash
cd cli && go mod tidy && go build -o opd .
sudo cp opd /usr/local/bin/opd
```

## Usage

**Step 1 — Start the daemon** (keep open, or run in background):
```
opd daemon
```

**Step 2 — Add a server:**
```
opd add my-server
```
Then place your `server.jar` in the shown directory.

**Step 3 — Control servers:**
```
opd start my-server
opd stop my-server
opd restart my-server
opd status
opd logs my-server
opd metrics my-server
opd console my-server
opd tui
opd list
opd remove my-server
```

## Data Directory

| OS | Path |
|---|---|
| Windows | `%APPDATA%\opd\servers\` |
| Linux / macOS | `/var/lib/opd/servers/` |

## IPC

The daemon listens on **TCP `127.0.0.1:51200`** — no Unix sockets, works everywhere.
