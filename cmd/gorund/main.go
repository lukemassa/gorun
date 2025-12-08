package main

import (
	"fmt"
	"os"

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
	server := server.NewServer(server.DefaultSock)
	switch os.Args[1] {
	case "run":
		server.Run()
	default:
		usage()
	}
}
