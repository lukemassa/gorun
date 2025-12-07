package main

import (
	"log"
	"os"
	"syscall"

	"github.com/lukemassa/gorun/internal/rpc"
)

func main() {
	client := rpc.NewClient()

	env := os.Environ()

	initialCmd := os.Args[0]
	initialArgs := os.Args[1:]

	binary, err := client.GetBinary(initialCmd, env)
	// URL host is ignored — must be syntactically valid, but irrelevant.
	if err != nil {
		log.Fatal(err)
	}

	args := []string{binary}
	args = append(args, initialArgs...)

	log.Printf("Translated initial command %q to %q, passing additional args %v", initialCmd, binary, initialArgs)

	err = syscall.Exec(binary, args, env)
	if err != nil {
		log.Fatalf("exec failed: %v", err)
	}
	// Unreachable
}
