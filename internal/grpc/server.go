package grpc

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sooraj-zebu/falcon/internal/service"
	"github.com/sooraj-zebu/falcon/internal/storage"
	pb "github.com/sooraj-zebu/falcon/proto"

	"google.golang.org/grpc"
)

type Server struct {
	pb.UnimplementedFalconServiceServer

	edgeService *service.EdgeService
	fileService *service.FileService
}

type TransferServer struct {
	pb.UnimplementedTransferServiceServer
	mountPath string
}

// NewServer
func NewServer(edgeService *service.EdgeService, fileService *service.FileService) *Server {
	return &Server{
		edgeService: edgeService,
		fileService: fileService,
	}
}

// RegisterEdge
func (s *Server) RegisterEdge(
	ctx context.Context,
	req *pb.RegisterRequest,
) (*pb.RegisterResponse, error) {

	log.Println("[gRPC] RegisterEdge")

	edgeID, err := s.edgeService.RegisterEdge(req.Name, req.Version, req.Ip)
	if err != nil {
		return nil, err
	}

	return &pb.RegisterResponse{
		EdgeId: edgeID,
		Status: "registered",
	}, nil
}

// Heartbeat
func (s *Server) Heartbeat(
	ctx context.Context,
	req *pb.HeartbeatRequest,
) (*pb.HeartbeatResponse, error) {

	if err := s.edgeService.Heartbeat(req.EdgeId); err != nil {
		return nil, err
	}

	return &pb.HeartbeatResponse{Status: "ok"}, nil
}

// ReportStorage
func (s *Server) ReportStorage(
	ctx context.Context,
	req *pb.StorageReport,
) (*pb.StorageResponse, error) {

	log.Println("[gRPC] storage report:", req.EdgeId)
	if err := s.edgeService.ReportStorage(
		req.EdgeId,
		req.MountPath,
		req.TotalBytes,
		req.UsedBytes,
		req.FreeBytes,
		req.Writable,
	); err != nil {
		return nil, err
	}

	return &pb.StorageResponse{Status: "ok"}, nil
}

// UploadInventory
func (s *Server) UploadInventory(
	stream pb.FalconService_UploadInventoryServer,
) error {

	var (
		files  []storage.FileInfo
		edgeID string
	)

	for {

		record, err := stream.Recv()

		if err == io.EOF {

			if edgeID == "" {
				return fmt.Errorf("missing edge_id")
			}

			if err := s.fileService.SaveInventory(edgeID, files); err != nil {
				return err
			}

			return stream.SendAndClose(&pb.InventoryStatus{
				Success:       true,
				FilesReceived: int32(len(files)),
			})
		}

		if err != nil {
			return err
		}

		if edgeID == "" {
			edgeID = record.EdgeId
		}

		modified, err := time.Parse(time.RFC3339, record.ModifiedAt)
		if err != nil {
			modified = time.Now()
		}

		files = append(files, storage.FileInfo{
			Name:         record.FileName,
			RelativePath: record.RelativePath,
			Size:         record.Size,
			Modified:     modified,
		})
	}
}

// StartServer
func StartServer(
	port int,
	edgeService *service.EdgeService,
	fileService *service.FileService,
	logger *log.Logger,
) error {

	lis, err := net.Listen("tcp", ":"+fmt.Sprint(port))
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer()

	pb.RegisterFalconServiceServer(
		grpcServer,
		NewServer(edgeService, fileService),
	)

	pb.RegisterTransferServiceServer(
		grpcServer,
		&TransferServer{},
	)

	logger.Println("[gRPC] running on", port)

	return grpcServer.Serve(lis)
}

func StartEdgeTransferServer(
	port int,
	mountPath string,
	logger *log.Logger,
) error {
	lis, err := net.Listen("tcp", ":"+fmt.Sprint(port))
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer()
	pb.RegisterTransferServiceServer(
		grpcServer,
		&TransferServer{mountPath: mountPath},
	)

	logger.Println("[Edge gRPC] transfer server running on", port)

	return grpcServer.Serve(lis)
}

// TransferFile
func (s *TransferServer) TransferFile(
	stream pb.TransferService_TransferFileServer,
) error {

	fmt.Println("[Transfer] receiving file")

	var (
		file     *os.File
		filePath string
	)

	for {

		chunk, err := stream.Recv()

		if err == io.EOF {

			if file != nil {
				_ = file.Close()
			}

			return stream.SendAndClose(&pb.TransferStatus{
				Status: "success",
			})
		}

		if err != nil {
			if file != nil {
				_ = file.Close()
			}
			return err
		}

		if file == nil {
			filePath, err = s.resolveDestinationPath(chunk.FileId)
			if err != nil {
				return err
			}

			if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
				return err
			}

			file, err = os.OpenFile(filePath, os.O_CREATE|os.O_RDWR, 0644)
			if err != nil {
				return err
			}

			if _, err := file.Seek(chunk.ChunkIndex, io.SeekStart); err != nil {
				return err
			}
		}

		if _, err := file.Write(chunk.Data); err != nil {
			if file != nil {
				_ = file.Close()
			}
			return err
		}
	}
}

func (s *TransferServer) resolveDestinationPath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("missing destination path")
	}

	cleanPath := filepath.Clean(path)
	if s.mountPath == "" {
		return cleanPath, nil
	}

	mountPath := filepath.Clean(s.mountPath)
	if filepath.IsAbs(cleanPath) {
		if cleanPath == mountPath || strings.HasPrefix(cleanPath, mountPath+string(os.PathSeparator)) {
			return cleanPath, nil
		}
		return "", fmt.Errorf("destination path must be inside mount path %s", mountPath)
	}

	return filepath.Join(mountPath, cleanPath), nil
}
