package main

import (
	"github.com/lukemassa/gorun/internal/server"
)

func main() {

	server := server.NewServer(server.DefaultSock)

	server.Run()
}
