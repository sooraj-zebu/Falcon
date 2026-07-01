package transfer

import (
	"context"
	"fmt"
	"io"

	pb "github.com/sooraj-zebu/falcon/proto"
)

type Engine struct {
	client pb.TransferServiceClient
}

func NewEngine(client pb.TransferServiceClient) *Engine {
	return &Engine{client: client}
}

func (e *Engine) SendFile(
	ctx context.Context,
	transferID string,
	fileID string,
	reader io.Reader,
) error {

	stream, err := e.client.TransferFile(ctx)
	if err != nil {
		return err
	}

	chunks, err := ChunkFile(reader)
	if err != nil {
		return err
	}

	for _, c := range chunks {

		err := stream.Send(&pb.FileChunk{
			TransferId: transferID,
			FileId:     fileID,
			ChunkIndex:  c.Index,
			Data:        c.Data,
		})

		if err != nil {
			return err
		}
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		return err
	}

	fmt.Println("Transfer complete:", resp.Status)

	return nil
}
