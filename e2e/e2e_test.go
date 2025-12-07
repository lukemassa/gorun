package e2e

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var cliPath string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "mycli-e2e-*")
	if err != nil {
		panic(err)
	}
	// you must clean it up manually afterwards
	defer os.RemoveAll(dir)

	cliPath = filepath.Join(dir, "gorun-test-binary")
	cmd := exec.Command("go", "build", "-o", cliPath, "github.com/lukemassa/gorun/cmd/gorun")
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0") // optional but hermetic
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to build test binary: %s\n%s", err, out)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

type RunResult struct {
	Stdout string
	Stderr string
	Code   int
}

func runCLI(args ...string) (RunResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, cliPath, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	code := 0
	if err != nil {
		ee, ok := err.(*exec.ExitError)
		if !ok {
			// non-process error: timeout, context canceled, etc.
			return RunResult{}, fmt.Errorf("failed to run with args: '%v': %v", args, err)
		}
		code = ee.ExitCode()
	}

	return RunResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
		Code:   code,
	}, nil
}

func TestRun(t *testing.T) {
	result, err := runCLI("hello")
	assert.NoError(t, err)

	assert.Equal(t, "hello\n", result.Stdout)
	assert.Equal(t, 0, result.Code)
	assert.Contains(t, result.Stderr, "Translated initial command")
}
