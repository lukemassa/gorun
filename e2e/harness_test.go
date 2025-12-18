package e2e

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/lukemassa/gorun/internal/server"
)

var (
	cliPath         string
	gorunWorkingDir string
)

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "gorun-test-e2e-*")
	if err != nil {
		panic(err)
	}
	// you must clean it up manually afterwards
	defer os.RemoveAll(dir)

	cliPath = filepath.Join(dir, "cmd")
	cmd := exec.Command("go", "build", "-o", cliPath, "github.com/lukemassa/gorun/cmd/gorun")
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0") // optional but hermetic
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to build test binary: %s\n%s", err, out)
		os.Exit(1)
	}
	gorunWorkingDir = dir

	server := server.NewServer(dir)
	cancel, err := server.Start()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to startz test server: %s\n%s", err, out)
		os.Exit(1)
	}

	estatus := m.Run()
	cancel()
	os.Exit(estatus)
}

type RunResult struct {
	Stdout string
	Stderr string
	Code   int
}

// runCLI runs the CLI with the args, in a directory with the files from fsys
// If the command times out, the code is set to -1
func runCLI(t *testing.T, workingDir string, args ...string) RunResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, cliPath, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Dir = workingDir
	cmd.Env = append(cmd.Env, fmt.Sprintf("GORUN_WORKING_DIR=%s", gorunWorkingDir))
	cmd.Env = append(cmd.Env, fmt.Sprintf("PWD=%s", workingDir))
	cmd.Env = append(cmd.Env, "GORUN_DEBUG=1")

	err := cmd.Run()

	code := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else if errors.Is(err, context.DeadlineExceeded) {
			code = -1
		} else {
			// test harness failure
			t.Fatalf("failed to run with args %v: %v", args, err)
		}
	}

	return RunResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
		Code:   code,
	}
}

func writeFS(t *testing.T, src fs.FS, dst string) {
	t.Helper()
	if src == nil {
		return
	}

	err := fs.WalkDir(src, ".", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		full := filepath.Join(dst, path)

		if d.IsDir() {
			return os.MkdirAll(full, 0o755)
		}

		data, err := fs.ReadFile(src, path)
		if err != nil {
			return err
		}

		return os.WriteFile(full, data, 0o644)
	})

	if err != nil {
		t.Fatalf("WriteFS failed populating %q: %v", dst, err)
	}
}
