package service

import (
	"time"

	"github.com/sooraj-zebu/falcon/internal/repository"
	"github.com/sooraj-zebu/falcon/internal/storage"
)

type FileService struct {
	repo *repository.FileRepository
}

func NewFileService(repo *repository.FileRepository) *FileService {
	return &FileService{
		repo: repo,
	}
}

// SaveInventory stores file metadata into core DB
// Core side already handles UPSERT logic
func (s *FileService) SaveInventory(edgeID string, files []storage.FileInfo) error {

	for _, file := range files {

		// defensive normalization (important for consistency)
		if file.RelativePath == "" {
			continue
		}

		if file.Modified.IsZero() {
			file.Modified = time.Now()
		}

		err := s.repo.SaveFile(edgeID, file)
		if err != nil {
			return err
		}
	}

	return nil
}
