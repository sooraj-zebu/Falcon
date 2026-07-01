package repository

import (
	"github.com/sooraj-zebu/falcon/internal/database"
	"github.com/sooraj-zebu/falcon/internal/storage"
)

// FileRepository handles DB operations for files
type FileRepository struct {
	db *database.Database
}

// Constructor
func NewFileRepository(db *database.Database) *FileRepository {
	return &FileRepository{
		db: db,
	}
}

//
// =========================
// SNAPSHOT STORAGE (CURRENT STATE)
// =========================
//

// SaveFile inserts or updates file snapshot
func (r *FileRepository) SaveFile(edgeID string, file storage.FileInfo) error {

	query := `
	INSERT INTO files (
		edge_id,
		relative_path,
		file_name,
		size,
		modified_at
	)
	VALUES (?, ?, ?, ?, ?)
	ON CONFLICT(edge_id, relative_path)
	DO UPDATE SET
		file_name = excluded.file_name,
		size = excluded.size,
		modified_at = excluded.modified_at,
		updated_at = CURRENT_TIMESTAMP
	`

	_, err := r.db.DB.Exec(
		query,
		edgeID,
		file.RelativePath,
		file.Name,
		file.Size,
		file.Modified,
	)

	return err
}

//
// =========================
// EVENT LOGGING (FUTURE DELTA SYSTEM)
// =========================
//

// SaveEvent stores file change events (for delta sync system)
func (r *FileRepository) SaveEvent(edgeID string, file storage.FileInfo) error {

	query := `
	INSERT INTO file_events (
		edge_id,
		relative_path,
		file_name,
		event_type,
		size,
		modified_at
	)
	VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.DB.Exec(
		query,
		edgeID,
		file.RelativePath,
		file.Name,
		"modified", // default for now (later we compute created/modified/deleted)
		file.Size,
		file.Modified,
	)

	return err
}
