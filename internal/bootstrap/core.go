package bootstrap

import (
	"fmt"

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

	// Create Core App
	core := app.NewCore(cfg, log)
	core.DB = db
	core.EdgeService = edgeService

	// 🚀 START gRPC SERVER (NEW)
	go func() {
		err := grpc.StartServer(cfg.Server.GRPCPort, edgeService, log)
		if err != nil {
			log.Println("gRPC server error:", err)
		}
	}()

	return core, nil
}
