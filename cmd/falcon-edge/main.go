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
		"configs/edge.yaml",
		"configuration file",
	)

	flag.Parse()

	edge, err := bootstrap.NewEdge(*configFile)

	if err != nil {
		log.Fatal(err)
	}

	edge.Start()

	sig := make(chan os.Signal, 1)

	signal.Notify(
		sig,
		os.Interrupt,
		syscall.SIGTERM,
	)

	<-sig

	edge.Stop()

}
