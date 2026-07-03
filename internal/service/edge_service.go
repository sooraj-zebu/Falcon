package service

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/sooraj-zebu/falcon/internal/repository"
)

type EdgeService struct {
	repo       *repository.EdgeRepository
	healthRepo *repository.EdgeHealthRepository
}

func (s *EdgeService) ListEdges() ([]repository.Edge, error) {
	return s.repo.ListEdges()
}

func NewEdgeService(repo *repository.EdgeRepository, healthRepo *repository.EdgeHealthRepository) *EdgeService {
	return &EdgeService{
		repo:       repo,
		healthRepo: healthRepo,
	}
}

// RegisterEdge is called by the gRPC layer.
func (s *EdgeService) RegisterEdge(name, version, ip string) (string, error) {

	// Temporary edge ID generation.
	// Later we'll replace this with a UUID.
	edgeID := fmt.Sprintf("edge-%s", name)

	// Save to database.
	runtime := parseEdgeRuntime(ip)

	if err := s.repo.CreateEdge(name, version, runtime.Host, edgeID); err != nil {
		return "", err
	}

	if s.healthRepo != nil {
		if err := s.healthRepo.UpsertRuntime(edgeID, runtime.Host, runtime.HTTPPort, runtime.GRPCPort); err != nil {
			return "", err
		}
	}

	return edgeID, nil
}

// Heartbeat updates the edge last_seen timestamp
func (s *EdgeService) Heartbeat(edgeID string) error {
	if err := s.repo.UpdateHeartbeat(edgeID); err != nil {
		return err
	}
	return nil
}

// MarkOffline marks inactive edges as offline.
func (s *EdgeService) MarkOffline(timeoutMinutes int) error {
	return s.repo.MarkOffline(timeoutMinutes)
}

// OnlineCount returns the number of online edges.
func (s *EdgeService) OnlineCount() (int, error) {
	return s.repo.OnlineCount()
}

func (s *EdgeService) ReportStorage(
	edgeID string,
	mountPath string,
	totalBytes uint64,
	usedBytes uint64,
	freeBytes uint64,
	writable bool,
) error {
	if s.healthRepo == nil {
		return nil
	}

	message := "OK"
	if !writable {
		message = "mount is not writable or not healthy"
	}

	return s.healthRepo.UpsertStorage(
		edgeID,
		mountPath,
		writable,
		writable,
		writable,
		writable,
		message,
		totalBytes,
		usedBytes,
		freeBytes,
	)
}

func (s *EdgeService) ListHealth() ([]repository.EdgeHealth, error) {
	if s.healthRepo == nil {
		return nil, nil
	}
	return s.healthRepo.List()
}

func (s *EdgeService) GetHealth(edgeID string) (repository.EdgeHealth, error) {
	return s.healthRepo.Get(edgeID)
}

type edgeRuntime struct {
	Host     string `json:"host"`
	HTTPPort int    `json:"http_port"`
	GRPCPort int    `json:"grpc_port"`
}

func parseEdgeRuntime(value string) edgeRuntime {
	runtime := edgeRuntime{
		Host:     value,
		HTTPPort: 12168,
		GRPCPort: 12169,
	}

	if err := json.Unmarshal([]byte(value), &runtime); err == nil && runtime.Host != "" {
		if runtime.HTTPPort == 0 {
			runtime.HTTPPort = 12168
		}
		if runtime.GRPCPort == 0 {
			runtime.GRPCPort = 12169
		}
		return runtime
	}

	parts := strings.Split(value, ":")
	if len(parts) >= 2 {
		runtime.Host = parts[0]
		if port, err := strconv.Atoi(parts[1]); err == nil {
			runtime.HTTPPort = port
		}
	}
	if len(parts) >= 3 {
		if port, err := strconv.Atoi(parts[2]); err == nil {
			runtime.GRPCPort = port
		}
	}

	return runtime
}
