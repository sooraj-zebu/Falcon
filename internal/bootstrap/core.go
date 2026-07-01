package bootstrap

import (
	"fmt"

        httpserver "github.com/sooraj-zebu/falcon/internal/http"
	"github.com/sooraj-zebu/falcon/internal/app"
	"github.com/sooraj-zebu/falcon/internal/config"
	"github.com/sooraj-zebu/falcon/internal/database"
	"github.com/sooraj-zebu/falcon/internal/grpc"
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
	edgeService := service.NewEdgeService(edgeRepo)
	fileRepo := repository.NewFileRepository(db)
	fileService := service.NewFileService(fileRepo)
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
	err := httpserver.StartServer(cfg.Server.HTTPPort, edgeService, log)
	if err != nil {
		log.Println("HTTP server error:", err)
	}
	}()
	
	go service.StartMonitor(edgeService, log)
	
	return core, nil
}

