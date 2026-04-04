-- name: GetDownloadByID :one
SELECT * FROM downloads
WHERE id = :id;

-- name: GetDownloadsByChannel :many
SELECT * FROM downloads
WHERE channel = :channel
ORDER BY created_at DESC;

-- name: CreateDownload :one
INSERT INTO downloads (id, channel, url, status, progress)
VALUES (:id, :channel, :url, 'pending', 0)
RETURNING *;

-- name: UpdateDownloadStatus :exec
UPDATE downloads
SET status       = :status,
    progress     = :progress,
    title        = :title,
    filename     = :filename,
    error        = :error,
    completed_at = :completed_at
WHERE id = :id;
