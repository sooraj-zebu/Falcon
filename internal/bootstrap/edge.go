package bootstrap

import (
	"github.com/sooraj-zebu/falcon/internal/app"
	"github.com/sooraj-zebu/falcon/internal/config"
	"github.com/sooraj-zebu/falcon/internal/logger"
)

func NewEdge(configFile string) (*app.EdgeApp, error) {

	cfg, err := config.Load(configFile)

	if err != nil {
		return nil, err
	}

	logg := logger.New()

	edge := app.NewEdge(cfg, logg)

	return edge, nil

}
