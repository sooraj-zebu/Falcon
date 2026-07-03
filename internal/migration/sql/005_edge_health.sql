CREATE TABLE IF NOT EXISTS edge_health (
    edge_id TEXT PRIMARY KEY,
    host TEXT NOT NULL,
    http_port INTEGER NOT NULL,
    grpc_port INTEGER NOT NULL,

    mount_path TEXT,
    mounted INTEGER NOT NULL DEFAULT 0,
    readable INTEGER NOT NULL DEFAULT 0,
    writable INTEGER NOT NULL DEFAULT 0,
    healthy INTEGER NOT NULL DEFAULT 0,
    health_message TEXT,

    total_bytes INTEGER NOT NULL DEFAULT 0,
    used_bytes INTEGER NOT NULL DEFAULT 0,
    free_bytes INTEGER NOT NULL DEFAULT 0,

    last_heartbeat DATETIME,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_edge_health_updated
ON edge_health(updated_at);
