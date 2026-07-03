package bootstrap

import (
	"fmt"
	"os"

	"github.com/sooraj-zebu/falcon/internal/app"
	"github.com/sooraj-zebu/falcon/internal/config"
	"github.com/sooraj-zebu/falcon/internal/database"
	"github.com/sooraj-zebu/falcon/internal/grpc"
	httpserver "github.com/sooraj-zebu/falcon/internal/http"
	"github.com/sooraj-zebu/falcon/internal/logger"
	"github.com/sooraj-zebu/falcon/internal/migration"
	"github.com/sooraj-zebu/falcon/internal/repository"
	"github.com/sooraj-zebu/falcon/internal/service"
)

func NewCore(configFile string) (*app.CoreApp, error) {

	// Load configuration
	cfg, err := config.Load(configFile)
	if err != nil {
		return nil, err
	}

	// Logger
	log := logger.New()

	// DB path
	dbPath := fmt.Sprintf("%s/falcon.db", cfg.Storage.DataDir)
	if err := os.MkdirAll(cfg.Storage.DataDir, 0755); err != nil {
		return nil, err
	}

	// Open DB
	db, err := database.New(dbPath)
	if err != nil {
		return nil, err
	}

	// Run migrations
	migrator := migration.New(db.DB)
	if err := migrator.Run(); err != nil {
		return nil, err
	}

	// Repository + Service
	edgeRepo := repository.NewEdgeRepository(db)
	edgeHealthRepo := repository.NewEdgeHealthRepository(db)
	edgeService := service.NewEdgeService(edgeRepo, edgeHealthRepo)
	fileRepo := repository.NewFileRepository(db)
	fileService := service.NewFileService(fileRepo)
	transferRepo := repository.NewTransferRepository(db)
	coreHTTPHost := cfg.Core.Host
	if coreHTTPHost == "" {
		coreHTTPHost = "127.0.0.1"
	}
	transferService := service.NewTransferService(
		transferRepo,
		edgeService,
		coreHTTPHost,
		cfg.Server.HTTPPort,
		log,
	)
	transferService.StartExistingWorkers()
	//syncRepo := repository.NewSyncQueueRepository(db)
	//storageRepo := repository.NewStorageRepository(db)
	//storageService := service.NewStorageService(storageRepo)

	// Create Core App
	core := app.NewCore(cfg, log)
	core.DB = db
	core.EdgeService = edgeService

	// 🚀 START gRPC SERVER (NEW)
	go func() {
		err := grpc.StartServer(cfg.Server.GRPCPort, edgeService, fileService, log)
		if err != nil {
			log.Println("gRPC server error:", err)
		}
	}()

	go func() {
		err := httpserver.StartServer(cfg.Server.HTTPPort, edgeService, transferService, log)
		if err != nil {
			log.Println("HTTP server error:", err)
		}
	}()

	go service.StartMonitor(edgeService, log)

	return core, nil
}
