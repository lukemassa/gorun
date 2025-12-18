package main

import (
	"fmt"
	"log"
	"os"

	"github.com/lukemassa/gorun/internal/config"
	"github.com/lukemassa/gorun/internal/server"
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: goruncd start|stop|run\n")
	os.Exit(1)
}

func main() {
	if len(os.Args) != 2 {
		usage()
	}
	s := server.NewServer(config.WorkingDir())
	cmd := os.Args[1]
	if cmd == "run" {
		s.Run()
		return
	}
	daemon := server.NewDaemon(s)
	var err error
	switch cmd {
	case "start":
		err = daemon.Start()
	case "stop":
		err = daemon.Stop()
	default:
		usage()
	}
	if err != nil {
		log.Fatal(err)
	}
}
