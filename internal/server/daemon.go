package server

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	log "github.com/lukemassa/clilog"
)

type Daemon struct {
	server *Server
}

func NewDaemon(s *Server) *Daemon {
	return &Daemon{
		server: s,
	}
}

func (d *Daemon) pidFile() string {
	return filepath.Join(d.server.cacheDir, "gorun.pid")
}

func (d *Daemon) logFile() string {
	// TODO: Do not overwrite the log file on every run
	return filepath.Join(d.server.cacheDir, "gorun.log")
}

func (d *Daemon) currentPid() (int, error) {
	content, err := os.ReadFile(d.pidFile())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, nil
		}
		return 0, err
	}
	pidAsString := strings.TrimSpace(string(content))
	pid, err := strconv.Atoi(pidAsString)
	if err != nil {
		return 0, err
	}
	return pid, nil
}

func (d *Daemon) savePid(pid int) error {
	file, err := os.Create(d.pidFile())
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = fmt.Fprintf(file, "%d", pid)
	return err
}

func pidAlive(pid int) bool {
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// POSIX: signal 0 checks existence
	return p.Signal(syscall.Signal(0)) == nil
}

func (d *Daemon) deletePid() error {
	return os.Remove(d.pidFile())
}

func (d *Daemon) Start() error {
	pid, err := d.currentPid()
	if err != nil {
		return err
	}
	if pid != 0 && pidAlive(pid) {
		return fmt.Errorf("daemon already running pid %d", pid)
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}

	gorunLog, err := os.Create(d.logFile())
	if err != nil {
		return err
	}
	defer gorunLog.Close()

	cmd := exec.Command(exe, "run")

	cmd.Stdout = gorunLog
	cmd.Stderr = gorunLog

	err = cmd.Start()
	if err != nil {
		return err
	}
	err = d.savePid(cmd.Process.Pid)
	if err != nil {
		return err
	}
	log.Infof("Started process %d", cmd.Process.Pid)
	return nil
}

func (d *Daemon) kill(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	// TODO: Can I check here to make sure this at least vaguely looks like the command we want it to be?

	err = process.Signal(syscall.SIGTERM)
	if err != nil {
		return err
	}
	for range 50 {
		if !pidAlive(pid) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	log.Warnf("Could not kill %d with with term, sending kill", pid)

	err = process.Signal(syscall.SIGKILL)
	if err != nil {
		return err
	}
	return nil
}

func (d *Daemon) Stop() error {
	pid, err := d.currentPid()
	if err != nil {
		return err
	}
	if pid == 0 {
		return errors.New("no pid found")
	}

	err = d.kill(pid)
	if err != nil {
		return err
	}
	err = d.deletePid()
	if err != nil {
		return err
	}
	log.Infof("Stopped %d", pid)
	return nil
}
