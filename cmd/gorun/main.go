package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	log "github.com/lukemassa/clilog"
	"github.com/lukemassa/gorun/internal/client"
	"github.com/lukemassa/gorun/internal/config"
)

func promptYesNo(prompt string) bool {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("%s [y/n]: ", prompt)

		line, err := reader.ReadString('\n')
		if err != nil {
			log.Warnf("Unexpected error parsing stdin: %v", err)
			return false
		}

		switch strings.ToLower(strings.TrimSpace(line)) {
		case "y", "yes":
			return true
		case "n", "no":
			return false
		default:
			fmt.Println("Please answer y or n.")
		}
	}
}

func getExecutable(c *client.Client, mainPackage string, env []string) string {
	executable, err := c.GetCommand(mainPackage, env)
	if err != nil {
		if !errors.Is(err, syscall.ECONNREFUSED) {
			log.Fatal(err)
		}
		log.Warn("Gorun appears to not be running")
		if !promptYesNo("Start up gorund?") {
			log.Fatal("Exiting")
		}
		cmd := exec.Command("gorund", "start")
		err := cmd.Run()
		if err != nil {
			log.Fatal(err)
		}
		time.Sleep(100 * time.Millisecond)
		log.Warn("Started up gorun")
		executable, err = c.GetCommand(mainPackage, env)
		if err != nil {
			log.Fatal(err)
		}

	}
	return executable
}

func main() {
	workingDir := os.Getenv("GORUN_WORKING_DIR")
	if workingDir == "" {
		workingDir = config.WorkingDir()
	}

	if os.Getenv("GORUN_DEBUG") != "" {
		log.SetLogLevel(log.LevelDebug)
	}

	verb := "run"
	if os.Getenv("GORUN_DELETE") != "" {
		verb = "delete"
	}
	client := client.NewClient(workingDir)

	env := os.Environ()
	if len(os.Args) < 2 {
		log.Fatal("Expect argument for package")
	}
	mainPackage := os.Args[1]
	mainArgs := os.Args[2:]

	switch verb {
	case "run":

		executable := getExecutable(client, mainPackage, env)
		args := []string{executable}
		args = append(args, mainArgs...)

		log.Debugf("Compiled context for %q to %q, passing additional args %v", mainPackage, executable, mainArgs)

		err := syscall.Exec(executable, args, env)
		if err != nil {
			log.Fatalf("exec failed: %v", err)
		}
		// Unreachable
	case "delete":
		err := client.DeleteCommand(mainPackage, env)
		if err != nil {
			log.Fatalf("delete failed: %v", err)
		}
	}
}
