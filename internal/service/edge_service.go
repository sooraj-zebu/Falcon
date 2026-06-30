package service

import (
	"fmt"

	"github.com/sooraj-zebu/falcon/internal/repository"
)

type EdgeService struct {
	repo *repository.EdgeRepository
}

func NewEdgeService(repo *repository.EdgeRepository) *EdgeService {
	return &EdgeService{
		repo: repo,
	}
}

// RegisterEdge → called by gRPC layer
func (s *EdgeService) RegisterEdge(name, version, ip string) (string, error) {

	// Generate simple edge ID (we will improve later)
	edgeID := fmt.Sprintf("edge-%s", name)

	// Store in database via repository
	err := s.repo.CreateEdge(name, version, ip, edgeID)
	if err != nil {
		return "", err
	}

	return edgeID, nil
}
