package repository

import (
	"time"

	"github.com/sooraj-zebu/falcon/internal/database"
	"github.com/sooraj-zebu/falcon/internal/storage"
)

type SyncQueueRepository struct {
	db *database.Database
}

func NewSyncQueueRepository(db *database.Database) *SyncQueueRepository {
	return &SyncQueueRepository{db: db}
}

// ----------------------------
// ENQUEUE
// ----------------------------
func (r *SyncQueueRepository) Enqueue(edgeID string, f storage.FileInfo) error {

	_, err := r.db.DB.Exec(`
		INSERT INTO sync_queue (
			edge_id, relative_path, file_name, size, modified_at, status
		) VALUES (?, ?, ?, ?, ?, 'pending')
	`,
		edgeID,
		f.RelativePath,
		f.Name,
		f.Size,
		f.Modified,
	)

	return err
}

// ----------------------------
// SIMPLE FALLBACK BATCH
// ----------------------------
func (r *SyncQueueRepository) FetchBatch(limit int) ([]storage.FileInfo, error) {

	rows, err := r.db.DB.Query(`
		SELECT relative_path, file_name, size, modified_at
		FROM sync_queue
		WHERE status = 'pending'
		LIMIT ?
	`, limit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []storage.FileInfo

	for rows.Next() {
		var f storage.FileInfo

		_ = rows.Scan(
			&f.RelativePath,
			&f.Name,
			&f.Size,
			&f.Modified,
		)

		files = append(files, f)
	}

	return files, nil
}

// ----------------------------
// SIMPLE MARK DONE
// ----------------------------
func (r *SyncQueueRepository) MarkProcessed(paths []string) error {

	for _, p := range paths {
		_, _ = r.db.DB.Exec(`
			UPDATE sync_queue
			SET status='done',
			    updated_at = CURRENT_TIMESTAMP
			WHERE relative_path=?
		`, p)
	}

	return nil
}

// ----------------------------
// MARK DONE (LEASE MODE)
// ----------------------------
func (r *SyncQueueRepository) MarkDone(path string, edgeID string) error {

	_, err := r.db.DB.Exec(`
		UPDATE sync_queue
		SET status = 'done',
		    updated_at = CURRENT_TIMESTAMP
		WHERE edge_id = ? AND relative_path = ?
	`, edgeID, path)

	return err
}

// ----------------------------
// LEASE BASED CLAIM (CRASH SAFE)
// ----------------------------
func (r *SyncQueueRepository) ClaimLeaseBatch(workerID string, limit int, leaseDuration time.Duration) ([]storage.FileInfo, error) {

	tx, err := r.db.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.Query(`
		SELECT id, edge_id, relative_path, file_name, size, modified_at
		FROM sync_queue
		WHERE status = 'pending'
		   OR (status = 'processing' AND lease_until < CURRENT_TIMESTAMP)
		LIMIT ?
	`, limit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var (
		files []storage.FileInfo
		ids   []int
	)

	now := time.Now()
	leaseUntil := now.Add(leaseDuration)

	for rows.Next() {

		var (
			f      storage.FileInfo
			id     int
			edgeID string
		)

		if err := rows.Scan(
			&id,
			&edgeID,
			&f.RelativePath,
			&f.Name,
			&f.Size,
			&f.Modified,
		); err != nil {
			return nil, err
		}

		// ❌ REMOVED: f.EdgeID = edgeID (invalid struct)

		files = append(files, f)
		ids = append(ids, id)
	}

	// LOCK LEASE
	for _, id := range ids {
		_, err := tx.Exec(`
			UPDATE sync_queue
			SET status = 'processing',
			    locked_by = ?,
			    locked_at = ?,
			    lease_until = ?
			WHERE id = ?
		`, workerID, now, leaseUntil, id)

		if err != nil {
			return nil, err
		}
	}

	return files, tx.Commit()
}
