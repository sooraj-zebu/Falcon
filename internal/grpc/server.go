package grpc

import (
	"context"
	"log"
	"net"
	"fmt"

	pb "github.com/sooraj-zebu/falcon/proto"
	"github.com/sooraj-zebu/falcon/internal/service"
	"google.golang.org/grpc"
)

// Server holds dependencies
type Server struct {
	pb.UnimplementedFalconServiceServer
	edgeService *service.EdgeService
}

// Constructor
func NewServer(edgeService *service.EdgeService) *Server {
	return &Server{
		edgeService: edgeService,
	}
}

// RegisterEdge → real implementation
func (s *Server) RegisterEdge(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {

	log.Println("Edge registration request received")

	// Pass to service layer (IMPORTANT)
	edgeID, err := s.edgeService.RegisterEdge(req.Name, req.Version, req.IP)
	if err != nil {
		return nil, err
	}

	return &pb.RegisterResponse{
		EdgeId: edgeID,
		Status:  "registered",
	}, nil
}

// Start gRPC server
func StartServer(port int, edgeService *service.EdgeService, logger *log.Logger) error {

	lis, err := net.Listen("tcp", ":"+fmt.Sprint(port))
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer()

	pb.RegisterFalconServiceServer(
		grpcServer,
		NewServer(edgeService),
	)

	logger.Println("gRPC Server started on port", port)

	return grpcServer.Serve(lis)
}
