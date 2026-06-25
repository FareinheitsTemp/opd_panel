-- +goose Up
CREATE TABLE IF NOT EXISTS servers (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL UNIQUE,
    type        TEXT NOT NULL,
    version     TEXT NOT NULL,
    port        INTEGER NOT NULL,
    ram_min     INTEGER NOT NULL DEFAULT 512,
    ram_max     INTEGER NOT NULL DEFAULT 1024,
    status      TEXT NOT NULL DEFAULT 'stopped',
    created_at  DATETIME NOT NULL,
    updated_at  DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS backups (
    id          TEXT PRIMARY KEY,
    server_id   TEXT NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    path        TEXT NOT NULL,
    size_bytes  INTEGER NOT NULL DEFAULT 0,
    status      TEXT NOT NULL DEFAULT 'done',
    created_at  DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS events (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    server_id   TEXT NOT NULL,
    type        TEXT NOT NULL,
    payload     TEXT,
    created_at  DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_events_server ON events(server_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_backups_server ON backups(server_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS events;
DROP TABLE IF EXISTS backups;
DROP TABLE IF EXISTS servers;
