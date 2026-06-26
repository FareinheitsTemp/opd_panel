-- +goose Up

CREATE TABLE IF NOT EXISTS schedules (
    id          TEXT PRIMARY KEY,
    server_id   TEXT NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    cron_expr   TEXT NOT NULL,
    enabled     INTEGER NOT NULL DEFAULT 1,
    tasks       TEXT NOT NULL DEFAULT '[]',
    last_run_at DATETIME,
    created_at  DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS subusers (
    id          TEXT PRIMARY KEY,
    server_id   TEXT NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    email       TEXT NOT NULL,
    user_id     TEXT NOT NULL DEFAULT '',
    permissions TEXT NOT NULL DEFAULT '',
    created_at  DATETIME NOT NULL,
    UNIQUE(server_id, email)
);

CREATE TABLE IF NOT EXISTS allocations (
    id          TEXT PRIMARY KEY,
    server_id   TEXT NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    ip          TEXT NOT NULL,
    port        INTEGER NOT NULL UNIQUE,
    alias       TEXT NOT NULL DEFAULT '',
    is_primary  INTEGER NOT NULL DEFAULT 0,
    created_at  DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS server_databases (
    id          TEXT PRIMARY KEY,
    server_id   TEXT NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    db_name     TEXT NOT NULL UNIQUE,
    db_user     TEXT NOT NULL,
    db_pass_enc TEXT NOT NULL,
    host        TEXT NOT NULL,
    port        INTEGER NOT NULL DEFAULT 3306,
    created_at  DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_schedules_server ON schedules(server_id);
CREATE INDEX IF NOT EXISTS idx_subusers_server  ON subusers(server_id);
CREATE INDEX IF NOT EXISTS idx_alloc_server     ON allocations(server_id);
CREATE INDEX IF NOT EXISTS idx_db_server        ON server_databases(server_id);

-- +goose Down
DROP TABLE IF EXISTS server_databases;
DROP TABLE IF EXISTS allocations;
DROP TABLE IF EXISTS subusers;
DROP TABLE IF EXISTS schedules;
