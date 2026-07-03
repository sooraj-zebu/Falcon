package bootstrap

import (
	"os"

	"github.com/sooraj-zebu/falcon/internal/app"
	"github.com/sooraj-zebu/falcon/internal/client"
	"github.com/sooraj-zebu/falcon/internal/config"
	"github.com/sooraj-zebu/falcon/internal/database"
	"github.com/sooraj-zebu/falcon/internal/logger"
	"github.com/sooraj-zebu/falcon/internal/migration"
	"github.com/sooraj-zebu/falcon/internal/repository"
)

func NewEdge(configFile string) (*app.EdgeApp, error) {

	cfg, err := config.Load(configFile)
	if err != nil {
		return nil, err
	}

	logg := logger.New()

	if err := os.MkdirAll(cfg.Storage.DataDir, 0755); err != nil {
		return nil, err
	}

	// DB for sync system
	db, err := database.New(cfg.Storage.DataDir + "/edge.db")
	if err != nil {
		return nil, err
	}

	migrator := migration.New(db.DB)
	if err := migrator.Run(); err != nil {
		return nil, err
	}

	// repo for crash-safe queue
	syncRepo := repository.NewSyncQueueRepository(db)

	// grpc client
	client, err := client.NewEdgeClient(
		cfg.Core.Host,
		cfg.Core.GRPCPort,
	)
	if err != nil {
		return nil, err
	}

	edge := &app.EdgeApp{
		Config:   cfg,
		Logger:   logg,
		Client:   client,
		SyncRepo: syncRepo,
	}

	return edge, nil
}
