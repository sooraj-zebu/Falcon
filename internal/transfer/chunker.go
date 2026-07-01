package transfer

import (
	"io"
)

const ChunkSize = 64 * 1024 // 64 KB

type Chunk struct {
	Index int64
	Data  []byte
}

func ChunkFile(reader io.Reader) ([]Chunk, error) {

	var chunks []Chunk
	buf := make([]byte, ChunkSize)

	var index int64 = 0

	for {
		n, err := reader.Read(buf)
		if n > 0 {
			chunks = append(chunks, Chunk{
				Index: index,
				Data:  buf[:n],
			})
			index++
		}

		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, err
		}
	}

	return chunks, nil
}
