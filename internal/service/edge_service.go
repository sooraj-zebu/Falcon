package service

import (
	"fmt"

	"github.com/sooraj-zebu/falcon/internal/repository"
)

type EdgeService struct {
	repo *repository.EdgeRepository
}

func (s *EdgeService) ListEdges() ([]repository.Edge, error) {
	return s.repo.ListEdges()
}

func NewEdgeService(repo *repository.EdgeRepository) *EdgeService {
	return &EdgeService{
		repo: repo,
	}
}

// RegisterEdge is called by the gRPC layer.
func (s *EdgeService) RegisterEdge(name, version, ip string) (string, error) {

	// Temporary edge ID generation.
	// Later we'll replace this with a UUID.
	edgeID := fmt.Sprintf("edge-%s", name)

	// Save to database.
	if err := s.repo.CreateEdge(name, version, ip, edgeID); err != nil {
		return "", err
	}

	return edgeID, nil
}
// Heartbeat updates the edge last_seen timestamp
func (s *EdgeService) Heartbeat(edgeID string) error {
	return s.repo.UpdateHeartbeat(edgeID)
}
// MarkOffline marks inactive edges as offline.
func (s *EdgeService) MarkOffline(timeoutMinutes int) error {
	return s.repo.MarkOffline(timeoutMinutes)
}

// OnlineCount returns the number of online edges.
func (s *EdgeService) OnlineCount() (int, error) {
	return s.repo.OnlineCount()
}
