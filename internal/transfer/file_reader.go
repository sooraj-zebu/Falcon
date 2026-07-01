package transfer

import (
	"os"
)

type FileReader struct {
	file *os.File
}

func NewFileReader(path string) (*FileReader, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	return &FileReader{file: f}, nil
}

func (r *FileReader) ReadChunk(buf []byte) (int, error) {
	return r.file.Read(buf)
}

func (r *FileReader) Close() error {
	return r.file.Close()
}
