package app

import (
	"encoding/json"
	"log"
	"net"
	"time"

	"github.com/sooraj-zebu/falcon/internal/client"
	"github.com/sooraj-zebu/falcon/internal/config"
	grpcserver "github.com/sooraj-zebu/falcon/internal/grpc"
	httpserver "github.com/sooraj-zebu/falcon/internal/http"
	"github.com/sooraj-zebu/falcon/internal/repository"
	"github.com/sooraj-zebu/falcon/internal/service"
	"github.com/sooraj-zebu/falcon/internal/storage"
	"github.com/sooraj-zebu/falcon/internal/watcher"
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
	normalizeEdgePorts(a.Config)

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
		info = &storage.Health{
			MountPath: a.Config.Storage.MountPath,
			Healthy:   false,
			Message:   err.Error(),
		}
	} else {
		a.Logger.Println("Storage OK:", info.MountPath)
	}

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

	go func() {
		err := grpcserver.StartEdgeTransferServer(
			a.Config.Server.GRPCPort,
			a.Config.Storage.MountPath,
			a.Logger,
		)
		if err != nil {
			a.Logger.Println("edge transfer gRPC server error:", err)
		}
	}()

	go func() {
		err := httpserver.StartEdgeControlServer(
			a.Config.Server.HTTPPort,
			a.Config.Storage.MountPath,
			a.Logger,
		)
		if err != nil {
			a.Logger.Println("edge control HTTP server error:", err)
		}
	}()

	// -------------------------
	// REGISTER EDGE
	// -------------------------
	runtime, _ := json.Marshal(map[string]any{
		"host":      outboundIP(),
		"http_port": a.Config.Server.HTTPPort,
		"grpc_port": a.Config.Server.GRPCPort,
	})

	edgeID, err := a.Client.Register(
		a.Config.Edge.Name,
		a.Config.App.Version,
		string(runtime),
	)
	if err != nil {
		a.Logger.Println("Register failed:", err)
		return
	}

	a.EdgeID = edgeID
	a.Logger.Println("Edge registered:", edgeID)
	_ = a.Client.ReportStorage(a.EdgeID, *info)

	if a.Config.Edge.DisableAutoSync {
		a.Logger.Println("Automatic sync disabled; starting heartbeat only")
		go a.startHeartbeat()
		return
	}

	// -------------------------
	// WATCHER INIT (NO FULL SCAN ANYMORE)
	// -------------------------
	if !info.Healthy {
		a.Logger.Println("Watcher disabled because storage is unhealthy:", info.Message)
		go a.startHeartbeat()
		return
	}

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
	if a.Client == nil {
		return
	}

	if err := a.sendHeartbeat(); err != nil {
		a.Logger.Println("Initial heartbeat failed:", err)
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if err := a.sendHeartbeat(); err != nil {
			a.Logger.Println("Heartbeat failed:", err)
		}
	}
}

func (a *EdgeApp) sendHeartbeat() error {
	if a.Client == nil {
		return nil
	}

	err := a.Client.Heartbeat(a.EdgeID)
	if err != nil {
		return err
	}

	manager := storage.NewManager(
		a.Config.Storage.MountPath,
		a.Logger,
	)

	info, healthErr := manager.CheckHealth()
	if healthErr != nil {
		info = &storage.Health{
			MountPath: a.Config.Storage.MountPath,
			Healthy:   false,
			Message:   healthErr.Error(),
		}
	}

	if err := a.Client.ReportStorage(a.EdgeID, *info); err != nil {
		return err
	}

	a.Logger.Println("Heartbeat sent")
	return nil
}

func normalizeEdgePorts(cfg *config.Config) {
	if cfg.Server.HTTPPort == 0 {
		cfg.Server.HTTPPort = 12168
	}
	if cfg.Server.GRPCPort == 0 {
		cfg.Server.GRPCPort = 12169
	}
	if cfg.Core.HTTPPort == 0 {
		cfg.Core.HTTPPort = 12166
	}
}

func outboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()

	localAddr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		return "127.0.0.1"
	}

	return localAddr.IP.String()
}
