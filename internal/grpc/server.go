package grpc

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"time"
	"os"

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

// TransferFile
func (s *TransferServer) TransferFile(
	stream pb.TransferService_TransferFileServer,
) error {

	fmt.Println("[Transfer] receiving file")

	var (
		file     *os.File
		filePath = "/tmp/falcon-transfer.tmp"
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
			file, err = os.Create(filePath)
			if err != nil {
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
