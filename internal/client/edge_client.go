package client

import (
	"context"
	"fmt"
	"time"

	pb "github.com/sooraj-zebu/falcon/proto"

	"github.com/sooraj-zebu/falcon/internal/storage"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type EdgeClient struct {
	conn     *grpc.ClientConn
	client   pb.FalconServiceClient
	transfer pb.TransferServiceClient
}

// NewEdgeClient
func NewEdgeClient(host string, port int) (*EdgeClient, error) {

	addr := fmt.Sprintf("%s:%d", host, port)

	conn, err := grpc.Dial(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	return &EdgeClient{
		conn:     conn,
		client:   pb.NewFalconServiceClient(conn),
		transfer: pb.NewTransferServiceClient(conn),
	}, nil
}

// Register edge
func (c *EdgeClient) Register(name, version, ip string) (string, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := c.client.RegisterEdge(ctx, &pb.RegisterRequest{
		Name:    name,
		Version: version,
		Ip:      ip,
	})
	if err != nil {
		return "", err
	}

	return resp.EdgeId, nil
}

// Heartbeat
func (c *EdgeClient) Heartbeat(edgeID string) error {

	_, err := c.client.Heartbeat(
		context.Background(),
		&pb.HeartbeatRequest{
			EdgeId: edgeID,
		},
	)

	return err
}

func (c *EdgeClient) ReportStorage(edgeID string, health storage.Health) error {
	_, err := c.client.ReportStorage(
		context.Background(),
		&pb.StorageReport{
			EdgeId:     edgeID,
			MountPath:  health.MountPath,
			TotalBytes: health.Total,
			UsedBytes:  health.Used,
			FreeBytes:  health.Free,
			Writable:   health.Healthy && health.Writable,
		},
	)
	return err
}

// UploadInventory
func (c *EdgeClient) UploadInventory(
	edgeID string,
	files []storage.FileInfo,
) error {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	stream, err := c.client.UploadInventory(ctx)
	if err != nil {
		return err
	}

	for _, file := range files {

		if err := stream.Send(&pb.FileRecord{
			EdgeId:       edgeID,
			RelativePath: file.RelativePath,
			FileName:     file.Name,
			Size:         file.Size,
			ModifiedAt:   file.Modified.Format(time.RFC3339),
		}); err != nil {
			return err
		}
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("inventory upload failed")
	}

	return nil
}

// Close gRPC connection
func (c *EdgeClient) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}
