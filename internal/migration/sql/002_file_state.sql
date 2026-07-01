CREATE TABLE IF NOT EXISTS file_state (
    relative_path TEXT PRIMARY KEY,
    size INTEGER NOT NULL,
    modified_at TEXT NOT NULL,
    checksum TEXT,
    last_seen DATETIME DEFAULT CURRENT_TIMESTAMP
);
