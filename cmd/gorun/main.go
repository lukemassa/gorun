package main

import (
	"os"
	"syscall"

	log "github.com/lukemassa/clilog"
	"github.com/lukemassa/gorun/internal/client"
	"github.com/lukemassa/gorun/internal/config"
)

func main() {
	sock := os.Getenv("GORUN_SOCKET")
	if sock == "" {
		sock = config.DefaultSock()
	}

	if os.Getenv("GORUN_DEBUG") != "" {
		log.SetLogLevel(log.LevelDebug)
	}
	client := client.NewClient(sock)

	env := os.Environ()
	if len(os.Args) < 2 {
		log.Fatal("Expect argument for package")
	}
	mainPackage := os.Args[1]
	mainArgs := os.Args[2:]

	executable, err := client.GetCommand(mainPackage, env)
	// URL host is ignored — must be syntactically valid, but irrelevant.
	if err != nil {
		log.Fatal(err)
	}

	args := []string{executable}
	args = append(args, mainArgs...)

	log.Debugf("Compiled context for %q to %q, passing additional args %v", mainPackage, executable, mainArgs)

	err = syscall.Exec(executable, args, env)
	if err != nil {
		log.Fatalf("exec failed: %v", err)
	}
	// Unreachable
}
