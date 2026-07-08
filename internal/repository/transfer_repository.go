package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/sooraj-zebu/falcon/internal/database"
)

type TransferRepository struct {
	db *database.Database
}

type TransferJob struct {
	ID                int64  `json:"id"`
	JobID             string `json:"job_id"`
	SourceEdgeID      string `json:"source_edge_id"`
	DestinationEdgeID string `json:"destination_edge_id"`
	SourcePath        string `json:"source_path"`
	DestinationPath   string `json:"destination_path"`
	Status            string `json:"status"`
	CurrentFile       string `json:"current_file"`
	BytesTotal        int64  `json:"bytes_total"`
	BytesTransferred  int64  `json:"bytes_transferred"`
	ChunkSize         int64  `json:"chunk_size"`
	Attempts          int    `json:"attempts"`
	WorkerID          string `json:"worker_id"`
	ErrorMessage      string `json:"error_message"`
	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
	StartedAt         string `json:"started_at"`
	CompletedAt       string `json:"completed_at"`
}

type TransferWorkerRecord struct {
	ID           int64  `json:"id"`
	WorkerID     string `json:"worker_id"`
	Name         string `json:"name"`
	Status       string `json:"status"`
	CurrentJobID string `json:"current_job_id"`
	CreatedAt    string `json:"created_at"`
	LastSeen     string `json:"last_seen"`
}

func NewTransferRepository(db *database.Database) *TransferRepository {
	return &TransferRepository{db: db}
}

func (r *TransferRepository) CreateJob(job TransferJob) error {
	_, err := r.db.DB.Exec(`
		INSERT INTO transfer_jobs (
			job_id,
			source_edge_id,
			destination_edge_id,
			source_path,
			destination_path,
			status,
			chunk_size
		)
		VALUES (?, ?, ?, ?, ?, 'queued', ?)
	`,
		job.JobID,
		job.SourceEdgeID,
		job.DestinationEdgeID,
		job.SourcePath,
		job.DestinationPath,
		job.ChunkSize,
	)
	return err
}

