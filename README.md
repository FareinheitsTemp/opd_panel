# OPD Panel

> Open Panel Daemon — lightweight self-hosted Minecraft server manager

CLI-first, zero web UI. Runs on the same machine as your Minecraft servers.
Go daemon + Rust agent + interactive CLI.

## Architecture

```
[opd CLI] <──Unix Socket──> [opdd daemon] <──localhost HTTP──> [opd-agent (Rust)]
                                                                       │
                                                              [Java Minecraft Servers]
                                                                       │
                                                              [Playit.gg Tunnel]
```

## Stack

- **Go** — daemon (`opdd`), CLI (`opd`), version manager, tunnel integrations
- **Rust** — process supervisor, metrics, log streaming, backups
- **SQLite** — embedded database (no external deps for MVP)
- **cgroups v2** — RAM/CPU isolation without Docker
- **bubbletea** — interactive TUI console
- **Playit.gg** — free public tunnel (no port forwarding needed)

## Quick Start

```bash
curl -sSL https://raw.githubusercontent.com/FareinheitsTemp/opd_panel/main/scripts/install.sh | bash
```

## CLI Reference

```bash
opd server list
opd server create --name paper-1 --type paper --version 1.21.4 --ram 2048
opd server start paper-1
opd server stop paper-1
opd server console paper-1          # interactive bubbletea console
opd server logs paper-1 --follow
opd backup create paper-1
opd backup list paper-1
opd versions list --type paper
opd tunnel attach paper-1
```

## Supported Server Types

| Type | Source |
|------|--------|
| Vanilla | Mojang Piston API |
| Paper | PaperMC API v2 |
| Purpur | PurpurMC API |
| Fabric | FabricMC Meta API |
| Forge/NeoForge | Phase 2 |

## Roadmap

- [ ] Phase 1 — Core (daemon, agent, CLI, SQLite)
- [ ] Phase 2 — Networking (Playit.gg, DuckDNS)
- [ ] Phase 3 — Comfort (backups, metrics, auto-download jars)
- [ ] Phase 4 — Scale (multi-node, gRPC, plugin system)

## License

MIT
