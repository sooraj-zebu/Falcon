package service

import (
	"log"

	"github.com/sooraj-zebu/falcon/internal/client"
	"github.com/sooraj-zebu/falcon/internal/repository"
	"github.com/sooraj-zebu/falcon/internal/storage"
	"github.com/sooraj-zebu/falcon/internal/watcher"
)

type SyncService struct {
	client *client.EdgeClient
	repo   *repository.SyncQueueRepository
	logger *log.Logger
}

func NewSyncService(
	c *client.EdgeClient,
	r *repository.SyncQueueRepository,
	logger *log.Logger,
) *SyncService {
	return &SyncService{
		client: c,
		repo:   r,
		logger: logger,
	}
}

// enqueue from watcher
func (s *SyncService) Enqueue(edgeID string, f storage.FileInfo) error {
	return s.repo.Enqueue(edgeID, f)
}

// worker loop
func (s *SyncService) RunWorker(edgeID string, events <-chan watcher.Event) {

	for e := range events {

		err := s.repo.Enqueue(edgeID, e.File)
		if err != nil {
			s.logger.Println("enqueue failed:", err)
		}
	}
}
