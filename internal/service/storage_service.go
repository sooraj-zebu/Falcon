package service

import "github.com/sooraj-zebu/falcon/internal/repository"

type StorageService struct {
	repo *repository.StorageRepository
}

func NewStorageService(repo *repository.StorageRepository) *StorageService {
	return &StorageService{
		repo: repo,
	}
}
