CREATE TABLE IF NOT EXISTS transfer_jobs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id TEXT UNIQUE NOT NULL,

    source_edge_id TEXT NOT NULL,
    destination_edge_id TEXT NOT NULL,
    source_path TEXT NOT NULL,
    destination_path TEXT NOT NULL,

    status TEXT NOT NULL DEFAULT 'queued',
    current_file TEXT,
    bytes_total INTEGER NOT NULL DEFAULT 0,
    bytes_transferred INTEGER NOT NULL DEFAULT 0,
    chunk_size INTEGER NOT NULL DEFAULT 67108864,

    attempts INTEGER NOT NULL DEFAULT 0,
    worker_id TEXT,
    error_message TEXT,

    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    started_at DATETIME,
    completed_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_transfer_jobs_status_created
ON transfer_jobs(status, created_at);

CREATE TABLE IF NOT EXISTS transfer_workers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    worker_id TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'running',
    current_job_id TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_seen DATETIME DEFAULT CURRENT_TIMESTAMP
);
