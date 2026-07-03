package repository

import "github.com/sooraj-zebu/falcon/internal/database"

type EdgeHealthRepository struct {
	db *database.Database
}

type EdgeHealth struct {
	EdgeID        string `json:"edge_id"`
	Name          string `json:"name"`
	Status        string `json:"status"`
	Host          string `json:"host"`
	HTTPPort      int    `json:"http_port"`
	GRPCPort      int    `json:"grpc_port"`
	MountPath     string `json:"mount_path"`
	Mounted       bool   `json:"mounted"`
	Readable      bool   `json:"readable"`
	Writable      bool   `json:"writable"`
	Healthy       bool   `json:"healthy"`
	HealthMessage string `json:"health_message"`
	TotalBytes    uint64 `json:"total_bytes"`
	UsedBytes     uint64 `json:"used_bytes"`
	FreeBytes     uint64 `json:"free_bytes"`
	LastHeartbeat string `json:"last_heartbeat"`
	UpdatedAt     string `json:"updated_at"`
}

func NewEdgeHealthRepository(db *database.Database) *EdgeHealthRepository {
	return &EdgeHealthRepository{db: db}
}

func (r *EdgeHealthRepository) UpsertRuntime(edgeID, host string, httpPort, grpcPort int) error {
	_, err := r.db.DB.Exec(`
		INSERT INTO edge_health (
			edge_id,
			host,
			http_port,
			grpc_port,
			last_heartbeat,
			updated_at
		)
		VALUES (?, ?, ?, ?, datetime('now'), datetime('now'))
		ON CONFLICT(edge_id)
		DO UPDATE SET
			host = excluded.host,
			http_port = excluded.http_port,
			grpc_port = excluded.grpc_port,
			last_heartbeat = datetime('now'),
			updated_at = datetime('now')
	`, edgeID, host, httpPort, grpcPort)
	return err
}

func (r *EdgeHealthRepository) UpsertStorage(
	edgeID string,
	mountPath string,
	mounted bool,
	readable bool,
	writable bool,
	healthy bool,
	message string,
	totalBytes uint64,
	usedBytes uint64,
	freeBytes uint64,
) error {
	_, err := r.db.DB.Exec(`
		INSERT INTO edge_health (
			edge_id,
			host,
			http_port,
			grpc_port,
			mount_path,
			mounted,
			readable,
			writable,
			healthy,
			health_message,
			total_bytes,
			used_bytes,
			free_bytes,
			updated_at
		)
		VALUES (?, '', 0, 0, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))
		ON CONFLICT(edge_id)
		DO UPDATE SET
			mount_path = excluded.mount_path,
			mounted = excluded.mounted,
			readable = excluded.readable,
			writable = excluded.writable,
			healthy = excluded.healthy,
			health_message = excluded.health_message,
			total_bytes = excluded.total_bytes,
			used_bytes = excluded.used_bytes,
			free_bytes = excluded.free_bytes,
			updated_at = datetime('now')
	`,
		edgeID,
		mountPath,
		boolToInt(mounted),
		boolToInt(readable),
		boolToInt(writable),
		boolToInt(healthy),
		message,
		totalBytes,
		usedBytes,
		freeBytes,
	)
	return err
}

func (r *EdgeHealthRepository) List() ([]EdgeHealth, error) {
	rows, err := r.db.DB.Query(`
		SELECT
			h.edge_id,
			COALESCE(e.name, ''),
			COALESCE(e.status, ''),
			h.host,
			h.http_port,
			h.grpc_port,
			COALESCE(h.mount_path, ''),
			h.mounted,
			h.readable,
			h.writable,
			h.healthy,
			COALESCE(h.health_message, ''),
			h.total_bytes,
			h.used_bytes,
			h.free_bytes,
			COALESCE(h.last_heartbeat, ''),
			COALESCE(h.updated_at, '')
		FROM edge_health h
		LEFT JOIN edges e ON e.edge_id = h.edge_id
		ORDER BY h.edge_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var health []EdgeHealth
	for rows.Next() {
		var item EdgeHealth
		var mounted, readable, writable, healthy int
		if err := rows.Scan(
			&item.EdgeID,
			&item.Name,
			&item.Status,
			&item.Host,
			&item.HTTPPort,
			&item.GRPCPort,
			&item.MountPath,
			&mounted,
			&readable,
			&writable,
			&healthy,
			&item.HealthMessage,
			&item.TotalBytes,
			&item.UsedBytes,
			&item.FreeBytes,
			&item.LastHeartbeat,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}

		item.Mounted = mounted == 1
		item.Readable = readable == 1
		item.Writable = writable == 1
		item.Healthy = healthy == 1
		health = append(health, item)
	}

	return health, rows.Err()
}

func (r *EdgeHealthRepository) Get(edgeID string) (EdgeHealth, error) {
	rows, err := r.List()
	if err != nil {
		return EdgeHealth{}, err
	}
	for _, item := range rows {
		if item.EdgeID == edgeID {
			return item, nil
		}
	}
	return EdgeHealth{}, ErrNotFound
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
