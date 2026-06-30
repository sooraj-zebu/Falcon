package logger

import (
	"log"
	"os"
)

func New() *log.Logger {

	return log.New(
		os.Stdout,
		"[Falcon] ",
		log.LstdFlags|log.Lshortfile,
	)

}
