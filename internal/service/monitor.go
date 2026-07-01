package service

import (
	"log"
	"time"
)

func StartMonitor(edgeService *EdgeService, logger *log.Logger) {

    logger.Println("===== Offline monitor started =====")

    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    for range ticker.C {

        logger.Println("===== Checking edge status =====")

        err := edgeService.MarkOffline(1)
        if err != nil {
            logger.Println("Offline monitor:", err)
            continue
        }

        count, err := edgeService.OnlineCount()
        if err != nil {
            logger.Println(err)
        }

        logger.Println("Online edges:", count)
    }
}

/*func StartMonitor(edgeService *EdgeService, logger *log.Logger) {

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {

		err := edgeService.MarkOffline(1)
		if err != nil {
			logger.Println("Offline monitor:", err)
			continue
		}

		count, _ := edgeService.OnlineCount()

		logger.Println("Online edges:", count)
	}
}
*/