func (r *TransferRepository) ListJobs(limit int) ([]TransferJob, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := r.db.DB.Query(`
		SELECT
			id,
			job_id,
			source_edge_id,
			destination_edge_id,
			source_path,
			destination_path,
			status,
			COALESCE(current_file, ''),
			bytes_total,
			bytes_transferred,
			chunk_size,
			attempts,
			COALESCE(worker_id, ''),
			COALESCE(error_message, ''),
			COALESCE(created_at, ''),
			COALESCE(updated_at, ''),
			COALESCE(started_at, ''),
			COALESCE(completed_at, '')
		FROM transfer_jobs
		ORDER BY id DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []TransferJob
	for rows.Next() {
		job, err := scanTransferJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}

	return jobs, rows.Err()
}

func (r *TransferRepository) ClaimNextJob(workerID string) (*TransferJob, error) {
	tx, err := r.db.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var jobID string
	err = tx.QueryRow(`
		SELECT job_id
		FROM transfer_jobs
		WHERE status IN ('queued', 'failed')
		ORDER BY created_at, id
		LIMIT 1
	`).Scan(&jobID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	result, err := tx.Exec(`
		UPDATE transfer_jobs
		SET
			status = 'running',
			worker_id = ?,
			attempts = attempts + 1,
			error_message = NULL,
			started_at = COALESCE(started_at, datetime('now')),
			updated_at = datetime('now')
		WHERE job_id = ?
		AND status IN ('queued', 'failed')
	`, workerID, jobID)
	if err != nil {
		return nil, err
	}

	changed, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if changed == 0 {
		return nil, nil
	}

	if _, err := tx.Exec(`
		UPDATE transfer_workers
		SET current_job_id = ?, last_seen = datetime('now')
		WHERE worker_id = ?
	`, jobID, workerID); err != nil {
		return nil, err
	}

	row := tx.QueryRow(`
		SELECT
			id,
			job_id,
			source_edge_id,
			destination_edge_id,
			source_path,
			destination_path,
			status,
			COALESCE(current_file, ''),
			bytes_total,
			bytes_transferred,
			chunk_size,
			attempts,
			COALESCE(worker_id, ''),
			COALESCE(error_message, ''),
			COALESCE(created_at, ''),
			COALESCE(updated_at, ''),
			COALESCE(started_at, ''),
			COALESCE(completed_at, '')
		FROM transfer_jobs
		WHERE job_id = ?
	`, jobID)

	job, err := scanTransferJob(row)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &job, nil
}

func (r *TransferRepository) UpdateProgress(jobID, currentFile string, total, transferred int64) error {
	_, err := r.db.DB.Exec(`
		UPDATE transfer_jobs
		SET
			current_file = ?,
			bytes_total = ?,
			bytes_transferred = ?,
			updated_at = datetime('now')
		WHERE job_id = ?
	`, currentFile, total, transferred, jobID)
	return err
}

func (r *TransferRepository) MarkJobRunning(jobID string) error {
	_, err := r.db.DB.Exec(`
		UPDATE transfer_jobs
		SET
			status = 'running',
			attempts = attempts + 1,
			started_at = COALESCE(started_at, datetime('now')),
			error_message = NULL,
			updated_at = datetime('now')
		WHERE job_id = ?
	`, jobID)
	return err
}

func (r *TransferRepository) SetStatus(
	jobID string,
	status string,
	currentFile string,
	total int64,
	transferred int64,
	message string,
) error {
	completedSQL := "NULL"
	if status == "completed" {
		completedSQL = "datetime('now')"
	}

	_, err := r.db.DB.Exec(fmt.Sprintf(`
		UPDATE transfer_jobs
		SET
			status = ?,
			current_file = COALESCE(NULLIF(?, ''), current_file),
			bytes_total = CASE WHEN ? > 0 THEN ? ELSE bytes_total END,
			bytes_transferred = CASE WHEN ? > 0 OR ? = 'completed' THEN ? ELSE bytes_transferred END,
			error_message = ?,
			updated_at = datetime('now'),
			completed_at = %s
		WHERE job_id = ?
	`, completedSQL),
		status,
		currentFile,
		total,
		total,
		transferred,
		status,
		transferred,
		message,
		jobID,
	)
	return err
}

func (r *TransferRepository) MarkJobComplete(jobID, workerID string) error {
	_, err := r.db.DB.Exec(`
		UPDATE transfer_jobs
		SET
			status = 'completed',
			bytes_transferred = bytes_total,
			error_message = NULL,
			updated_at = datetime('now'),
			completed_at = datetime('now')
		WHERE job_id = ?
	`, jobID)
	if err != nil {
		return err
	}

	return r.clearWorkerJob(workerID)
}

func (r *TransferRepository) MarkJobFailed(jobID, workerID string, cause error) error {
	message := ""
	if cause != nil {
		message = cause.Error()
	}

	_, err := r.db.DB.Exec(`
		UPDATE transfer_jobs
		SET
			status = 'failed',
			error_message = ?,
			updated_at = datetime('now')
		WHERE job_id = ?
	`, message, jobID)
	if err != nil {
		return err
	}

	return r.clearWorkerJob(workerID)
}

func (r *TransferRepository) CreateWorker(worker TransferWorkerRecord) error {
	_, err := r.db.DB.Exec(`
		INSERT INTO transfer_workers (worker_id, name, status)
		VALUES (?, ?, 'running')
	`, worker.WorkerID, worker.Name)
	return err
}

func (r *TransferRepository) ListWorkers() ([]TransferWorkerRecord, error) {
	rows, err := r.db.DB.Query(`
		SELECT
			id,
			worker_id,
			name,
			status,
			COALESCE(current_job_id, ''),
			COALESCE(created_at, ''),
			COALESCE(last_seen, '')
		FROM transfer_workers
		ORDER BY id DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workers []TransferWorkerRecord
	for rows.Next() {
		var worker TransferWorkerRecord
		if err := rows.Scan(
			&worker.ID,
			&worker.WorkerID,
			&worker.Name,
			&worker.Status,
			&worker.CurrentJobID,
			&worker.CreatedAt,
			&worker.LastSeen,
		); err != nil {
			return nil, err
		}
		workers = append(workers, worker)
	}

	return workers, rows.Err()
}

func (r *TransferRepository) WorkerHeartbeat(workerID string) error {
	_, err := r.db.DB.Exec(`
		UPDATE transfer_workers
		SET last_seen = datetime('now')
		WHERE worker_id = ?
	`, workerID)
	return err
}

func (r *TransferRepository) clearWorkerJob(workerID string) error {
	_, err := r.db.DB.Exec(`
		UPDATE transfer_workers
		SET current_job_id = NULL, last_seen = datetime('now')
		WHERE worker_id = ?
	`, workerID)
	return err
}

func (r *TransferRepository) DeleteWorker(workerID string) (int64, error) {
	res, err := r.db.DB.Exec(`
		DELETE FROM transfer_workers
		WHERE worker_id = ?
	`, workerID)
	if err != nil {
		return 0, err
	}

	return res.RowsAffected()
}

func NewTransferID() string {
	return fmt.Sprintf("transfer-%d", time.Now().UnixNano())
}

func NewTransferWorkerID() string {
	return fmt.Sprintf("worker-%d", time.Now().UnixNano())
}

type transferJobScanner interface {
	Scan(dest ...any) error
}

func scanTransferJob(scanner transferJobScanner) (TransferJob, error) {
	var job TransferJob
	err := scanner.Scan(
		&job.ID,
		&job.JobID,
		&job.SourceEdgeID,
		&job.DestinationEdgeID,
		&job.SourcePath,
		&job.DestinationPath,
		&job.Status,
		&job.CurrentFile,
		&job.BytesTotal,
		&job.BytesTransferred,
		&job.ChunkSize,
		&job.Attempts,
		&job.WorkerID,
		&job.ErrorMessage,
		&job.CreatedAt,
		&job.UpdatedAt,
		&job.StartedAt,
		&job.CompletedAt,
	)
	return job, err
}
