package main

import (
	"github.com/lukemassa/gorun/internal/rpc"
)

func main() {

	server := rpc.Server{}

	server.Run()
}
