package repository

import (
	"fmt"

	"github.com/sooraj-zebu/falcon/internal/database"
)

type EdgeRepository struct {
	db *database.Database
}

type Edge struct {
	ID        int    `json:"id"`
	EdgeID    string `json:"edge_id"`
	Name      string `json:"name"`
	Hostname  string `json:"hostname"`
	IPAddress string `json:"ip_address"`
	Status    string `json:"status"`
	Version   string `json:"version"`
	LastSeen  string `json:"last_seen"`
}

func (r *EdgeRepository) ListEdges() ([]Edge, error) {

	rows, err := r.db.DB.Query(`
	SELECT
    		id,
    		edge_id,
    		name,
    		COALESCE(hostname, ''),
    		COALESCE(ip_address, ''),
    		COALESCE(status, ''),
    		COALESCE(version, ''),
    		COALESCE(last_seen, '')
		FROM edges
	ORDER BY id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var edges []Edge

	for rows.Next() {

		var e Edge

		err := rows.Scan(
			&e.ID,
			&e.EdgeID,
			&e.Name,
			&e.Hostname,
			&e.IPAddress,
			&e.Status,
			&e.Version,
			&e.LastSeen,
		)

		if err != nil {
			return nil, err
		}

		edges = append(edges, e)
	}

	return edges, nil
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
	VALUES (?, ?, ?, ?, ?, datetime('now'), datetime('now'))

	ON CONFLICT(edge_id)
	DO UPDATE SET
		name = excluded.name,
		ip_address = excluded.ip_address,
		version = excluded.version,
		status = 'online',
		last_seen = datetime('now')
	`

	_, err := r.db.DB.Exec(
		query,
		edgeID,
		name,
		ip,
		"online",
		version,
	)

	return err
}

func (r *EdgeRepository) UpdateHeartbeat(edgeID string) error {

	query := `
	UPDATE edges
	SET
		last_seen = datetime('now'),
		status = 'online'
	WHERE edge_id = ?
	`

	_, err := r.db.DB.Exec(query, edgeID)

	return err
}
// MarkOffline marks stale edges as offline.
func (r *EdgeRepository) MarkOffline(timeoutMinutes int) error {

	query := `
	UPDATE edges
	SET status = 'offline'
	WHERE datetime(last_seen) < datetime('now', ?)
	`

	_, err := r.db.DB.Exec(
		query,
		fmt.Sprintf("-%d minutes", timeoutMinutes),
	)

	return err
}

// OnlineCount returns the number of online edges.
func (r *EdgeRepository) OnlineCount() (int, error) {

	var count int

	err := r.db.DB.QueryRow(
		"SELECT COUNT(*) FROM edges WHERE status='online'",
	).Scan(&count)

	return count, err
}
