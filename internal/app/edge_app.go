package app

import (
	"log"
	"time"

	"github.com/sooraj-zebu/falcon/internal/client"
	"github.com/sooraj-zebu/falcon/internal/config"
	"github.com/sooraj-zebu/falcon/internal/service"
	"github.com/sooraj-zebu/falcon/internal/storage"
	"github.com/sooraj-zebu/falcon/internal/watcher"
	"github.com/sooraj-zebu/falcon/internal/repository"
)

type EdgeApp struct {
	Config *config.Config
	Logger *log.Logger
	Client *client.EdgeClient

	EdgeID string

	SyncRepo *repository.SyncQueueRepository
}

func (a *EdgeApp) Start() {

	a.Logger.Println("Starting Edge:", a.Config.Edge.Name)

	// -------------------------
	// STORAGE INIT (HEALTH ONLY)
	// -------------------------
	manager := storage.NewManager(
		a.Config.Storage.MountPath,
		a.Logger,
	)

	info, err := manager.CheckHealth()
	if err != nil {
		a.Logger.Println("Storage check failed:", err)
		return
	}

	a.Logger.Println("Storage OK:", info.MountPath)

	// -------------------------
	// CONNECT TO CORE
	// -------------------------
	c, err := client.NewEdgeClient(
		a.Config.Core.Host,
		a.Config.Core.GRPCPort,
	)
	if err != nil {
		a.Logger.Println("Core connection failed:", err)
		return
	}

	a.Client = c

	// -------------------------
	// REGISTER EDGE
	// -------------------------
	edgeID, err := a.Client.Register(
		a.Config.Edge.Name,
		a.Config.App.Version,
		"127.0.0.1",
	)
	if err != nil {
		a.Logger.Println("Register failed:", err)
		return
	}

	a.EdgeID = edgeID
	a.Logger.Println("Edge registered:", edgeID)

	// -------------------------
	// WATCHER INIT (NO FULL SCAN ANYMORE)
	// -------------------------
	w, err := watcher.New(a.Config.Storage.MountPath, a.Logger)
	if err != nil {
		a.Logger.Println("Watcher init failed:", err)
		return
	}

	err = w.Start()
	if err != nil {
		a.Logger.Println("Watcher start failed:", err)
		return
	}

	// -------------------------
	// SYNC WORKER
	// -------------------------
	worker := service.NewSyncWorker(a.Client, a.SyncRepo, a.Logger)

	go worker.Run(a.EdgeID, w.Events)

	// -------------------------
	// HEARTBEAT LOOP
	// -------------------------
	go a.startHeartbeat()
}

func (a *EdgeApp) Stop() {

	a.Logger.Println("Stopping Edge...")

	if a.Client != nil {
		_ = a.Client.Close()
	}

	a.Logger.Println("Edge stopped")
}

func (a *EdgeApp) startHeartbeat() {

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {

		if a.Client == nil {
			continue
		}

		err := a.Client.Heartbeat(a.EdgeID)
		if err != nil {
			a.Logger.Println("Heartbeat failed:", err)
			continue
		}

		a.Logger.Println("Heartbeat sent")
	}
}
