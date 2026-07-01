CREATE TABLE IF NOT EXISTS sync_queue (
    id INTEGER PRIMARY KEY AUTOINCREMENT,

    edge_id TEXT NOT NULL,
    relative_path TEXT NOT NULL,
    file_name TEXT NOT NULL,
    size INTEGER NOT NULL,
    modified_at DATETIME NOT NULL,

    status TEXT NOT NULL DEFAULT 'pending',

    -- 🧠 LEASE FIELDS (NEW)
    locked_by TEXT DEFAULT NULL,
    locked_at DATETIME DEFAULT NULL,
    lease_until DATETIME DEFAULT NULL,

    retry_count INTEGER NOT NULL DEFAULT 0,

    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance + recovery
CREATE INDEX IF NOT EXISTS idx_sync_status ON sync_queue(status);
CREATE INDEX IF NOT EXISTS idx_sync_lease ON sync_queue(lease_until);
CREATE INDEX IF NOT EXISTS idx_sync_locked_by ON sync_queue(locked_by);
