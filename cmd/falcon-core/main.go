package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/sooraj-zebu/falcon/internal/bootstrap"
)

func main() {

	configFile := flag.String(
		"config",
		"configs/core.yaml",
		"Configuration file",
	)

	flag.Parse()

	core, err := bootstrap.NewCore(*configFile)
	if err != nil {
		log.Fatal(err)
	}

	core.Start()

	sig := make(chan os.Signal, 1)

	signal.Notify(sig,
		os.Interrupt,
		syscall.SIGTERM,
	)

	<-sig

	core.Logger.Println("Shutdown complete.")
}
