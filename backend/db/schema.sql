CREATE TABLE IF NOT EXISTS downloads (
    id           TEXT PRIMARY KEY,
    channel      TEXT NOT NULL,
    url          TEXT NOT NULL,
    title        TEXT,
    status       TEXT NOT NULL DEFAULT 'pending',
    progress     INTEGER NOT NULL DEFAULT 0,
    filename     TEXT,
    error        TEXT,
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_downloads_channel ON downloads(channel);
CREATE INDEX IF NOT EXISTS idx_downloads_created_at ON downloads(created_at DESC);
