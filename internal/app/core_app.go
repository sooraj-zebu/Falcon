package app

import (
	"log"

	"github.com/sooraj-zebu/falcon/internal/config"
	"github.com/sooraj-zebu/falcon/internal/database"
	"github.com/sooraj-zebu/falcon/internal/service"
)

type CoreApp struct {
	Config *config.Config
	Logger *log.Logger

	DB *database.Database

	EdgeService *service.EdgeService
}

func NewCore(cfg *config.Config, logger *log.Logger) *CoreApp {

	return &CoreApp{
		Config: cfg,
		Logger: logger,
	}
}

func (a *CoreApp) Start() {

	a.Logger.Println("===================================")
	a.Logger.Println(a.Config.App.Name)
	a.Logger.Println("Version:", a.Config.App.Version)
	a.Logger.Println("HTTP Port:", a.Config.Server.HTTPPort)
	a.Logger.Println("gRPC Port:", a.Config.Server.GRPCPort)
	a.Logger.Println("Database Connected")
	a.Logger.Println("Edge Service Initialized")
	a.Logger.Println("===================================")

	a.Logger.Println("Falcon Core Started")
}
