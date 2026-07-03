package service

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sooraj-zebu/falcon/internal/repository"
)

const DefaultTransferChunkSize int64 = 64 * 1024 * 1024

type TransferService struct {
	repo   *repository.TransferRepository
	logger *log.Logger

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

type CreateTransferRequest struct {
	SourceEdgeID      string `json:"source_edge_id"`
	DestinationEdgeID string `json:"destination_edge_id"`
	SourcePath        string `json:"source_path"`
	DestinationPath   string `json:"destination_path"`
	ChunkSize         int64  `json:"chunk_size"`
}

func NewTransferService(repo *repository.TransferRepository, logger *log.Logger) *TransferService {
	ctx, cancel := context.WithCancel(context.Background())
	return &TransferService{
		repo:   repo,
		logger: logger,
		ctx:    ctx,
		cancel: cancel,
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

	s.startWorker(worker.WorkerID)
	return worker, nil
}

func (s *TransferService) ListWorkers() ([]repository.TransferWorkerRecord, error) {
	return s.repo.ListWorkers()
}

func (s *TransferService) StartExistingWorkers() {
	workers, err := s.repo.ListWorkers()
	if err != nil {
		s.logger.Println("transfer worker restore failed:", err)
		return
	}

	for _, worker := range workers {
		if worker.Status == "running" {
			s.startWorker(worker.WorkerID)
		}
	}
}

func (s *TransferService) Stop() {
	s.cancel()
	s.wg.Wait()
}

func (s *TransferService) startWorker(workerID string) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-s.ctx.Done():
				return
			case <-ticker.C:
				if err := s.repo.WorkerHeartbeat(workerID); err != nil {
					s.logger.Println("transfer worker heartbeat failed:", err)
				}

				job, err := s.repo.ClaimNextJob(workerID)
				if err != nil {
					s.logger.Println("transfer job claim failed:", err)
					continue
				}
				if job == nil {
					continue
				}

				if err := s.copyPath(*job); err != nil {
					s.logger.Println("transfer job failed:", job.JobID, err)
					_ = s.repo.MarkJobFailed(job.JobID, workerID, err)
					continue
				}

				_ = s.repo.MarkJobComplete(job.JobID, workerID)
			}
		}
	}()
}

func (s *TransferService) copyPath(job repository.TransferJob) error {
	info, err := os.Stat(job.SourcePath)
	if err != nil {
		return err
	}

	if !info.IsDir() {
		transferred, err := s.copyFile(job, job.SourcePath, job.DestinationPath, 0, info.Size(), info.Size())
		job.BytesTransferred = transferred
		job.BytesTotal = info.Size()
		return err
	}

	total, err := directorySize(job.SourcePath)
	if err != nil {
		return err
	}

	var transferred int64
	err = filepath.WalkDir(job.SourcePath, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		rel, err := filepath.Rel(job.SourcePath, path)
		if err != nil {
			return err
		}

		dest := filepath.Join(job.DestinationPath, rel)
		if entry.IsDir() {
			return os.MkdirAll(dest, 0755)
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}

		nextTransferred, err := s.copyFile(job, path, dest, transferred, total, info.Size())
		transferred = nextTransferred
		return err
	})

	job.BytesTransferred = transferred
	job.BytesTotal = total
	return err
}

func (s *TransferService) copyFile(job repository.TransferJob, sourcePath, destinationPath string, baseTransferred, totalBytes, fileSize int64) (int64, error) {
	if err := os.MkdirAll(filepath.Dir(destinationPath), 0755); err != nil {
		return baseTransferred, err
	}

	source, err := os.Open(sourcePath)
	if err != nil {
		return baseTransferred, err
	}
	defer source.Close()

	destination, err := os.OpenFile(destinationPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return baseTransferred, err
	}
	defer destination.Close()

	offset := int64(0)
	if info, err := destination.Stat(); err == nil {
		offset = info.Size()
	}
	if offset > fileSize {
		if err := destination.Truncate(0); err != nil {
			return baseTransferred, err
		}
		offset = 0
	}

	if _, err := source.Seek(offset, io.SeekStart); err != nil {
		return baseTransferred, err
	}
	if _, err := destination.Seek(offset, io.SeekStart); err != nil {
		return baseTransferred, err
	}

	buffer := make([]byte, job.ChunkSize)
	transferred := baseTransferred + offset
	if err := s.repo.UpdateProgress(job.JobID, sourcePath, totalBytes, transferred); err != nil {
		return transferred, err
	}

	for {
		select {
		case <-s.ctx.Done():
			return transferred, s.ctx.Err()
		default:
		}

		n, readErr := source.Read(buffer)
		if n > 0 {
			if _, err := destination.Write(buffer[:n]); err != nil {
				return transferred, err
			}

			transferred += int64(n)
			if err := s.repo.UpdateProgress(job.JobID, sourcePath, totalBytes, transferred); err != nil {
				return transferred, err
			}
		}

		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return transferred, readErr
		}
	}

	return transferred, destination.Sync()
}

func directorySize(root string) (int64, error) {
	var total int64
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}

		total += info.Size()
		return nil
	})
	return total, err
}
