package app

import (
	"log"

	"github.com/sooraj-zebu/falcon/internal/client"
	"github.com/sooraj-zebu/falcon/internal/config"
)

type EdgeApp struct {
	Config *config.Config
	Logger *log.Logger
	Client *client.EdgeClient
}

func (a *EdgeApp) Start() {

	a.Logger.Println("Starting Edge:", a.Config.Edge.Name)

	// Create gRPC client
	c, err := client.NewEdgeClient(
		a.Config.Core.Host,
		a.Config.Core.GRPCPort,
	)

	if err != nil {
		a.Logger.Println("Failed to connect to Core:", err)
		return
	}

	a.Client = c

	// REGISTER EDGE (THIS IS MISSING IN YOUR RUN)
	edgeID, err := a.Client.Register(
		a.Config.Edge.Name,
		a.Config.App.Version,
		"127.0.0.1",
	)

	if err != nil {
		a.Logger.Println("Edge registration failed:", err)
		return
	}

	a.Logger.Println("Edge registered successfully")
	a.Logger.Println("Edge ID:", edgeID)
}
