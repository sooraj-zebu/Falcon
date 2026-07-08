package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	nethttp "net/http"

	"github.com/sooraj-zebu/falcon/internal/repository"
)

const DefaultTransferChunkSize int64 = 64 * 1024 * 1024

type TransferService struct {
	repo         *repository.TransferRepository
	edgeService  *EdgeService
	coreHost     string
	coreHTTPPort int
	logger       *log.Logger
}

type CreateTransferRequest struct {
	SourceEdgeID      string `json:"source_edge_id"`
	DestinationEdgeID string `json:"destination_edge_id"`
	SourcePath        string `json:"source_path"`
	DestinationPath   string `json:"destination_path"`
	ChunkSize         int64  `json:"chunk_size"`
}

func NewTransferService(
	repo *repository.TransferRepository,
	edgeService *EdgeService,
	coreHost string,
	coreHTTPPort int,
	logger *log.Logger,
) *TransferService {
	return &TransferService{
		repo:         repo,
		edgeService:  edgeService,
		coreHost:     coreHost,
		coreHTTPPort: coreHTTPPort,
		logger:       logger,
	}
}

func (s *TransferService) CreateJob(req CreateTransferRequest) (repository.TransferJob, error) {
	if req.SourceEdgeID == "" {
		return repository.TransferJob{}, fmt.Errorf("source_edge_id is required")
	}
	if req.DestinationEdgeID == "" {
		return repository.TransferJob{}, fmt.Errorf("destination_edge_id is required")
	}
	if req.SourcePath == "" {
		return repository.TransferJob{}, fmt.Errorf("source_path is required")
	}
	if req.DestinationPath == "" {
		return repository.TransferJob{}, fmt.Errorf("destination_path is required")
	}
	if req.ChunkSize <= 0 {
		req.ChunkSize = DefaultTransferChunkSize
	}

	job := repository.TransferJob{
		JobID:             repository.NewTransferID(),
		SourceEdgeID:      req.SourceEdgeID,
		DestinationEdgeID: req.DestinationEdgeID,
		SourcePath:        req.SourcePath,
		DestinationPath:   req.DestinationPath,
		ChunkSize:         req.ChunkSize,
	}

	if err := s.repo.CreateJob(job); err != nil {
		return repository.TransferJob{}, err
	}

	if err := s.dispatchEdgeTransfer(job); err != nil {
		_ = s.repo.MarkJobFailed(job.JobID, "", err)
		return repository.TransferJob{}, err
	}

	return job, nil
}

func (s *TransferService) ListJobs(limit int) ([]repository.TransferJob, error) {
	return s.repo.ListJobs(limit)
}

func (s *TransferService) CreateWorker(name string) (repository.TransferWorkerRecord, error) {
	if name == "" {
		name = "transfer worker"
	}

	worker := repository.TransferWorkerRecord{
		WorkerID: repository.NewTransferWorkerID(),
		Name:     name,
		Status:   "running",
	}

	if err := s.repo.CreateWorker(worker); err != nil {
		return repository.TransferWorkerRecord{}, err
	}

	return worker, nil
}

func (s *TransferService) ListWorkers() ([]repository.TransferWorkerRecord, error) {
	return s.repo.ListWorkers()
}

var ErrWorkerNotFound = errors.New("worker not found")

func (s *TransferService) DeleteWorker(workerID string) error {
	rows, err := s.repo.DeleteWorker(workerID)
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrWorkerNotFound
	}
	return nil
}

func (s *TransferService) StartExistingWorkers() {
}

func (s *TransferService) Stop() {
}

func (s *TransferService) UpdateJobStatus(
	jobID string,
	status string,
	currentFile string,
	total int64,
	transferred int64,
	message string,
) error {
	return s.repo.SetStatus(jobID, status, currentFile, total, transferred, message)
}

func (s *TransferService) dispatchEdgeTransfer(job repository.TransferJob) error {
	source, err := s.edgeService.GetHealth(job.SourceEdgeID)
	if err != nil {
		return fmt.Errorf("source edge health missing: %w", err)
	}
	destination, err := s.edgeService.GetHealth(job.DestinationEdgeID)
	if err != nil {
		return fmt.Errorf("destination edge health missing: %w", err)
	}

	if source.Status != "online" {
		return fmt.Errorf("source edge is not online")
	}
	if destination.Status != "online" {
		return fmt.Errorf("destination edge is not online")
	}
	if !source.Healthy {
		return fmt.Errorf("source mount is unhealthy: %s", source.HealthMessage)
	}
	if !destination.Healthy {
		return fmt.Errorf("destination mount is unhealthy: %s", destination.HealthMessage)
	}
	if source.HTTPPort == 0 || destination.HTTPPort == 0 || destination.GRPCPort == 0 {
		return fmt.Errorf("edge runtime ports are missing")
	}

	payload := map[string]any{
		"job_id":                job.JobID,
		"source_path":           job.SourcePath,
		"destination_path":      job.DestinationPath,
		"destination_host":      destination.Host,
		"destination_http_port": destination.HTTPPort,
		"destination_grpc_port": destination.GRPCPort,
		"core_host":             s.coreHost,
		"core_http_port":        s.coreHTTPPort,
		"chunk_size":            job.ChunkSize,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	if err := s.repo.MarkJobRunning(job.JobID); err != nil {
		return err
	}

	url := fmt.Sprintf("http://%s:%d/api/v1/edge/transfers", source.Host, source.HTTPPort)
	resp, err := nethttp.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("source edge rejected transfer: %s", resp.Status)
	}

	return nil
}
