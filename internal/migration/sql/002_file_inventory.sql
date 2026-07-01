CREATE TABLE IF NOT EXISTS files (
    id INTEGER PRIMARY KEY AUTOINCREMENT,

    edge_id TEXT NOT NULL,

    relative_path TEXT NOT NULL,
    file_name TEXT NOT NULL,

    size INTEGER NOT NULL,

    modified_at DATETIME,

    checksum TEXT,

    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(edge_id, relative_path)
);
