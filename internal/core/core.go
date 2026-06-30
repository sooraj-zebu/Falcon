package core

import (
	"time"

	"github.com/sooraj-zebu/falcon/internal/logger"
)

func Start() {
	logger.Logger.Println("Starting Core Services...")

	time.Sleep(1 * time.Second)

	logger.Logger.Println("Core Services Started")
}
