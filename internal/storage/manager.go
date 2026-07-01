package storage

import "log"

type Manager struct {
	mountPath string
	logger    *log.Logger
}

func NewManager(mountPath string, logger *log.Logger) *Manager {
	return &Manager{
		mountPath: mountPath,
		logger:    logger,
	}
}
