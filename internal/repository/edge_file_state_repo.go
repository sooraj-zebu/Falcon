package repository

import (
	"database/sql"
)

type EdgeFileStateRepo struct {
	db *sql.DB
}

func NewEdgeFileStateRepo(db *sql.DB) *EdgeFileStateRepo {
	return &EdgeFileStateRepo{db: db}
}

// Check file state
func (r *EdgeFileStateRepo) Get(path string) (size int64, mod string, exists bool, err error) {

	row := r.db.QueryRow(`
		SELECT size, modified_at
		FROM file_state
		WHERE relative_path = ?
	`, path)

	err = row.Scan(&size, &mod)
	if err == sql.ErrNoRows {
		return 0, "", false, nil
	}
	if err != nil {
		return 0, "", false, err
	}

	return size, mod, true, nil
}

// Upsert file state
func (r *EdgeFileStateRepo) Upsert(path string, size int64, mod string) error {

	_, err := r.db.Exec(`
		INSERT INTO file_state(relative_path, size, modified_at)
		VALUES (?, ?, ?)
		ON CONFLICT(relative_path)
		DO UPDATE SET
			size = excluded.size,
			modified_at = excluded.modified_at
	`, path, size, mod)

	return err
}
