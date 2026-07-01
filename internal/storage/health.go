package storage

import (
	"os"
	"syscall"
)

/*type Health struct {
	MountPath string

	Mounted   bool
	Writable  bool
	Readable  bool

	Filesystem string

	Total uint64
	Used  uint64
	Free  uint64

	Healthy bool
	Message string
}
*/

func (m *Manager) CheckHealth() (*Health, error){

	// Check path exists
	_, err := os.Stat(m.mountPath)
	if err != nil {
		return &Health{
			MountPath: m.mountPath,
			Healthy:   false,
			Message:   "Storage path does not exist",
		}, err
	}


	// Filesystem statistics
	var stat syscall.Statfs_t

	err = syscall.Statfs(m.mountPath, &stat)
	if err != nil {
		return &Health{
			MountPath: m.mountPath,
			Healthy:   false,
			Message:   "Unable to read filesystem information",
		}, err
	}

	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bavail * uint64(stat.Bsize)
	used := total - free

	// Write test
	writable := true

	testFile := m.mountPath + "/.falcon-write-test"

	f, err := os.Create(testFile)
	if err != nil {
		writable = false
	} else {
		f.Close()
		_ = os.Remove(testFile)
	}

	return &Health{
		MountPath: m.mountPath,

		Mounted:   true,
		Readable:  true,
		Writable:  writable,

		Filesystem: "unknown",

		Total: total,
		Used:  used,
		Free:  free,

		Healthy: writable,
		Message: "OK",
	}, nil
}

