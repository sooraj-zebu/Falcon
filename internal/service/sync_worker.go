package service

import (
	"fmt"
	"log"
	"time"

	"github.com/sooraj-zebu/falcon/internal/client"
	"github.com/sooraj-zebu/falcon/internal/repository"
	"github.com/sooraj-zebu/falcon/internal/storage"
	"github.com/sooraj-zebu/falcon/internal/watcher"
)

const batchSize = 50

type SyncWorker struct {
	client *client.EdgeClient
	logger *log.Logger
	repo   *repository.SyncQueueRepository
}

func NewSyncWorker(
	c *client.EdgeClient,
	r *repository.SyncQueueRepository,
	logger *log.Logger,
) *SyncWorker {

	return &SyncWorker{
		client: c,
		repo:   r,
		logger: logger,
	}
}

// watcher → queue
func (w *SyncWorker) Run(edgeID string, events <-chan watcher.Event) {

    workerID := fmt.Sprintf("%s-%d", edgeID, time.Now().UnixNano())

    go func() {

        for range time.Tick(2 * time.Second) {

            batch, err := w.repo.ClaimLeaseBatch(workerID, 20, 30*time.Second)
            if err != nil {
                w.logger.Println("claim failed:", err)
                continue
            }

            for _, file := range batch {

                err := w.client.UploadInventory(edgeID, []storage.FileInfo{file})
                if err != nil {
                    w.logger.Println("upload failed:", err)
                    continue
                }

                _ = w.repo.MarkDone(file.RelativePath, edgeID)
            }
        }
    }()

    // ingest watcher events → queue
    for e := range events {
        _ = w.repo.Enqueue(edgeID, e.File)
    }
}
