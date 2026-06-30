package repository

import (
	"time"

	"github.com/sooraj-zebu/falcon/internal/database"
)

type EdgeRepository struct {
	db *database.Database
}

func NewEdgeRepository(db *database.Database) *EdgeRepository {
	return &EdgeRepository{
		db: db,
	}
}

func (r *EdgeRepository) CreateEdge(name, version, ip, edgeID string) error {

	query := `
	INSERT INTO edges (
		edge_id,
		name,
		ip_address,
		status,
		version,
		last_seen,
		created_at
	)
	VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.DB.Exec(
		query,
		edgeID,
		name,
		ip,
		"online",
		version,
		time.Now(),
		time.Now(),
	)

	return err
}
