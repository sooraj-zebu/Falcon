package storage

import (
	"os"
	"path/filepath"
	"time"
)

type FileInfo struct {
	Path         string
	Name         string
	RelativePath string

	Size int64

	Modified time.Time

	IsDir bool
}
func (m *Manager) Scan() ([]FileInfo, error) {

	var files []FileInfo

	err := filepath.Walk(
		m.mountPath,
		func(path string, info os.FileInfo, err error) error {

			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			rel, _ := filepath.Rel(
				m.mountPath,
				path,
			)

			files = append(files, FileInfo{
				Path:         path,
				Name:         info.Name(),
				RelativePath: rel,
				Size:         info.Size(),
				Modified:     info.ModTime(),
				IsDir:        false,
			})

			return nil
		},
	)

	return files, err
}
