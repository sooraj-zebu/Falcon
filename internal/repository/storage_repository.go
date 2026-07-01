package repository

import (
	"github.com/sooraj-zebu/falcon/internal/database"
)

type StorageRepository struct {
	db *database.Database
}

func NewStorageRepository(db *database.Database) *StorageRepository {
	return &StorageRepository{db: db}
}

func (r *StorageRepository) RegisterFile(
	fileID, edgeID, path string,
	size int64,
	checksum string,
) error {

	query := `
	INSERT INTO files (file_id, edge_id, path, size, checksum)
	VALUES (?, ?, ?, ?, ?)
	`

	_, err := r.db.DB.Exec(query,
		fileID,
		edgeID,
		path,
		size,
		checksum,
	)

	return err
}

func (r *StorageRepository) CreateTransfer(
	transferID, sourceEdge, targetEdge, fileID string,
) error {

	query := `
	INSERT INTO transfers (
		transfer_id,
		source_edge,
		target_edge,
		file_id,
		status,
		progress
	)
	VALUES (?, ?, ?, ?, 'queued', 0)
	`

	_, err := r.db.DB.Exec(query,
		transferID,
		sourceEdge,
		targetEdge,
		fileID,
	)

	return err
}
