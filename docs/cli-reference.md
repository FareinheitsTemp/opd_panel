# CLI Reference

## Global Flags

```
--socket string   Unix socket path (default: /var/run/opd/opd.sock)
--output string   Output format: table | json (default: table)
--debug           Enable debug output
```

## Server Commands

```bash
opd server list
opd server create --name <name> --type <type> --version <v> --ram <mb> --port <port>
opd server start <id>
opd server stop <id>
opd server restart <id>
opd server delete <id>
opd server info <id>                  # details + live metrics
opd server console <id>               # interactive bubbletea console
opd server logs <id> [--follow] [--lines 100]
opd server update <id>                # update jar to latest build
```

## Backup Commands

```bash
opd backup create <server_id>
opd backup list <server_id>
opd backup restore <server_id> <backup_id>
```

## Version Commands

```bash
opd versions list --type <paper|vanilla|purpur|fabric>
opd versions list --all
```

## Tunnel Commands

```bash
opd tunnel attach <server_id>
opd tunnel detach <server_id>
opd tunnel status
```

## User Commands

```bash
opd user list
opd user create --name <name> --role <admin|viewer>
opd user delete <name>
```
