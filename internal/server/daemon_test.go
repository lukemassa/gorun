package server

import (
	"io"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"

	log "github.com/lukemassa/clilog"
	"github.com/stretchr/testify/assert"
)

type mockRunner struct {
	isStarted bool
}

func (m *mockRunner) Start(_ io.Writer) (int, error) {
	m.isStarted = true
	return 1234, nil
}

func (m *mockRunner) Alive(pid int) bool {
	return m.isStarted
}

func (m *mockRunner) Stop(pid int) error {
	m.isStarted = false
	return nil
}

func TestDaemon(t *testing.T) {
	log.Info("Testing daemon")

	runner := &mockRunner{}
	dir := t.TempDir()
	s := NewServer(dir)
	d := NewDaemon(s, runner)

	log.Info("Deamon is configured")
	assert.False(t, runner.isStarted)
	err := d.Stop()
	assert.ErrorContains(t, err, "no pid found")

	err = d.Start()
	assert.NoError(t, err)

	log.Info("Deamon started")

	assert.True(t, runner.isStarted)

	err = d.Start()
	assert.ErrorContains(t, err, "already running")

	err = d.Stop()
	assert.NoError(t, err)
	log.Info("Deamon stopped")
}

func waitForPid(t *testing.T, pid int) {
	t.Helper()

	var status syscall.WaitStatus
	for {
		_, err := syscall.Wait4(pid, &status, 0, nil) // BLOCKING
		if err == syscall.EINTR {
			continue
		}
		if err == syscall.ECHILD {
			return
		}
		if err != nil {
			t.Fatal(err)
		}
		return
	}
}

func TestOSProcessController(t *testing.T) {

	// Setup a process that writes to stdout then sleeps
	p := NewOSProcessController("sh", "-c", "echo hello && sleep 10")
	dir := t.TempDir()
	logPath := filepath.Join(dir, "out.log")
	f, err := os.Create(logPath)
	assert.NoError(t, err)
	defer f.Close()

	// Start the process
	pid, err := p.Start(f)
	assert.NoError(t, err)

	// Wait until the contents are written to the file
	assert.Eventually(t, func() bool {
		b, _ := os.ReadFile(logPath)
		return string(b) == "hello\n"
	}, time.Second, 10*time.Millisecond)

	// Expect it still to be running after this
	assert.True(t, p.Alive(pid))

	// Stop the process, which requires us to to all Wait on the pid so it doesn't become a zombie
	var wg sync.WaitGroup
	var stopError error
	wg.Go(func() {
		waitForPid(t, pid)
	})
	wg.Go(func() {
		stopError = p.Stop(pid)
	})
	wg.Wait()
	assert.NoError(t, stopError)

	assert.False(t, p.Alive(pid))

}
