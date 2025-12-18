package server

import (
	"errors"
	"fmt"
	"io"
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
	server            *Server
	processController ProcessController
}

type ProcessController interface {
	Start(logFile io.Writer) (pid int, err error)
	Stop(pid int) error
	Alive(pid int) bool
}

type OSProcessController struct {
	cmd  string
	args []string
}

func NewOSProcessController(cmd string, args ...string) OSProcessController {
	return OSProcessController{
		cmd:  cmd,
		args: args,
	}
}

func (o OSProcessController) Start(log io.Writer) (int, error) {
	cmd := exec.Command(o.cmd, o.args...)
	cmd.Stdout = log
	cmd.Stderr = log
	err := cmd.Start()
	if err != nil {
		return 0, err
	}
	return cmd.Process.Pid, nil
}

func (o OSProcessController) Stop(pid int) error {
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
		if !o.Alive(pid) {
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

func (o OSProcessController) Alive(pid int) bool {
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// POSIX: signal 0 checks existence
	return p.Signal(syscall.Signal(0)) == nil
}

func NewDaemon(s *Server, processController ProcessController) *Daemon {
	return &Daemon{
		server:            s,
		processController: processController,
	}
}

func (d *Daemon) pidFile() string {
	return filepath.Join(d.server.workingDir, "gorun.pid")
}

func (d *Daemon) logFile() string {
	// TODO: Do not overwrite the log file on every run
	return filepath.Join(d.server.workingDir, "gorun.log")
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

func (d *Daemon) deletePid() error {
	return os.Remove(d.pidFile())
}

func (d *Daemon) Start() error {
	pid, err := d.currentPid()
	if err != nil {
		return err
	}
	if pid != 0 && d.processController.Alive(pid) {
		return fmt.Errorf("daemon already running pid %d", pid)
	}

	gorunLog, err := os.Create(d.logFile())
	if err != nil {
		return err
	}
	defer gorunLog.Close()

	pid, err = d.processController.Start(gorunLog)
	if err != nil {
		return err
	}
	err = d.savePid(pid)
	if err != nil {
		return err
	}
	log.Infof("Started process %d", pid)
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

	err = d.processController.Stop(pid)
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
